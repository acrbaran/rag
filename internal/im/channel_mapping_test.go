package im

import (
	"testing"

	"github.com/acrbaran/rag/internal/types"
)

// Files uploaded through an IM channel are tagged with a knowledge Channel.
func TestIMPlatformToChannel(t *testing.T) {
	cases := map[string]string{
		"wechat": types.ChannelWechat,
		"slack":    types.ChannelSlack,
		// Platforms without a dedicated channel fall back to the generic one.
		"telegram": types.ChannelIM,
		"":         types.ChannelIM,
	}

	for platform, want := range cases {
		if got := imPlatformToChannel(platform); got != want {
			t.Errorf("imPlatformToChannel(%q) = %q, want %q", platform, got, want)
		}
	}
}
