package service

import (
	"context"
	"strings"
	"testing"

	"github.com/acrbaran/rag/internal/types"
)

func TestFAQImportUserMessagesFollowLocale(t *testing.T) {
	progress := &types.FAQImportProgress{Total: 3, SuccessCount: 2, FailedCount: 1}
	for _, tc := range []struct {
		locale string
		want   string
		blank  string
	}{
		{"tr-TR", "Yüklenen: 3 / Başarılı: 2 / Başarısız: 1", "Ana soru boş olamaz"},
		{"en-US", "Uploaded: 3 / Succeeded: 2 / Failed: 1", "Main question cannot be empty"},
	} {
		ctx := context.WithValue(context.Background(), types.LanguageContextKey, tc.locale)
		message := (&knowledgeService{}).buildFAQImportResultMessage(ctx, "done", progress)
		if !strings.Contains(message, tc.want) {
			t.Errorf("locale %s: message = %q", tc.locale, message)
		}
		err := validateFAQEntryPayloadBasic(ctx, &types.FAQEntryPayload{})
		if err == nil || err.Error() != tc.blank {
			t.Errorf("locale %s: validation error = %v", tc.locale, err)
		}
	}
}
