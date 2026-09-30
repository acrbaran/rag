package types

import (
	"context"
	"strings"
	"time"
)

// PromptPlaceholder represents a placeholder that can be used in prompt templates
type PromptPlaceholder struct {
	// Name is the placeholder name (without braces), e.g., "query"
	Name string `json:"name"`
	// Label is a short label for the placeholder
	Label string `json:"label"`
	// Description explains what this placeholder represents
	Description string `json:"description"`
}

// PromptFieldType represents the type of prompt field
type PromptFieldType string

const (
	// PromptFieldSystemPrompt is for system prompts (normal mode)
	PromptFieldSystemPrompt PromptFieldType = "system_prompt"
	// PromptFieldAgentSystemPrompt is for agent mode system prompts
	PromptFieldAgentSystemPrompt PromptFieldType = "agent_system_prompt"
	// PromptFieldContextTemplate is for context templates
	PromptFieldContextTemplate PromptFieldType = "context_template"
	// PromptFieldRewriteSystemPrompt is for rewrite system prompts
	PromptFieldRewriteSystemPrompt PromptFieldType = "rewrite_system_prompt"
	// PromptFieldRewritePrompt is for rewrite user prompts
	PromptFieldRewritePrompt PromptFieldType = "rewrite_prompt"
	// PromptFieldFallbackPrompt is for fallback prompts
	PromptFieldFallbackPrompt PromptFieldType = "fallback_prompt"
)

// All available placeholders in the system
var (
	// Common placeholders
	PlaceholderQuery = PromptPlaceholder{
		Name:        "query",
		Label:       "User question",
		Description: "The user's current question or query",
	}

	PlaceholderContexts = PromptPlaceholder{
		Name:        "contexts",
		Label:       "Retrieved content",
		Description: "Relevant content retrieved from the knowledge base",
	}

	PlaceholderCurrentTime = PromptPlaceholder{
		Name:        "current_time",
		Label:       "Current date",
		Description: "Current date in ISO format (2006-01-02)",
	}

	PlaceholderCurrentWeek = PromptPlaceholder{
		Name:        "current_week",
		Label:       "Current weekday",
		Description: "Current day of the week (for example, Monday)",
	}

	// Rewrite prompt placeholders
	PlaceholderConversation = PromptPlaceholder{
		Name:        "conversation",
		Label:       "Conversation history",
		Description: "Formatted conversation history for follow-up questions",
	}

	PlaceholderYesterday = PromptPlaceholder{
		Name:        "yesterday",
		Label:       "Yesterday's date",
		Description: "Yesterday's date in ISO format (2006-01-02)",
	}

	PlaceholderAnswer = PromptPlaceholder{
		Name:        "answer",
		Label:       "Assistant answer",
		Description: "The assistant's answer for formatting conversation history",
	}

	// Agent mode specific placeholders
	PlaceholderKnowledgeBases = PromptPlaceholder{
		Name:        "knowledge_bases",
		Label:       "Knowledge bases",
		Description: "Formatted knowledge base list with names, descriptions, and document counts",
	}

	PlaceholderWebSearchStatus = PromptPlaceholder{
		Name:        "web_search_status",
		Label:       "Web search status",
		Description: "Whether the web search tool is enabled or disabled",
	}

	PlaceholderLanguage = PromptPlaceholder{
		Name:        "language",
		Label:       "User language",
		Description: "User interface language preference for the assistant's answers",
	}
)

// LocalizePlaceholders translates the display metadata without changing placeholder names.
func LocalizePlaceholders(ctx context.Context, placeholders []PromptPlaceholder) []PromptPlaceholder {
	if LanguageFromContextOrDefault(ctx) == "en-US" {
		return placeholders
	}
	translated := make([]PromptPlaceholder, len(placeholders))
	for i, placeholder := range placeholders {
		translated[i] = placeholder
		switch placeholder.Name {
		case "query":
			translated[i].Label, translated[i].Description = "Kullanıcı sorusu", "Kullanıcının mevcut sorusu veya sorgusu"
		case "contexts":
			translated[i].Label, translated[i].Description = "Getirilen içerik", "Bilgi tabanından getirilen ilgili içerik"
		case "current_time":
			translated[i].Label, translated[i].Description = "Geçerli tarih", "ISO biçiminde geçerli tarih (2006-01-02)"
		case "current_week":
			translated[i].Label, translated[i].Description = "Haftanın günü", "Haftanın geçerli günü (örneğin Pazartesi)"
		case "conversation":
			translated[i].Label, translated[i].Description = "Sohbet geçmişi", "Takip soruları için biçimlendirilmiş sohbet geçmişi"
		case "yesterday":
			translated[i].Label, translated[i].Description = "Dünün tarihi", "ISO biçiminde dünün tarihi (2006-01-02)"
		case "answer":
			translated[i].Label, translated[i].Description = "Asistan yanıtı", "Sohbet geçmişini biçimlendirmek için asistanın yanıtı"
		case "knowledge_bases":
			translated[i].Label, translated[i].Description = "Bilgi tabanları", "Adlar, açıklamalar ve belge sayılarıyla biçimlendirilmiş bilgi tabanı listesi"
		case "web_search_status":
			translated[i].Label, translated[i].Description = "Web arama durumu", "Web arama aracının etkin veya devre dışı olma durumu"
		case "language":
			translated[i].Label, translated[i].Description = "Kullanıcı dili", "Asistan yanıtları için kullanıcı arayüzü dil tercihi"
		}
	}
	return translated
}

// PlaceholdersByField returns the available placeholders for a specific prompt field type
func PlaceholdersByField(fieldType PromptFieldType) []PromptPlaceholder {
	switch fieldType {
	case PromptFieldSystemPrompt:
		// Normal mode system prompt
		return []PromptPlaceholder{
			PlaceholderQuery,
			PlaceholderContexts,
			PlaceholderCurrentTime,
			PlaceholderCurrentWeek,
			PlaceholderLanguage,
		}
	case PromptFieldAgentSystemPrompt:
		// Agent mode system prompt
		return []PromptPlaceholder{
			PlaceholderKnowledgeBases,
			PlaceholderWebSearchStatus,
			PlaceholderCurrentTime,
			PlaceholderLanguage,
		}
	case PromptFieldContextTemplate:
		return []PromptPlaceholder{
			PlaceholderQuery,
			PlaceholderContexts,
			PlaceholderCurrentTime,
			PlaceholderCurrentWeek,
			PlaceholderLanguage,
		}
	case PromptFieldRewriteSystemPrompt:
		// Rewrite system prompt supports same placeholders as rewrite user prompt
		return []PromptPlaceholder{
			PlaceholderQuery,
			PlaceholderConversation,
			PlaceholderCurrentTime,
			PlaceholderYesterday,
			PlaceholderLanguage,
		}
	case PromptFieldRewritePrompt:
		return []PromptPlaceholder{
			PlaceholderQuery,
			PlaceholderConversation,
			PlaceholderCurrentTime,
			PlaceholderYesterday,
			PlaceholderLanguage,
		}
	case PromptFieldFallbackPrompt:
		return []PromptPlaceholder{
			PlaceholderQuery,
			PlaceholderLanguage,
		}
	default:
		return []PromptPlaceholder{}
	}
}

// AllPlaceholders returns all available placeholders in the system
func AllPlaceholders() []PromptPlaceholder {
	return []PromptPlaceholder{
		PlaceholderQuery,
		PlaceholderContexts,
		PlaceholderCurrentTime,
		PlaceholderCurrentWeek,
		PlaceholderConversation,
		PlaceholderYesterday,
		PlaceholderAnswer,
		PlaceholderKnowledgeBases,
		PlaceholderWebSearchStatus,
		PlaceholderLanguage,
	}
}

// PlaceholderMap returns a map of field types to their available placeholders
func PlaceholderMap() map[PromptFieldType][]PromptPlaceholder {
	return map[PromptFieldType][]PromptPlaceholder{
		PromptFieldSystemPrompt:        PlaceholdersByField(PromptFieldSystemPrompt),
		PromptFieldAgentSystemPrompt:   PlaceholdersByField(PromptFieldAgentSystemPrompt),
		PromptFieldContextTemplate:     PlaceholdersByField(PromptFieldContextTemplate),
		PromptFieldRewriteSystemPrompt: PlaceholdersByField(PromptFieldRewriteSystemPrompt),
		PromptFieldRewritePrompt:       PlaceholdersByField(PromptFieldRewritePrompt),
		PromptFieldFallbackPrompt:      PlaceholdersByField(PromptFieldFallbackPrompt),
	}
}

// ---------------------------------------------------------------------------
// Unified prompt placeholder rendering
// ---------------------------------------------------------------------------

// PlaceholderValues is a map of placeholder names (without braces) to their
// replacement values. Example: {"query": "How to use?", "language": "English"}
type PlaceholderValues map[string]string

// RenderPromptPlaceholders replaces all {{key}} occurrences in template with
// the corresponding values from vals. Unknown placeholders are left untouched.
//
// Built-in auto-values (filled when not supplied explicitly):
//   - {{current_time}} -> time.Now().Format("2006-01-02") (date only; clock
//     precision would bust provider prefix caches on every request)
//   - {{current_week}} -> current weekday name
//   - {{yesterday}}    -> yesterday's date (2006-01-02)
func RenderPromptPlaceholders(template string, vals PlaceholderValues) string {
	if template == "" {
		return ""
	}

	// Populate auto-generated values when callers don't supply them.
	autoFill := func(key, value string) {
		if _, exists := vals[key]; !exists {
			if strings.Contains(template, "{{"+key+"}}") {
				vals[key] = value
			}
		}
	}

	now := time.Now()
	autoFill("current_time", now.Format("2006-01-02"))
	autoFill("current_week", now.Weekday().String())
	autoFill("yesterday", now.AddDate(0, 0, -1).Format("2006-01-02"))

	result := template
	for key, value := range vals {
		placeholder := "{{" + key + "}}"
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}
	return result
}
