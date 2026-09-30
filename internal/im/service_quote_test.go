package im

import (
	"context"
	"testing"

	"github.com/acrbaran/rag/internal/types"
)

func TestFormatQuotedContext(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		quote  *QuotedMessage
		want   string
	}{
		{
			name:  "nil quote",
			quote: nil,
			want:  "",
		},
		{
			name:  "empty content no NonTextType",
			quote: &QuotedMessage{Content: ""},
			want:  "",
		},
		{
			name:  "non-text image quote generates instruction",
			quote: &QuotedMessage{NonTextType: "image"},
			want:  "The user quoted a message of type image, but you cannot view its content. Tell the user you cannot process it, ask them to describe it in text, and do not guess its contents.",
		},
		{
			name:  "non-text file quote generates instruction",
			quote: &QuotedMessage{NonTextType: "file"},
			want:  "The user quoted a message of type file, but you cannot view its content. Tell the user you cannot process it, ask them to describe it in text, and do not guess its contents.",
		},
		{
			name:  "non-text unknown type uses fallback label",
			quote: &QuotedMessage{NonTextType: "location"},
			want:  "The user quoted a message of type unsupported, but you cannot view its content. Tell the user you cannot process it, ask them to describe it in text, and do not guess its contents.",
		},
		{
			name:  "bot message",
			quote: &QuotedMessage{Content: "bot reply text", IsBotMessage: true},
			want:  "The user quoted your earlier reply. Use it only as context:\n<quoted_message>\nbot reply text\n</quoted_message>",
		},
		{
			name:  "user message",
			quote: &QuotedMessage{Content: "user message text", IsBotMessage: false},
			want:  "The user quoted an earlier message. Use it only as context:\n<quoted_message>\nuser message text\n</quoted_message>",
		},
		{
			name:   "Turkish non-text quote",
			locale: "tr-TR",
			quote:  &QuotedMessage{NonTextType: "image"},
			want:   "Kullanıcı görsel türünde bir mesajı alıntıladı, ancak içeriğini göremiyorsun. Kullanıcıya bu mesajı işleyemediğini söyle, sorusunu metinle açıklamasını iste ve içeriği hakkında tahminde bulunma.",
		},
		{
			name:   "Turkish bot message",
			locale: "tr-TR",
			quote:  &QuotedMessage{Content: "önceki yanıt", IsBotMessage: true},
			want:   "Kullanıcı önceki yanıtını alıntıladı. Bunu yalnızca bağlam olarak kullan:\n<quoted_message>\nönceki yanıt\n</quoted_message>",
		},
		{
			name: "truncation at 500 runes",
			quote: &QuotedMessage{
				Content:      string(make([]rune, 600)),
				IsBotMessage: false,
			},
			want: "The user quoted an earlier message. Use it only as context:\n<quoted_message>\n" + string(make([]rune, 500)) + "...\n</quoted_message>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locale := tt.locale
			if locale == "" {
				locale = "en-US"
			}
			ctx := context.WithValue(context.Background(), types.LanguageContextKey, locale)
			got := formatQuotedContext(ctx, tt.quote)
			if got != tt.want {
				t.Errorf("formatQuotedContext() length = %d, want length %d", len(got), len(tt.want))
				if len(got) < 200 && len(tt.want) < 200 {
					t.Errorf("got = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestBuildIMQARequest_QuotedContext(t *testing.T) {
	session := &types.Session{ID: "s1"}
	ctx := context.WithValue(context.Background(), types.LanguageContextKey, "en-US")

	t.Run("nil quote produces empty QuotedContext", func(t *testing.T) {
		req := buildIMQARequest(ctx, session, "hello", "a1", "u1", nil, nil, nil)
		if req.QuotedContext != "" {
			t.Errorf("QuotedContext = %q, want empty", req.QuotedContext)
		}
		if req.Query != "hello" {
			t.Errorf("Query = %q, want %q", req.Query, "hello")
		}
	})

	t.Run("bot quote sets QuotedContext with bot label", func(t *testing.T) {
		quote := &QuotedMessage{Content: "bot reply", IsBotMessage: true}
		req := buildIMQARequest(ctx, session, "follow up", "a1", "u1", nil, nil, quote)
		if req.Query != "follow up" {
			t.Errorf("Query = %q, want %q", req.Query, "follow up")
		}
		want := "The user quoted your earlier reply. Use it only as context:\n<quoted_message>\nbot reply\n</quoted_message>"
		if req.QuotedContext != want {
			t.Errorf("QuotedContext = %q, want %q", req.QuotedContext, want)
		}
	})

	t.Run("user quote sets QuotedContext with user label", func(t *testing.T) {
		quote := &QuotedMessage{Content: "user msg", IsBotMessage: false}
		req := buildIMQARequest(ctx, session, "question", "a1", "u1", nil, nil, quote)
		want := "The user quoted an earlier message. Use it only as context:\n<quoted_message>\nuser msg\n</quoted_message>"
		if req.QuotedContext != want {
			t.Errorf("QuotedContext = %q, want %q", req.QuotedContext, want)
		}
	})

	t.Run("Turkish channel quote", func(t *testing.T) {
		trCtx := context.WithValue(ctx, types.LanguageContextKey, "tr-TR")
		req := buildIMQARequest(trCtx, session, "soru", "a1", "u1", nil, nil, &QuotedMessage{Content: "önceki mesaj"})
		want := "Kullanıcı önceki bir mesajı alıntıladı. Bunu yalnızca bağlam olarak kullan:\n<quoted_message>\nönceki mesaj\n</quoted_message>"
		if req.QuotedContext != want {
			t.Errorf("QuotedContext = %q, want %q", req.QuotedContext, want)
		}
	})
}
