package handler

import (
	"strings"
	"testing"
)

// The channel-creation endpoint rejects any platform without a registered
// adapter factory, so this set must track the factories wired in the container.
func TestValidIMPlatforms(t *testing.T) {
	want := []string{"slack", "telegram", "wechat", "qqbot"}
	for _, platform := range want {
		if !validIMPlatforms[platform] {
			t.Errorf("platform %q is not accepted", platform)
		}
	}
	if validIMPlatforms["nonsense"] {
		t.Error("unknown platform is accepted")
	}
	for _, platform := range []string{"wecom", "feishu", "lark", "dingtalk", "mattermost", "yunzhijia"} {
		if validIMPlatforms[platform] {
			t.Errorf("removed platform %q is accepted", platform)
		}
	}
}

// The 400 message is derived from validIMPlatforms; it must not drift as
// platforms are added.
func TestInvalidIMPlatformError_ListsEveryPlatform(t *testing.T) {
	for platform := range validIMPlatforms {
		if !strings.Contains(invalidIMPlatformError, "'"+platform+"'") {
			t.Errorf("error message omits %q: %s", platform, invalidIMPlatformError)
		}
	}
}
