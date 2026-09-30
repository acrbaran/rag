package agent

import (
	"context"
	"testing"

	agenttools "github.com/acrbaran/rag/internal/agent/tools"
	"github.com/acrbaran/rag/internal/types"
)

func TestFormatToolHintUsesRequestLanguage(t *testing.T) {
	for _, tt := range []struct {
		locale string
		want   string
	}{
		{"en-US", `Search web("weather")`},
		{"tr-TR", `Web'de ara("weather")`},
	} {
		ctx := context.WithValue(context.Background(), types.LanguageContextKey, tt.locale)
		if got := formatToolHint(ctx, agenttools.ToolWebSearch, map[string]any{"query": "weather"}); got != tt.want {
			t.Errorf("locale %s: got %q, want %q", tt.locale, got, tt.want)
		}
	}
}
