package im

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// HelpCommand implements /help [command].
type HelpCommand struct {
	registry *CommandRegistry
}

func newHelpCommand(registry *CommandRegistry) *HelpCommand {
	return &HelpCommand{registry: registry}
}

func (c *HelpCommand) Name() string { return "help" }
func (c *HelpCommand) Description(ctx context.Context) string {
	return imText(ctx, "List available commands or show help for one command", "Kullanılabilir komutları listele veya bir komutun yardımını göster")
}

func (c *HelpCommand) Execute(ctx context.Context, _ *CommandContext, args []string) (*CommandResult, error) {
	// /help <command> — show detailed usage for a specific command
	if len(args) > 0 {
		name := strings.ToLower(args[0])
		cmd, _, ok := c.registry.Parse("/" + name)
		if !ok {
			return &CommandResult{
				Content: fmt.Sprintf(imText(ctx, "Unknown command %s. Send /help to see all commands.", "Bilinmeyen komut: %s. Tüm komutları görmek için /help gönderin."), args[0]),
			}, nil
		}
		return &CommandResult{
			Content: fmt.Sprintf("**/%s** — %s", cmd.Name(), cmd.Description(ctx)),
		}, nil
	}

	// /help — list all commands sorted by name
	cmds := c.registry.All()
	sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name() < cmds[j].Name() })

	var sb strings.Builder
	sb.WriteString(imText(ctx, "**Available commands**\n\n", "**Kullanılabilir komutlar**\n\n"))
	for _, cmd := range cmds {
		sb.WriteString(fmt.Sprintf("· `/%s` — %s\n", cmd.Name(), cmd.Description(ctx)))
	}
	sb.WriteString(imText(ctx, "\nSend /help <command> for details", "\nAyrıntılar için /help <komut> gönderin"))
	return &CommandResult{Content: sb.String()}, nil
}
