package im

import "context"

// StopCommand implements /stop.
// It cancels the in-flight QA request for the current user+chat, allowing the
// user to abort a long-running ReAct reasoning chain without waiting for it to
// complete. If no request is in progress the command simply acknowledges.
type StopCommand struct{}

func newStopCommand() *StopCommand { return &StopCommand{} }

func (c *StopCommand) Name() string { return "stop" }
func (c *StopCommand) Description(ctx context.Context) string {
	return imText(ctx, "Stop the current response", "Geçerli yanıtı durdur")
}

func (c *StopCommand) Execute(ctx context.Context, _ *CommandContext, _ []string) (*CommandResult, error) {
	return &CommandResult{
		Content: imText(ctx, "✅ Requested to stop the current response.", "✅ Geçerli yanıtın durdurulması istendi."),
		Action:  ActionStop,
	}, nil
}
