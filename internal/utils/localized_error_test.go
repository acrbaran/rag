package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestUserFacingConnectionErrorsUseRequestedLanguage(t *testing.T) {
	for _, tc := range []struct {
		language string
		want     string
	}{
		{"tr-TR", "Bağlantı reddedildi"},
		{"en-US", "Connection refused"},
	} {
		message := SanitizeStorageConnectivityError(errors.New("dial 10.0.0.4:9000: connection refused"), tc.language)
		if !strings.Contains(message, tc.want) || strings.Contains(message, "10.0.0.4") {
			t.Fatalf("unexpected %s storage error: %q", tc.language, message)
		}
	}
	for _, tc := range []struct {
		language string
		want     string
	}{
		{"tr-TR", "güvenlik denetiminden geçemedi"},
		{"en-US", "failed the security check"},
	} {
		message := FormatSSRFError("URL", "example.com", ErrSSRFHostNotWhitelisted, tc.language)
		if !strings.Contains(message, tc.want) || !strings.Contains(message, "SSRF_WHITELIST_EXTRA") {
			t.Fatalf("unexpected %s SSRF error: %q", tc.language, message)
		}
	}
}
