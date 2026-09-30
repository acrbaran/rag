package types

import "strings"

// MaxBrowserSearchInstructionsLength is the stored search-preference cap.
const MaxBrowserSearchInstructionsLength = 4000

// DefaultBrowserSearchInstructions is shown in settings and sent to the browser
// tool when the user has not saved a custom preference.
const DefaultBrowserSearchInstructions = `Default search engine: Google.
Search URL: https://www.google.com/search?q={query}`

// EffectiveBrowserSearchInstructions returns the saved preference, or the default.
func (p UserPreferences) EffectiveBrowserSearchInstructions() string {
	if p.BrowserSearchInstructions != nil {
		if value := strings.TrimSpace(*p.BrowserSearchInstructions); value != "" {
			return value
		}
	}
	return DefaultBrowserSearchInstructions
}
