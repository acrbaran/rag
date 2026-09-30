package im

import "context"

// ClearCommand implements /clear.
// It soft-deletes the current ChannelSession and clears the LLM context so
// the next message starts a completely fresh conversation.
type ClearCommand struct{}

func newClearCommand() *ClearCommand { return &ClearCommand{} }

func (c *ClearCommand) Name() string { return "clear" }
func (c *ClearCommand) Description(ctx context.Context) string {
	return imText(ctx, "Clear conversation history; the next message starts a new session", "Konuşma geçmişini temizle; sonraki mesaj yeni bir oturum başlatır")
}

func (c *ClearCommand) Execute(ctx context.Context, _ *CommandContext, _ []string) (*CommandResult, error) {
	return &CommandResult{
		Content: imText(ctx, "✅ Conversation cleared. Your next message will start a new session.", "✅ Konuşma temizlendi. Sonraki mesajınız yeni bir oturum başlatacak."),
		Action:  ActionClear,
	}, nil
}
