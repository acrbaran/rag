// Package providers registers Tencent Hunyuan through its OpenAI-compatible
// endpoint.
//
// Facts (https://cloud.tencent.com/document/product/1729/111007):
//   - https://api.hunyuan.cloud.tencent.com/v1 serves both chat
//     (/chat/completions) and embeddings (/embeddings); the only credential
//     is a console API key sent as Authorization: Bearer, so this vendor
//     needs no ExtraFields (the TC3-signed native API is a different host);
//   - the output cap is `max_tokens` (default 4096); the reference does not
//     document `max_completion_tokens`;
//   - `tool_choice` isteğe bağlı değerleri none, auto, custom içerir ve "yalnızca hunyuan-turbos,
//     hunyuan-functioncall modelleri için geçerlidir", bu nedenle `required` asla gönderilmez; "custom" ise
//     a Hunyuan-specific shape rather than OpenAI's named-function object,
//     so the "function" mode is withheld as well;
//   - temperature is [0.0, 2.0] and top_p [0.0, 1.0], neither restricted;
//   - stream_options.include_usage is supported;
//   - hunyuan-embedding returns a fixed 1024-dimension vector.
//
// Thinking (https://cloud.tencent.com/document/product/1729/105701): the
// yerel API `EnableThinking` sunar, "değer aktarılmazsa varsayılan olarak etkinleştirilir" ve anahtar
// "yalnızca hunyuan-a13b modeli için geçerlidir". T1 serisi koşulsuz olarak çıkarım yapar, bu nedenle
// models.json marks it "off": null, and hunyuan-a13b additionally accepts a
// `/no_think` prompt prefix
// (https://cloud.tencent.com/document/product/1729/104753).
//
// Second protocol: an Anthropic Messages facade at
// https://api.hunyuan.cloud.tencent.com/anthropic (path
// /anthropic/v1/messages) documents hunyuan-2.0-thinking-20251109 and
// hunyuan-2.0-instruct-20251111 with thinking {type, budget_tokens}
// (https://cloud.tencent.com/document/product/1729/127293). The
// OpenAI-compatible surface stays this vendor's default.
//
// unverified: the OpenAI-compatible reference documents no thinking field at
// all, so the snake_case `enable_thinking` spelling used here is carried over
// from the native API and is unconfirmed for this endpoint;
// unverified: response_format, parallel_tool_calls and prompt-cache
// accounting are absent from the compatible-interface reference, so the
// protocol defaults (supported, no cache counters) stand untested;
// unverified: Tencent's model list (1729/104753) and pricing page
// (1729/97731) document limits for hunyuan-a13b (224k in / 32k out) and the
// vision models only. hunyuan-t1-latest, hunyuan-turbos-latest, hunyuan-lite
// and hunyuan-turbos-vision appear in neither table (new models are
// migrating to TokenHub), so their context windows, output caps and prices
// are carried over unconfirmed; no retirement notice was found either, so
// the entries stay. hunyuan-t1-latest used to claim a 64K output cap above
// its own 32K context window; since no doc supports either number, the cap
// was dropped rather than corrected, leaving the context window as the only
// (still unconfirmed) limit. Treat it as a placeholder that the per-model
// field in the UI overrides, and do not raise it without a documented
// source: an overstated context window stops history compaction from firing
// and the upstream rejects the request instead;
// unverified: the hunyuan-2.0-* ids are documented only on the Anthropic
// facade, so their availability on this OpenAI-compatible host, their
// context windows and their output caps are unconfirmed.
package providers

import (
	_ "embed"

	"github.com/acrbaran/rag/internal/models/api"
	"github.com/acrbaran/rag/internal/types"
)

//go:embed assets/hunyuan.svg
var hunyuanIcon []byte

// HunyuanID is the provider identifier stored on model rows.
const HunyuanID = "hunyuan"

// HunyuanBaseURL is the OpenAI-compatible endpoint (chat and embedding).
const HunyuanBaseURL = "https://api.hunyuan.cloud.tencent.com/v1"

func newHunyuanProvider() *Definition {
	return &Definition{
		ID:           HunyuanID,
		Name:         "Tencent Hunyuan",
		Names:        map[string]string{"zh-CN": "腾讯混元 Hunyuan"},
		Description:  "hunyuan-t1-latest, hunyuan-turbos-latest, hunyuan-a13b, hunyuan-embedding, etc.",
		Website:      "https://cloud.tencent.com/product/hunyuan",
		Icon:         hunyuanIcon,
		API:          api.APIOpenAICompletions,
		Order:        13,
		RequiresAuth: true,
		Auth:         AuthBearer,
		URLPatterns:  []string{"hunyuan.cloud.tencent.com"},
		DefaultBaseURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: HunyuanBaseURL,
			types.ModelTypeEmbedding:   HunyuanBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
		},
		Compat: VendorCompat{
			// Embeddings yalın temeli kasıtlı olarak korur: "Embedding arayüzü şu anda yalnızca
			// input ve model parametrelerini destekler … dimensions 1024 olarak sabittir"
			// (https://cloud.tencent.com/document/product/1729/111007).
			OpenAICompletions: api.OpenAICompletionsCompat{
				MaxTokensField: api.Ptr("max_tokens"),
				ThinkingFormat: api.Ptr(api.ThinkingFormatEnableThinking),
				// isteğe bağlı değerler none, auto, custom içerir: ne "required" ne de OpenAI'nin
				// named-function object is documented.
				ToolChoiceModes: []string{"none", "auto"},
			},
		},
	}
}
