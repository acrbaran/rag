package im

import (
	"context"
	"fmt"
	"strings"

	"github.com/acrbaran/rag/internal/types/interfaces"
)

// InfoCommand implements /info.
// It shows the bound agent's profile and capabilities so IM users can
// understand what the bot can do without leaving the chat.
type InfoCommand struct {
	kbService interfaces.KnowledgeBaseService
}

func newInfoCommand(kbService interfaces.KnowledgeBaseService) *InfoCommand {
	return &InfoCommand{kbService: kbService}
}

func (c *InfoCommand) Name() string { return "info" }
func (c *InfoCommand) Description(ctx context.Context) string {
	return imText(ctx, "Show this agent's information and capabilities", "Bu ajanın bilgilerini ve yeteneklerini göster")
}

func (c *InfoCommand) Execute(ctx context.Context, cmdCtx *CommandContext, _ []string) (*CommandResult, error) {
	var sb strings.Builder

	// ── Header ──
	name := cmdCtx.AgentName
	if name == "" {
		name = imText(ctx, "Unnamed agent", "Adsız ajan")
	}
	sb.WriteString(fmt.Sprintf("🤖 **%s**\n", name))
	if cmdCtx.CustomAgent != nil && cmdCtx.CustomAgent.Description != "" {
		sb.WriteString(fmt.Sprintf("> %s\n", cmdCtx.CustomAgent.Description))
	}

	if cmdCtx.CustomAgent == nil {
		sb.WriteString(imText(ctx, "\nNo agent is bound. Send /help to see available commands.", "\nBağlı ajan yok. Kullanılabilir komutlar için /help gönderin."))
		return &CommandResult{Content: sb.String()}, nil
	}

	cfg := cmdCtx.CustomAgent.Config

	// ── Mode ──
	if cmdCtx.CustomAgent.IsAgentMode() {
		sb.WriteString(imText(ctx, "\n🧠 **Agent mode**\n", "\n🧠 **Ajan modu**\n"))
		sb.WriteString(imText(ctx, "Multi-step reasoning and tool calls (ReAct)\n", "Çok adımlı akıl yürütme ve araç çağrıları (ReAct)\n"))
	} else {
		sb.WriteString(imText(ctx, "\n🧠 **Agent mode**\n", "\n🧠 **Ajan modu**\n"))
		sb.WriteString(imText(ctx, "Direct answers from knowledge base retrieval (RAG)\n", "Bilgi bankası aramasına dayalı doğrudan yanıtlar (RAG)\n"))
	}

	// ── Knowledge bases ──
	// KBSelectionMode: "all" uses every KB under the tenant (IDs list is empty),
	// "selected" uses the explicit KnowledgeBases list, "none"/empty means disabled.
	sb.WriteString(imText(ctx, "\n📚 **Knowledge bases**\n", "\n📚 **Bilgi bankaları**\n"))
	if cfg.KBSelectionMode == "all" {
		kbs, err := c.kbService.ListKnowledgeBasesByTenantID(ctx, cmdCtx.TenantID)
		if err == nil && len(kbs) > 0 {
			for _, kb := range kbs {
				sb.WriteString(fmt.Sprintf("  · %s\n", kb.Name))
			}
			sb.WriteString(fmt.Sprintf(imText(ctx, "  %d total (all enabled)\n", "  Toplam %d (tümü etkin)\n"), len(kbs)))
		} else {
			sb.WriteString(imText(ctx, "  All enabled\n", "  Tümü etkin\n"))
		}
	} else if len(cfg.KnowledgeBases) > 0 {
		kbs, err := c.kbService.ListKnowledgeBasesByTenantID(ctx, cmdCtx.TenantID)
		if err == nil {
			nameMap := make(map[string]string, len(kbs))
			for _, kb := range kbs {
				nameMap[kb.ID] = kb.Name
			}
			for _, id := range cfg.KnowledgeBases {
				label := id
				if n, ok := nameMap[id]; ok {
					label = n
				}
				sb.WriteString(fmt.Sprintf("  · %s\n", label))
			}
		} else {
			sb.WriteString(fmt.Sprintf(imText(ctx, "  %d selected\n", "  %d seçili\n"), len(cfg.KnowledgeBases)))
		}
	} else {
		sb.WriteString(imText(ctx, "  Not configured\n", "  Yapılandırılmadı\n"))
	}

	// ── Skills ──
	sb.WriteString("\n⚡ **Skills**\n")
	if cfg.SkillsSelectionMode == "all" {
		sb.WriteString(imText(ctx, "  All enabled\n", "  Tümü etkin\n"))
	} else if cfg.SkillsSelectionMode == "selected" && len(cfg.SelectedSkills) > 0 {
		for _, s := range cfg.SelectedSkills {
			sb.WriteString(fmt.Sprintf("  · %s\n", s))
		}
	} else {
		sb.WriteString(imText(ctx, "  Not configured\n", "  Yapılandırılmadı\n"))
	}

	// ── MCP ──
	sb.WriteString(imText(ctx, "\n🔌 **MCP services**\n", "\n🔌 **MCP hizmetleri**\n"))
	if cfg.MCPSelectionMode == "all" {
		sb.WriteString(imText(ctx, "  All connected\n", "  Tümü bağlı\n"))
	} else if cfg.MCPSelectionMode == "selected" && len(cfg.MCPServices) > 0 {
		sb.WriteString(fmt.Sprintf(imText(ctx, "  %d services connected\n", "  %d hizmet bağlı\n"), len(cfg.MCPServices)))
	} else {
		sb.WriteString(imText(ctx, "  Not configured\n", "  Yapılandırılmadı\n"))
	}

	// ── Web search ──
	sb.WriteString(imText(ctx, "\n🌐 **Web search**\n", "\n🌐 **Web araması**\n"))
	if cfg.WebSearchEnabled {
		sb.WriteString(imText(ctx, "  Enabled\n", "  Etkin\n"))
	} else {
		sb.WriteString(imText(ctx, "  Disabled\n", "  Devre dışı\n"))
	}

	// ── Footer ──
	outputLabel := imText(ctx, "Streaming", "Akış halinde")
	if cmdCtx.ChannelOutputMode == "full" {
		outputLabel = imText(ctx, "Full response", "Tam yanıt")
	}
	sb.WriteString(fmt.Sprintf(imText(ctx, "\n⚙️ **Output mode**\n  %s\n", "\n⚙️ **Çıktı modu**\n  %s\n"), outputLabel))
	sb.WriteString(imText(ctx, "\n---\nSend /help to see all commands", "\n---\nTüm komutları görmek için /help gönderin"))

	return &CommandResult{Content: sb.String()}, nil
}
