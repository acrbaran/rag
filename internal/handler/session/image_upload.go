package session

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	filesvc "github.com/acrbaran/rag/internal/application/service/file"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/google/uuid"
)

const (
	maxImageSize   = 10 << 20 // 10MB per image
	maxImagesCount = 5
)

// saveImageAttachments decodes base64 images from the request and saves them to
// storage. The images slice is mutated in place: URL is populated.
// This is always called when images are present. VLM analysis is handled
// separately (either in the pipeline rewrite step for RAG paths, or via
// analyzeImageAttachments for pure chat paths with non-vision models).
func (h *Handler) saveImageAttachments(ctx context.Context, images []ImageAttachment, tenantID uint64, storageProvider string) error {
	if len(images) == 0 {
		return nil
	}
	if len(images) > maxImagesCount {
		return fmt.Errorf("too many images, max %d", maxImagesCount)
	}

	fileSvc := h.resolveImageFileService(ctx, storageProvider)

	for i := range images {
		img := &images[i]
		if img.Data == "" {
			continue
		}

		imgBytes, ext, err := decodeDataURI(img.Data)
		if err != nil {
			return fmt.Errorf("decode image %d: %w", i, err)
		}
		if len(imgBytes) > maxImageSize {
			return fmt.Errorf("image %d too large (%d bytes, max %d)", i, len(imgBytes), maxImageSize)
		}

		storedName := fmt.Sprintf("chat-images/%s%s", uuid.New().String(), ext)
		fileURL, err := fileSvc.SaveBytes(ctx, imgBytes, tenantID, storedName, false)
		if err != nil {
			return fmt.Errorf("save image %d: %w", i, err)
		}
		img.URL = fileURL
	}

	return nil
}

// analyzeImageAttachments runs VLM analysis on saved images and populates Caption.
// Used as a fallback for pure chat paths where the pipeline rewrite step won't run.
// For RAG paths, image analysis is handled in the pipeline rewrite step instead.
func (h *Handler) analyzeImageAttachments(ctx context.Context, images []ImageAttachment, vlmModelID string, userQuery string) {
	if len(images) == 0 || vlmModelID == "" {
		return
	}

	vlmModel, err := h.modelService.GetVLMModel(ctx, vlmModelID)
	if err != nil {
		logger.Warnf(ctx, "No VLM model available for image analysis, skipping: %v", err)
		return
	}

	for i := range images {
		img := &images[i]
		if img.Data == "" {
			continue
		}
		imgBytes, _, decErr := decodeDataURI(img.Data)
		if decErr != nil {
			logger.Warnf(ctx, "Failed to decode image %d for VLM analysis: %v", i, decErr)
			continue
		}
		prompt := buildImageAnalysisPrompt(ctx, userQuery)
		analysis, analysisErr := vlmModel.Predict(ctx, [][]byte{imgBytes}, prompt)
		if analysisErr != nil {
			logger.Warnf(ctx, "VLM analysis failed for image %d: %v", i, analysisErr)
		} else {
			img.Caption = analysis
		}
	}
}

// buildImageAnalysisPrompt generates a context-aware VLM prompt based on the
// user's question. Instead of doing generic OCR + Caption separately, we do a
// single analysis call that is tailored to the user's intent.
func buildImageAnalysisPrompt(ctx context.Context, userQuery string) string {
	if strings.TrimSpace(userQuery) == "" {
		return types.LocalizedText(ctx,
			"Bu görselin içeriğini analiz et. Metin varsa önemli bilgileri çıkar; doğal bir görüntüyse ana içeriğini açıkla. Kısa ve öz Türkçe yanıt ver.",
			"Analyze this image. Extract key information if it contains text; otherwise describe its main content. Answer concisely in English.")
	}
	return fmt.Sprintf(types.LocalizedText(ctx,
		`Kullanıcının sorusu: %s

Görseli soruyla ilgili olarak analiz et. Metin, belge veya tablo içeriyorsa ilgili önemli bilgileri çıkar. Fotoğraf, ekran görüntüsü veya grafikse ilgili görsel içeriği açıkla. Yalnızca kısa ve öz Türkçe analiz sonucunu yaz.`,
		`User's question: %s

Analyze the image in relation to the question. Extract relevant key information from any text, document or table. For a photo, screenshot or chart, describe the relevant visual content. Output only a concise analysis in English.`), userQuery)
}

func decodeDataURI(dataURI string) ([]byte, string, error) {
	if !strings.HasPrefix(dataURI, "data:") {
		return nil, "", fmt.Errorf("not a data URI")
	}
	idx := strings.Index(dataURI, ";base64,")
	if idx < 0 {
		return nil, "", fmt.Errorf("unsupported data URI encoding (expected base64)")
	}
	mimeType := dataURI[5:idx]
	decoded, err := base64.StdEncoding.DecodeString(dataURI[idx+8:])
	if err != nil {
		return nil, "", fmt.Errorf("base64 decode: %w", err)
	}
	ext := mimeToExt(mimeType)
	return decoded, ext, nil
}

func mimeToExt(mime string) string {
	switch strings.ToLower(mime) {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}

func (h *Handler) resolveImageFileService(ctx context.Context, storageProvider string) interfaces.FileService {
	tenant, _ := ctx.Value(types.TenantInfoContextKey).(*types.Tenant)
	if tenant == nil {
		return h.fileService
	}
	if h.storageResolver != nil {
		svc, resolvedProvider, err := h.storageResolver.ResolveFileService(ctx, tenant, "", storageProvider, "")
		if err == nil && svc != nil {
			logger.Infof(ctx, "[image-storage] using storage instance provider=%s for image uploads", resolvedProvider)
			return svc
		}
		if err != nil {
			logger.Warnf(ctx, "[image-storage] failed to resolve storage instance for provider=%s: %v", storageProvider, err)
		}
	}
	if strings.TrimSpace(storageProvider) == "" || tenant.StorageEngineConfig == nil {
		return h.fileService
	}

	svc, resolvedProvider, err := filesvc.NewFileServiceFromStorageConfig(storageProvider, tenant.StorageEngineConfig, "")
	if err != nil {
		logger.Warnf(ctx, "[image-storage] failed to create %s file service: %v, fallback to default", storageProvider, err)
		return h.fileService
	}
	logger.Infof(ctx, "[image-storage] using provider=%s for image uploads", resolvedProvider)
	return svc
}
