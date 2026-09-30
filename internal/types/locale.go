package types

import (
	"context"
	"strings"
)

var supportedLocales = map[string]struct{}{
	"en-US": {},
	"tr-TR": {},
}

// NormalizeSupportedLocale returns a trimmed, supported locale tag or an empty
// string when the value is blank or unsupported.
func NormalizeSupportedLocale(locale string) string {
	locale = strings.TrimSpace(locale)
	if _, ok := supportedLocales[locale]; ok {
		return locale
	}
	return ""
}

// LocalizedText selects user-facing text for the request's supported language.
func LocalizedText(ctx context.Context, turkish, english string) string {
	if LanguageFromContextOrDefault(ctx) == "en-US" {
		return english
	}
	return turkish
}
