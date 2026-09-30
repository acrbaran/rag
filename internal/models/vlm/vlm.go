package vlm

import (
	"context"
	"fmt"

	"github.com/acrbaran/rag/internal/logger"
	modelruntime "github.com/acrbaran/rag/internal/models/runtime"
	"github.com/acrbaran/rag/internal/types"
)

// VLM defines the interface for Vision Language Model operations.
type VLM interface {
	// Predict sends one or more images with a text prompt to the VLM and returns the generated text.
	Predict(ctx context.Context, imgBytes [][]byte, prompt string) (string, error)

	GetModelName() string
	GetModelID() string
}

// Config holds the configuration needed to create a VLM instance.
type Config struct {
	Source        types.ModelSource
	BaseURL       string
	ModelName     string
	APIKey        string
	ModelID       string
	InterfaceType string // "openai" (default)
	Provider      string
	// MaxConcurrency caps concurrent background calls to this model; 0 falls
	// back to the process-wide default (see limiter.GateN).
	MaxConcurrency int
	// Spec carries per-row catalog overrides (protocol, compat, levels). VLM
	// calls now go through the chat factory, so the override has to travel
	// with them — otherwise /models reports capabilities computed WITH the
	// override while the actual request is built without it.
	Spec  *types.ModelSpecOverride
	Extra map[string]any
	// CustomHeaders, uzak API çağrılırken özel HTTP istek başlıklarının eklenmesine izin verir (OpenAI Python SDK'sindeki extra_headers'a benzer).
	CustomHeaders map[string]string
	AppID         string
	AppSecret     string
}

// ConfigFromModel, types.Model'den vlm.Config oluşturur.
// Üretim yolu (DB'den başlatılan) ve test bağlantısı yolu (geçici form) bu eşlemeyi paylaşır.
// appID / appSecret, şifresi çözülmüş model kimlik bilgileridir; çağıran bunları iletmekten sorumludur.
// InterfaceType, source / model parametrelerine göre otomatik olarak makul varsayılan değere geri döner.
func ConfigFromModel(m *types.Model, appID, appSecret string) *Config {
	if m == nil {
		return nil
	}
	ifType := m.Parameters.InterfaceType
	if ifType == "" {
		ifType = "openai"
	}
	return &Config{
		ModelID:        m.ID,
		APIKey:         m.Parameters.APIKey,
		BaseURL:        m.Parameters.BaseURL,
		ModelName:      m.Name,
		Source:         m.Source,
		InterfaceType:  ifType,
		Provider:       m.Parameters.Provider,
		MaxConcurrency: m.Parameters.MaxConcurrency,
		Spec:           m.Parameters.Spec,
		Extra:          stringMapToAnyMap(m.Parameters.ExtraConfig),
		CustomHeaders:  m.Parameters.CustomHeaders,
		AppID:          appID,
		AppSecret:      appSecret,
	}
}

func stringMapToAnyMap(in map[string]string) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// NewVLM creates a VLM instance based on the provided configuration.
func NewVLM(config *Config) (VLM, error) {
	v, err := newVLM(config)
	if err != nil {
		return v, err
	}
	if logger.LLMDebugEnabled() {
		v = &debugVLM{inner: v}
	}
	v, err = wrapVLMLangfuse(v, nil)
	// Outermost: hold the per-model concurrency slot only around the real
	// provider round-trip, so the wait is excluded from debug/langfuse timing.
	return wrapVLMConcurrency(v, config.MaxConcurrency, err)
}

func newVLM(config *Config) (VLM, error) {
	providerID := config.Provider
	if providerID == "" {
		providerID = modelruntime.DetectByURL(config.BaseURL)
	}
	return NewRemoteAPIVLM(config)
}

// NewVLMFromLegacyConfig creates a VLM from a legacy VLMConfig (inline BaseURL/APIKey/ModelName).
func NewVLMFromLegacyConfig(vlmCfg types.VLMConfig) (VLM, error) {
	if !vlmCfg.IsEnabled() {
		return nil, fmt.Errorf("VLM config is not enabled")
	}

	ifType := vlmCfg.InterfaceType
	if ifType == "" {
		ifType = "openai"
	}

	return NewVLM(&Config{
		Source:        types.ModelSourceRemote,
		BaseURL:       vlmCfg.BaseURL,
		ModelName:     vlmCfg.ModelName,
		APIKey:        vlmCfg.APIKey,
		InterfaceType: ifType,
	})
}
