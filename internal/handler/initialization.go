package handler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/acrbaran/rag/internal/application/repository"
	chatpipeline "github.com/acrbaran/rag/internal/application/service/chat_pipeline"
	"github.com/acrbaran/rag/internal/assets"
	"github.com/acrbaran/rag/internal/config"
	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/handler/dto"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/middleware"
	"github.com/acrbaran/rag/internal/models/asr"
	"github.com/acrbaran/rag/internal/models/chat"
	"github.com/acrbaran/rag/internal/models/embedding"
	"github.com/acrbaran/rag/internal/models/rerank"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
)

// InitializationHandler başlatma işleyicisi
type InitializationHandler struct {
	config           *config.Config
	tenantService    interfaces.TenantService
	modelService     interfaces.ModelService
	kbService        interfaces.KnowledgeBaseService
	kbRepository     interfaces.KnowledgeBaseRepository
	knowledgeService interfaces.KnowledgeService
	documentReader   interfaces.DocumentReader
	pooler           embedding.EmbedderPooler
	storageResolver  interfaces.StorageBackendResolver
}

// NewInitializationHandler başlatma işleyicisi oluşturur
func NewInitializationHandler(
	config *config.Config,
	tenantService interfaces.TenantService,
	modelService interfaces.ModelService,
	kbService interfaces.KnowledgeBaseService,
	kbRepository interfaces.KnowledgeBaseRepository,
	knowledgeService interfaces.KnowledgeService,
	documentReader interfaces.DocumentReader,
	pooler embedding.EmbedderPooler,
	storageResolver interfaces.StorageBackendResolver,
) *InitializationHandler {
	return &InitializationHandler{
		config:           config,
		tenantService:    tenantService,
		modelService:     modelService,
		kbService:        kbService,
		kbRepository:     kbRepository,
		knowledgeService: knowledgeService,
		documentReader:   documentReader,
		pooler:           pooler,
		storageResolver:  storageResolver,
	}
}

// KBModelConfigRequest bilgi tabanı model yapılandırma isteği (basitleştirilmiş sürüm, yalnızca model ID'si gönderilir)
type KBModelConfigRequest struct {
	LLMModelID       string           `json:"llmModelId"       binding:"required"`
	EmbeddingModelID string           `json:"embeddingModelId"` // optional when RAG indexing is disabled
	VLMConfig        *types.VLMConfig `json:"vlm_config"`
	ASRConfig        *types.ASRConfig `json:"asr_config"`

	// Belge parçalama yapılandırması
	DocumentSplitting struct {
		ChunkSize         int                      `json:"chunkSize"`
		ChunkOverlap      int                      `json:"chunkOverlap"`
		Separators        []string                 `json:"separators"`
		ParserEngineRules []types.ParserEngineRule `json:"parserEngineRules,omitempty"`
		EnableParentChild bool                     `json:"enableParentChild"`
		ParentChunkSize   int                      `json:"parentChunkSize,omitempty"`
		ChildChunkSize    int                      `json:"childChunkSize,omitempty"`
		// Strategy / TokenLimit / Languages use pointer types so the
		// handler can distinguish "field absent in payload" (no change)
		// from "field present with empty/zero value" (clear / disable).
		// Without that distinction, users could set strategy="auto" once
		// but never reset it back to legacy / unset.
		Strategy                  *string   `json:"strategy,omitempty"`
		TokenLimit                *int      `json:"tokenLimit,omitempty"`
		Languages                 *[]string `json:"languages,omitempty"`
		TableMetadataInstructions *string   `json:"tableMetadataInstructions,omitempty"`
	} `json:"documentSplitting"`

	// Çok modlu yapılandırma (yalnızca modelle ilgili; depolama motoru storageProvider içinde yapılandırılır)
	Multimodal struct {
		Enabled bool `json:"enabled"`
	} `json:"multimodal"`

	// Depolama motoru seçimi ("local" | "minio" | "cos"); belge yüklemeyi ve belge içindeki görsellerin depolanmasını etkiler, parametreler genel ayarlardan okunur
	StorageProvider  string `json:"storageProvider"`
	StorageBackendID string `json:"storageBackendId"`

	// Bilgi grafiği yapılandırması
	NodeExtract struct {
		Enabled            bool                  `json:"enabled"`
		Text               string                `json:"text"`
		Tags               []string              `json:"tags"`
		Nodes              []types.GraphNode     `json:"nodes"`
		Relations          []types.GraphRelation `json:"relations"`
		CustomInstructions string                `json:"customInstructions"`
	} `json:"nodeExtract"`

	// Soru oluşturma yapılandırması
	QuestionGeneration struct {
		Enabled            bool   `json:"enabled"`
		QuestionCount      int    `json:"questionCount"`
		CustomInstructions string `json:"customInstructions"`
	} `json:"questionGeneration"`
}

// InitializationRequest başlatma istek yapısı
type InitializationRequest struct {
	LLM struct {
		Source    string `json:"source" binding:"required"`
		ModelName string `json:"modelName" binding:"required"`
		BaseURL   string `json:"baseUrl"`
		APIKey    string `json:"apiKey"`
	} `json:"llm" binding:"required"`

	Embedding struct {
		Source    string `json:"source" binding:"required"`
		ModelName string `json:"modelName" binding:"required"`
		BaseURL   string `json:"baseUrl"`
		APIKey    string `json:"apiKey"`
		Dimension int    `json:"dimension"` // embedding boyutu alanı eklendi
	} `json:"embedding" binding:"required"`

	Rerank struct {
		Enabled   bool   `json:"enabled"`
		ModelName string `json:"modelName"`
		BaseURL   string `json:"baseUrl"`
		APIKey    string `json:"apiKey"`
	} `json:"rerank"`

	Multimodal struct {
		Enabled bool `json:"enabled"`
		VLM     *struct {
			ModelName     string `json:"modelName"`
			BaseURL       string `json:"baseUrl"`
			APIKey        string `json:"apiKey"`
			InterfaceType string `json:"interfaceType"` // "openai"
		} `json:"vlm,omitempty"`
		StorageType string `json:"storageType"`
		COS         *struct {
			SecretID   string `json:"secretId"`
			SecretKey  string `json:"secretKey"`
			Region     string `json:"region"`
			BucketName string `json:"bucketName"`
			AppID      string `json:"appId"`
			PathPrefix string `json:"pathPrefix"`
		} `json:"cos,omitempty"`
		Minio *struct {
			BucketName string `json:"bucketName"`
			PathPrefix string `json:"pathPrefix"`
		} `json:"minio,omitempty"`
	} `json:"multimodal"`

	DocumentSplitting struct {
		ChunkSize    int      `json:"chunkSize" binding:"required,min=100,max=10000"`
		ChunkOverlap int      `json:"chunkOverlap" binding:"min=0"`
		Separators   []string `json:"separators" binding:"required,min=1"`
	} `json:"documentSplitting" binding:"required"`

	NodeExtract struct {
		Enabled bool     `json:"enabled"`
		Text    string   `json:"text"`
		Tags    []string `json:"tags"`
		Nodes   []struct {
			Name       string   `json:"name"`
			Attributes []string `json:"attributes"`
		} `json:"nodes"`
		Relations []struct {
			Node1 string `json:"node1"`
			Node2 string `json:"node2"`
			Type  string `json:"type"`
		} `json:"relations"`
	} `json:"nodeExtract"`

	QuestionGeneration struct {
		Enabled       bool `json:"enabled"`
		QuestionCount int  `json:"questionCount"`
	} `json:"questionGeneration"`
}

// UpdateKBConfig godoc
// @Summary      Bilgi tabanı yapılandırmasını güncelle
// @Description  Bilgi tabanı ID'sine göre model ve parçalama yapılandırmasını günceller
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        kbId     path      string               true  "Bilgi tabanı ID'si"
// @Param        request  body      KBModelConfigRequest true  "Yapılandırma isteği"
// @Success      200      {object}  map[string]interface{}  "Güncelleme başarılı"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Failure      404      {object}  errors.AppError         "Bilgi tabanı bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/config/{kbId} [put]
func (h *InitializationHandler) UpdateKBConfig(c *gin.Context) {
	ctx := c.Request.Context()
	kbIdStr := utils.SanitizeForLog(c.Param("kbId"))

	var req KBModelConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse KB config request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Bilgi tabanı bilgilerini al
	kb, err := h.kbService.GetKnowledgeBaseByID(ctx, kbIdStr)
	if err != nil || kb == nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"kbId": utils.SanitizeForLog(kbIdStr)})
		c.Error(errors.NewNotFoundError(types.LocalizedText(ctx, "Bilgi tabanı bulunamadı", "Knowledge base not found")))
		return
	}
	ownWorkspace, err := kbSettingsAccess(c, kb)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Embedding modelinin değiştirilip değiştirilemeyeceğini kontrol et
	if kb.EmbeddingModelID != "" && req.EmbeddingModelID != "" && kb.EmbeddingModelID != req.EmbeddingModelID {
		// Dosya olup olmadığını kontrol et
		knowledgeList, err := h.knowledgeService.ListPagedKnowledgeByKnowledgeBaseID(ctx,
			kbIdStr, &types.Pagination{
				Page:     1,
				PageSize: 1,
			}, types.KnowledgeListFilter{})
		if err == nil && knowledgeList != nil && knowledgeList.Total > 0 {
			logger.Error(ctx, "Cannot change embedding model when files exist")
			c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Bilgi tabanında dosyalar var; Embedding modeli değiştirilemez", "The embedding model cannot be changed while the knowledge base contains files")))
			return
		}
	}

	// Model ayrıntılarını veritabanından al ve doğrula
	llmModel, err := h.modelService.GetModelByID(ctx, req.LLMModelID)
	if err != nil || llmModel == nil {
		logger.Error(ctx, "LLM model not found")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "LLM modeli bulunamadı", "LLM model not found")))
		return
	}

	// Embedding modelini yalnızca gerektiğinde doğrula (RAG araması etkin olduğunda)
	if req.EmbeddingModelID != "" {
		embeddingModel, err := h.modelService.GetModelByID(ctx, req.EmbeddingModelID)
		if err != nil || embeddingModel == nil {
			logger.Error(ctx, "Embedding model not found")
			c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Embedding modeli bulunamadı", "Embedding model not found")))
			return
		}
	}

	// Bilgi tabanının model kimliğini güncelle
	kb.SummaryModelID = req.LLMModelID
	if req.EmbeddingModelID != "" {
		kb.EmbeddingModelID = req.EmbeddingModelID
	}

	// Çok modlu model yapılandırmasını işle
	kb.VLMConfig = types.VLMConfig{}
	if req.VLMConfig != nil && req.Multimodal.Enabled && req.VLMConfig.ModelID != "" {
		vllmModel, err := h.modelService.GetModelByID(ctx, req.VLMConfig.ModelID)
		if err != nil || vllmModel == nil {
			logger.Warn(ctx, "VLM model not found")
		} else {
			kb.VLMConfig.Enabled = req.VLMConfig.Enabled
			kb.VLMConfig.ModelID = req.VLMConfig.ModelID
		}
	}
	if !kb.VLMConfig.Enabled {
		kb.VLMConfig.ModelID = ""
	}

	// ASR konuşma tanıma yapılandırmasını işle
	kb.ASRConfig = types.ASRConfig{}
	if req.ASRConfig != nil && req.ASRConfig.Enabled && req.ASRConfig.ModelID != "" {
		asrModel, err := h.modelService.GetModelByID(ctx, req.ASRConfig.ModelID)
		if err != nil || asrModel == nil {
			logger.Warn(ctx, "ASR model not found")
		} else {
			kb.ASRConfig.Enabled = true
			kb.ASRConfig.ModelID = req.ASRConfig.ModelID
			kb.ASRConfig.Language = req.ASRConfig.Language
		}
	}

	// Belge parçalama yapılandırmasını güncelle
	if req.DocumentSplitting.ChunkSize > 0 {
		kb.ChunkingConfig.ChunkSize = req.DocumentSplitting.ChunkSize
	}
	if req.DocumentSplitting.ChunkOverlap >= 0 {
		kb.ChunkingConfig.ChunkOverlap = req.DocumentSplitting.ChunkOverlap
	}
	if len(req.DocumentSplitting.Separators) > 0 {
		kb.ChunkingConfig.Separators = req.DocumentSplitting.Separators
	}
	kb.ChunkingConfig.ParserEngineRules = req.DocumentSplitting.ParserEngineRules
	kb.ChunkingConfig.EnableParentChild = req.DocumentSplitting.EnableParentChild
	if req.DocumentSplitting.ParentChunkSize > 0 {
		kb.ChunkingConfig.ParentChunkSize = req.DocumentSplitting.ParentChunkSize
	}
	if req.DocumentSplitting.ChildChunkSize > 0 {
		kb.ChunkingConfig.ChildChunkSize = req.DocumentSplitting.ChildChunkSize
	}
	// Pointer-based fields support clearing (empty string / 0 / empty slice
	// is a valid "user picked default again" signal; absent in payload means
	// "no change").
	if req.DocumentSplitting.Strategy != nil {
		kb.ChunkingConfig.Strategy = *req.DocumentSplitting.Strategy
	}
	if req.DocumentSplitting.TokenLimit != nil {
		kb.ChunkingConfig.TokenLimit = *req.DocumentSplitting.TokenLimit
	}
	if req.DocumentSplitting.Languages != nil {
		kb.ChunkingConfig.Languages = *req.DocumentSplitting.Languages
	}
	if req.DocumentSplitting.TableMetadataInstructions != nil {
		kb.ChunkingConfig.TableMetadataInstructions = strings.TrimSpace(*req.DocumentSplitting.TableMetadataInstructions)
	}

	// Çok modlu yapılandırmayı güncelle
	if req.Multimodal.Enabled {
		// VLM model already set above
	} else {
		kb.VLMConfig.ModelID = ""
	}
	if req.VLMConfig != nil {
		kb.VLMConfig.DescriptionLanguage = strings.TrimSpace(req.VLMConfig.DescriptionLanguage)
		kb.VLMConfig.CustomInstructions = strings.TrimSpace(req.VLMConfig.CustomInstructions)
	}

	// Storage backends resolve per workspace and belong to the owner's
	// infrastructure: another workspace can neither see the owner's backends
	// nor bind the KB to one of its own, so it may only leave them unchanged.
	if ownWorkspace {
		if err := h.applyKBStorageBinding(ctx, kb, kbIdStr, &req); err != nil {
			_ = c.Error(err)
			return
		}
	} else if kbStorageBindingChanged(kb, req.StorageBackendID, req.StorageProvider) {
		_ = c.Error(errors.NewForbiddenError(types.LocalizedText(ctx, "Depolama ayarlarını yalnızca bilgi tabanının ait olduğu çalışma alanı değiştirebilir", "Only the knowledge base workspace can change storage settings")))
		return
	}

	// Bilgi grafiği yapılandırmasını güncelle
	if req.NodeExtract.Enabled {
		// Nodes ve Relations öğelerini işaretçi türüne dönüştür
		nodes := make([]*types.GraphNode, len(req.NodeExtract.Nodes))
		for i := range req.NodeExtract.Nodes {
			nodes[i] = &req.NodeExtract.Nodes[i]
		}
		relations := make([]*types.GraphRelation, len(req.NodeExtract.Relations))
		for i := range req.NodeExtract.Relations {
			relations[i] = &req.NodeExtract.Relations[i]
		}

		kb.ExtractConfig = &types.ExtractConfig{
			Enabled:            req.NodeExtract.Enabled,
			Text:               req.NodeExtract.Text,
			Tags:               req.NodeExtract.Tags,
			Nodes:              nodes,
			Relations:          relations,
			CustomInstructions: strings.TrimSpace(req.NodeExtract.CustomInstructions),
		}
	} else if kb.ExtractConfig != nil {
		kb.ExtractConfig.Enabled = false
	} else {
		kb.ExtractConfig = &types.ExtractConfig{Enabled: false}
	}
	if err := validateExtractConfig(kb.ExtractConfig); err != nil {
		logger.Error(ctx, "Invalid extract configuration", err)
		c.Error(err)
		return
	}

	// Soru oluşturma yapılandırmasını güncelle
	if req.QuestionGeneration.Enabled {
		questionCount := req.QuestionGeneration.QuestionCount
		if questionCount <= 0 {
			questionCount = 3
		}
		if questionCount > 10 {
			questionCount = 10
		}
		kb.QuestionGenerationConfig = &types.QuestionGenerationConfig{
			Enabled:            true,
			QuestionCount:      questionCount,
			CustomInstructions: strings.TrimSpace(req.QuestionGeneration.CustomInstructions),
		}
	} else {
		kb.QuestionGenerationConfig = &types.QuestionGenerationConfig{
			Enabled:            false,
			CustomInstructions: strings.TrimSpace(req.QuestionGeneration.CustomInstructions),
		}
	}
	types.NormalizeKnowledgeBasePromptInstructions(kb)
	if err := validateKnowledgeBasePromptInstructions(kb); err != nil {
		c.Error(err)
		return
	}

	// Güncellenmiş bilgi tabanını kaydet
	if err := h.kbRepository.UpdateKnowledgeBase(ctx, kb); err != nil {
		logger.Error(ctx, "Failed to update knowledge base", err)
		c.Error(errors.NewInternalServerError(types.LocalizedText(ctx, "Bilgi tabanı güncellenemedi: ", "Failed to update knowledge base: ") + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": types.LocalizedText(ctx, "Yapılandırma güncellendi", "Configuration updated successfully"),
	})
}

// kbSettingsAccess reports whether the caller's workspace owns kb. Another
// workspace may change KB settings only through an admin share: the frontend
// offers KB settings to share admins alone, and OrgRoleEditor edits content,
// not settings. KBAccessWrite on the route also admits share editors, so the
// grant's effective share permission is checked here.
func kbSettingsAccess(c *gin.Context, kb *types.KnowledgeBase) (bool, error) {
	if kb.TenantID == types.CallerFromContext(c.Request.Context()).TenantID {
		return true, nil
	}
	grant, ok := middleware.KBAccessFromContext(c)
	if !ok || grant.KnowledgeBase == nil || grant.KnowledgeBase.ID != kb.ID ||
		!grant.Permission.HasPermission(types.OrgRoleAdmin) {
		return false, errors.NewForbiddenError(types.LocalizedText(c.Request.Context(), "Paylaşılan bilgi tabanının ayarlarını değiştirmek için yönetici paylaşım izni gerekir", "Admin sharing permission is required to change shared knowledge base settings"))
	}
	return false, nil
}

// kbStorageBindingChanged reports whether a config request would rebind the
// KB's storage. A backend ID that matches the current one leaves the binding
// alone (its provider is only a projection); without one, an empty current
// provider means the default, which clients echo back as "local".
func kbStorageBindingChanged(kb *types.KnowledgeBase, backendID, provider string) bool {
	backendID = strings.TrimSpace(backendID)
	current := ""
	if kb.StorageBackendID != nil {
		current = *kb.StorageBackendID
	}
	if backendID != "" {
		return backendID != current
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	currentProvider := kb.GetStorageProvider()
	if currentProvider == "" {
		currentProvider = "local"
	}
	return provider != "" && provider != currentProvider
}

// applyKBStorageBinding binds the owner's storage instance to the KB. The
// caller's workspace must own the KB: backends resolve against TenantInfo.
func (h *InitializationHandler) applyKBStorageBinding(
	ctx context.Context, kb *types.KnowledgeBase, kbID string, req *KBModelConfigRequest,
) error {
	// Bind the concrete storage instance. Provider remains a compatibility
	// projection for older clients and historical rows.
	if strings.TrimSpace(req.StorageBackendID) != "" {
		tenant, _ := types.TenantInfoFromContext(ctx)
		backend, resolveErr := h.storageResolver.ResolveBackend(ctx, tenant, req.StorageBackendID, "")
		if resolveErr != nil || backend == nil {
			return errors.NewBadRequestError("Storage backend is unavailable")
		}
		oldID := ""
		if kb.StorageBackendID != nil {
			oldID = *kb.StorageBackendID
		}
		if oldID != "" && oldID != backend.ID {
			knowledgeList, listErr := h.knowledgeService.ListPagedKnowledgeByKnowledgeBaseID(ctx,
				kbID, &types.Pagination{Page: 1, PageSize: 1}, types.KnowledgeListFilter{})
			if listErr == nil && knowledgeList != nil && knowledgeList.Total > 0 {
				return errors.NewBadRequestError(
					"Storage backend cannot be changed while the knowledge base contains files; migrate storage first")
			}
		}
		kb.StorageBackendID = &backend.ID
		req.StorageProvider = backend.Provider
	}
	// Legacy provider projection.
	provider := strings.ToLower(strings.TrimSpace(req.StorageProvider))
	if provider == "" {
		provider = "local"
	}
	if !isStorageProviderAllowed(provider) {
		return errors.NewBadRequestError("Storage provider is not allowed by STORAGE_ALLOW_LIST")
	}
	oldProvider := kb.GetStorageProvider()
	if oldProvider == "" {
		oldProvider = "local"
	}
	if oldProvider != provider {
		knowledgeList, err := h.knowledgeService.ListPagedKnowledgeByKnowledgeBaseID(ctx,
			kbID, &types.Pagination{Page: 1, PageSize: 1}, types.KnowledgeListFilter{})
		if err == nil && knowledgeList != nil && knowledgeList.Total > 0 {
			logger.Warn(ctx, "Storage engine changed with existing files, old files may become inaccessible")
		}
	}
	kb.SetStorageProvider(provider)
	return nil
}

// InitializeByKB godoc
// @Summary      Bilgi tabanı yapılandırmasını başlat
// @Description  Bilgi tabanı kimliğine göre tam yapılandırma güncellemesi gerçekleştir
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        kbId     path      string  true  "Bilgi tabanı kimliği"
// @Param        request  body      handler.InitializationRequest  true  "Başlatma isteği"
// @Success      200      {object}  map[string]interface{}  "Başlatma başarılı"
// @Failure      400      {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/initialize/{kbId} [post]
func (h *InitializationHandler) InitializeByKB(c *gin.Context) {
	ctx := c.Request.Context()
	kbIdStr := utils.SanitizeForLog(c.Param("kbId"))

	req, err := h.bindInitializationRequest(ctx, c)
	if err != nil {
		c.Error(err)
		return
	}

	logger.Infof(
		ctx,
		"Starting knowledge base configuration update, kbId: %s, request: %s",
		utils.SanitizeForLog(kbIdStr),
		utils.SanitizeForLog(utils.ToJSON(req)),
	)

	kb, err := h.getKnowledgeBaseForInitialization(ctx, kbIdStr)
	if err != nil {
		c.Error(err)
		return
	}

	if err := h.validateInitializationConfigs(ctx, req); err != nil {
		c.Error(err)
		return
	}

	processedModels, err := h.processInitializationModels(ctx, kb, kbIdStr, req)
	if err != nil {
		c.Error(err)
		return
	}

	h.applyKnowledgeBaseInitialization(kb, req, processedModels)

	if err := h.kbRepository.UpdateKnowledgeBase(ctx, kb); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"kbId": utils.SanitizeForLog(kbIdStr)})
		c.Error(errors.NewInternalServerError(types.LocalizedText(ctx, "Bilgi tabanı yapılandırması güncellenemedi: ", "Failed to update knowledge base configuration: ") + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": types.LocalizedText(ctx, "Bilgi tabanı yapılandırması güncellendi", "Knowledge base configuration updated successfully"),
		"data": gin.H{
			// Through the response DTO, like every other body carrying a
			// model: types.Model marshals api_key in plaintext, and now that
			// the reuse path keeps the stored credential instead of
			// overwriting it, echoing the row would hand back a key the
			// caller never submitted. KnowledgeBase redacts itself.
			"models":         dto.NewModelResponses(ctx, processedModels),
			"knowledge_base": kb,
		},
	})
}

func (h *InitializationHandler) bindInitializationRequest(ctx context.Context, c *gin.Context) (*InitializationRequest, error) {
	var req InitializationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse initialization request", err)
		return nil, errors.NewBadRequestError(err.Error())
	}
	return &req, nil
}

func (h *InitializationHandler) getKnowledgeBaseForInitialization(ctx context.Context, kbIdStr string) (*types.KnowledgeBase, error) {
	kb, err := h.kbService.GetKnowledgeBaseByID(ctx, kbIdStr)
	if err != nil {
		// The repo's not-found sentinel must surface as 404, not 500.
		// Without this, every probe of a stale kb id from the
		// initialization flow burns ops attention with a fake server
		// error. See knowledgebase.go:validateAndGetKnowledgeBase.
		if stderrors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			return nil, errors.NewNotFoundError(types.LocalizedText(ctx, "Bilgi tabanı bulunamadı", "Knowledge base not found"))
		}
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"kbId": utils.SanitizeForLog(kbIdStr)})
		return nil, errors.NewInternalServerError(types.LocalizedText(ctx, "Bilgi tabanı bilgileri alınamadı: ", "Failed to get knowledge base information: ") + err.Error())
	}
	if kb == nil {
		logger.Error(ctx, "Knowledge base not found")
		return nil, errors.NewNotFoundError(types.LocalizedText(ctx, "Bilgi tabanı bulunamadı", "Knowledge base not found"))
	}
	// Initialization rewrites the models the KB points at, and those rows
	// belong to the KB's workspace. A shared-KB editor passes the route's
	// KBAccessWrite guard (which moves execution into that workspace), so
	// without this check it could repoint the owner's models at its own
	// endpoint and key.
	if kb.TenantID != types.CallerFromContext(ctx).TenantID {
		return nil, errors.NewForbiddenError(types.LocalizedText(ctx, "Bilgi tabanını yalnızca ait olduğu çalışma alanı başlatabilir", "Only the knowledge base workspace can initialize it"))
	}
	return kb, nil
}

// canUpdateTenantModels mirrors the PUT /models/:id guard: rewriting a stored
// model changes every KB and agent that uses it, so initializing a KB must not
// let a KB creator do what the model settings page reserves for admins.
func (h *InitializationHandler) canUpdateTenantModels(ctx context.Context) bool {
	if scope, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		return scope.FullAccess || scope.HasCapability(types.APIKeyCapabilityManageModels)
	}
	if types.CallerFromContext(ctx).Role.HasPermission(types.TenantRoleAdmin) || types.IsSystemAdminFromContext(ctx) {
		return true
	}
	// Same rollout switch as the route guards: role checks only log while
	// RBAC enforcement is off.
	return h.config == nil || !h.config.Tenant.IsRBACEnforced()
}

func (h *InitializationHandler) validateInitializationConfigs(ctx context.Context, req *InitializationRequest) error {
	// SSRF validation for all user-supplied BaseURLs
	urlsToCheck := []struct {
		label string
		url   string
	}{
		{"LLM BaseURL", req.LLM.BaseURL},
		{"Embedding BaseURL", req.Embedding.BaseURL},
		{"Rerank BaseURL", req.Rerank.BaseURL},
	}
	if req.Multimodal.VLM != nil {
		urlsToCheck = append(urlsToCheck, struct {
			label string
			url   string
		}{"VLM BaseURL", req.Multimodal.VLM.BaseURL})
	}
	for _, u := range urlsToCheck {
		if u.url != "" {
			if err := utils.ValidateURLForSSRF(u.url); err != nil {
				logger.Warnf(ctx, "SSRF validation failed for %s: %v", u.label, err)
				return errors.NewBadRequestError(utils.FormatSSRFError(u.label, u.url, err, types.LanguageFromContextOrDefault(ctx)))
			}
		}
	}

	if err := h.validateMultimodalConfig(ctx, req); err != nil {
		return err
	}
	if err := validateRerankConfig(ctx, req); err != nil {
		return err
	}
	return validateNodeExtractConfig(ctx, req)
}

func (h *InitializationHandler) validateMultimodalConfig(ctx context.Context, req *InitializationRequest) error {
	if !req.Multimodal.Enabled {
		return nil
	}

	storageType := strings.ToLower(req.Multimodal.StorageType)
	if req.Multimodal.VLM == nil {
		logger.Error(ctx, "Multimodal enabled but missing VLM configuration")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "Çok modlu özellikler için VLM yapılandırması gerekir", "VLM configuration is required for multimodal features"))
	}
	if req.Multimodal.VLM.ModelName == "" || req.Multimodal.VLM.BaseURL == "" {
		logger.Error(ctx, "VLM configuration incomplete")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "VLM yapılandırması eksik", "VLM configuration is incomplete"))
	}

	switch storageType {
	case "cos":
		if req.Multimodal.COS == nil || req.Multimodal.COS.SecretID == "" || req.Multimodal.COS.SecretKey == "" ||
			req.Multimodal.COS.Region == "" || req.Multimodal.COS.BucketName == "" ||
			req.Multimodal.COS.AppID == "" {
			logger.Error(ctx, "COS configuration incomplete")
			return errors.NewBadRequestError(types.LocalizedText(ctx, "COS yapılandırması eksik", "COS configuration is incomplete"))
		}
	case "minio":
		if req.Multimodal.Minio == nil || req.Multimodal.Minio.BucketName == "" ||
			os.Getenv("MINIO_ACCESS_KEY_ID") == "" || os.Getenv("MINIO_SECRET_ACCESS_KEY") == "" {
			logger.Error(ctx, "MinIO configuration incomplete")
			return errors.NewBadRequestError(types.LocalizedText(ctx, "MinIO yapılandırması eksik", "MinIO configuration is incomplete"))
		}
	}
	return nil
}

func validateRerankConfig(ctx context.Context, req *InitializationRequest) error {
	if !req.Rerank.Enabled {
		return nil
	}
	if req.Rerank.ModelName == "" || req.Rerank.BaseURL == "" {
		logger.Error(ctx, "Rerank configuration incomplete")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "Rerank yapılandırması eksik", "Rerank configuration is incomplete"))
	}
	return nil
}

func validateNodeExtractConfig(ctx context.Context, req *InitializationRequest) error {
	if !req.NodeExtract.Enabled {
		return nil
	}
	if strings.ToLower(os.Getenv("NEO4J_ENABLE")) != "true" {
		logger.Error(ctx, "Node Extractor configuration incomplete")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "NEO4J_ENABLE ortam değişkenini doğru yapılandırın", "Configure the NEO4J_ENABLE environment variable correctly"))
	}
	if req.NodeExtract.Text == "" || len(req.NodeExtract.Tags) == 0 {
		logger.Error(ctx, "Node Extractor configuration incomplete")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "Node Extractor yapılandırması eksik", "Node Extractor configuration is incomplete"))
	}
	if len(req.NodeExtract.Nodes) == 0 || len(req.NodeExtract.Relations) == 0 {
		logger.Error(ctx, "Node Extractor configuration incomplete")
		return errors.NewBadRequestError(types.LocalizedText(ctx, "Önce varlıkları ve ilişkileri çıkarın", "Extract entities and relationships first"))
	}
	return nil
}

type modelDescriptor struct {
	modelType     types.ModelType
	name          string
	source        types.ModelSource
	description   string
	baseURL       string
	apiKey        string
	dimension     int
	interfaceType string
}

func buildModelDescriptors(req *InitializationRequest) []modelDescriptor {
	descriptors := []modelDescriptor{
		{
			modelType:   types.ModelTypeKnowledgeQA,
			name:        utils.SanitizeForLog(req.LLM.ModelName),
			source:      types.ModelSource(req.LLM.Source),
			description: "LLM Model for Knowledge QA",
			baseURL:     utils.SanitizeForLog(req.LLM.BaseURL),
			apiKey:      req.LLM.APIKey,
		},
		{
			modelType:   types.ModelTypeEmbedding,
			name:        utils.SanitizeForLog(req.Embedding.ModelName),
			source:      types.ModelSource(req.Embedding.Source),
			description: "Embedding Model",
			baseURL:     utils.SanitizeForLog(req.Embedding.BaseURL),
			apiKey:      req.Embedding.APIKey,
			dimension:   req.Embedding.Dimension,
		},
	}

	if req.Rerank.Enabled {
		descriptors = append(descriptors, modelDescriptor{
			modelType:   types.ModelTypeRerank,
			name:        utils.SanitizeForLog(req.Rerank.ModelName),
			source:      types.ModelSourceRemote,
			description: "Rerank Model",
			baseURL:     utils.SanitizeForLog(req.Rerank.BaseURL),
			apiKey:      req.Rerank.APIKey,
		})
	}

	if req.Multimodal.Enabled && req.Multimodal.VLM != nil {
		descriptors = append(descriptors, modelDescriptor{
			modelType:     types.ModelTypeVLLM,
			name:          utils.SanitizeForLog(req.Multimodal.VLM.ModelName),
			source:        types.ModelSourceRemote,
			description:   "VLM Model",
			baseURL:       utils.SanitizeForLog(req.Multimodal.VLM.BaseURL),
			apiKey:        req.Multimodal.VLM.APIKey,
			interfaceType: req.Multimodal.VLM.InterfaceType,
		})
	}

	return descriptors
}

func (h *InitializationHandler) processInitializationModels(
	ctx context.Context,
	kb *types.KnowledgeBase,
	kbIdStr string,
	req *InitializationRequest,
) ([]*types.Model, error) {
	descriptors := buildModelDescriptors(req)
	var processedModels []*types.Model

	for _, descriptor := range descriptors {
		model := descriptor.toModel()
		// Stamp the KB's tenant before insert: toModel() carries no tenant and
		// modelRepository.GetByID filters on (tenant_id = ? OR is_builtin), so
		// an unstamped row lands at tenant_id = 0 where no tenant — not even
		// the one that just configured the KB — can ever read it back
		// (issue #3333).
		model.TenantID = kb.TenantID
		existingModelID := h.findExistingModelID(kb, descriptor.modelType)

		var existingModel *types.Model
		if existingModelID != "" {
			var err error
			existingModel, err = h.modelService.GetModelByID(ctx, existingModelID)
			if err != nil {
				logger.Warnf(ctx, "Failed to get existing model %s: %v, will create new one", existingModelID, err)
				existingModel = nil
			}
		}

		if existingModel != nil {
			if !h.canUpdateTenantModels(ctx) {
				return nil, errors.NewForbiddenError(types.LocalizedText(ctx, "Mevcut model yapılandırmasını değiştirmek için çalışma alanı yöneticisi izni gerekir", "Workspace admin permission is required to change an existing model"))
			}
			existingModel.Name = model.Name
			existingModel.Source = model.Source
			existingModel.Description = model.Description
			descriptor.applyToStoredParameters(&existingModel.Parameters)
			existingModel.UpdatedAt = time.Now()

			if err := h.modelService.UpdateModel(ctx, existingModel); err != nil {
				logger.ErrorWithFields(ctx, err, map[string]interface{}{
					"model_id": model.ID,
					"kb_id":    kbIdStr,
				})
				return nil, errors.NewInternalServerError(types.LocalizedText(ctx, "Model güncellenemedi: ", "Failed to update model: ") + err.Error())
			}
			processedModels = append(processedModels, existingModel)
			continue
		}

		if err := h.modelService.CreateModel(ctx, model); err != nil {
			logger.ErrorWithFields(ctx, err, map[string]interface{}{
				"model_id": model.ID,
				"kb_id":    kbIdStr,
			})
			return nil, errors.NewInternalServerError(types.LocalizedText(ctx, "Model oluşturulamadı: ", "Failed to create model: ") + err.Error())
		}
		processedModels = append(processedModels, model)
	}

	return processedModels, nil
}

// applyToStoredParameters merges the initialization payload into the
// parameters of a model row that already exists.
//
// The wizard collects four fields (endpoint, key, interface type, embedding
// dimension); everything else on the row — provider, extra_config, custom
// headers, spec, concurrency, context window — was configured in the model
// editor. Assigning toModel()'s parameters wholesale erased all of it, and
// blanked the stored API key whenever the payload carried none, so a KB that
// was merely re-initialized came back with a model nobody could call. Only
// the fields the payload actually carries are written; an empty one means
// "not submitted", not "clear it".
func (descriptor modelDescriptor) applyToStoredParameters(params *types.ModelParameters) {
	if descriptor.baseURL != "" {
		params.BaseURL = descriptor.baseURL
	}
	if descriptor.apiKey != "" {
		params.APIKey = descriptor.apiKey
	}
	if descriptor.interfaceType != "" {
		params.InterfaceType = descriptor.interfaceType
	}
	if descriptor.modelType == types.ModelTypeEmbedding && descriptor.dimension > 0 {
		params.EmbeddingParameters.Dimension = descriptor.dimension
	}
}

func (descriptor modelDescriptor) toModel() *types.Model {
	model := &types.Model{
		Type:        descriptor.modelType,
		Name:        descriptor.name,
		Source:      descriptor.source,
		Description: descriptor.description,
		Parameters: types.ModelParameters{
			BaseURL:       descriptor.baseURL,
			APIKey:        descriptor.apiKey,
			InterfaceType: descriptor.interfaceType,
		},
		IsDefault: false,
		Status:    types.ModelStatusActive,
	}

	if descriptor.modelType == types.ModelTypeEmbedding {
		model.Parameters.EmbeddingParameters = types.EmbeddingParameters{
			Dimension: descriptor.dimension,
		}
	}

	return model
}

func (h *InitializationHandler) findExistingModelID(kb *types.KnowledgeBase, modelType types.ModelType) string {
	switch modelType {
	case types.ModelTypeEmbedding:
		return kb.EmbeddingModelID
	case types.ModelTypeKnowledgeQA:
		return kb.SummaryModelID
	case types.ModelTypeVLLM:
		return kb.VLMConfig.ModelID
	default:
		return ""
	}
}

func (h *InitializationHandler) applyKnowledgeBaseInitialization(
	kb *types.KnowledgeBase,
	req *InitializationRequest,
	processedModels []*types.Model,
) {
	embeddingModelID, llmModelID, vlmModelID := extractModelIDs(processedModels)

	kb.SummaryModelID = llmModelID
	kb.EmbeddingModelID = embeddingModelID

	kb.ChunkingConfig = types.ChunkingConfig{
		ChunkSize:    req.DocumentSplitting.ChunkSize,
		ChunkOverlap: req.DocumentSplitting.ChunkOverlap,
		Separators:   req.DocumentSplitting.Separators,
	}

	if req.Multimodal.Enabled {
		kb.VLMConfig = types.VLMConfig{
			Enabled: req.Multimodal.Enabled,
			ModelID: vlmModelID,
		}
		switch req.Multimodal.StorageType {
		case "cos":
			if req.Multimodal.COS != nil {
				kb.SetStorageProvider("cos")
				// Legacy: also write to cos_config for backward compat with old code paths
				kb.StorageConfig = types.StorageConfig{
					Provider:   req.Multimodal.StorageType,
					BucketName: req.Multimodal.COS.BucketName,
					AppID:      req.Multimodal.COS.AppID,
					PathPrefix: req.Multimodal.COS.PathPrefix,
					SecretID:   req.Multimodal.COS.SecretID,
					SecretKey:  req.Multimodal.COS.SecretKey,
					Region:     req.Multimodal.COS.Region,
				}
			}
		case "minio":
			if req.Multimodal.Minio != nil {
				kb.SetStorageProvider("minio")
				// Legacy: also write to cos_config for backward compat with old code paths
				kb.StorageConfig = types.StorageConfig{
					Provider:   req.Multimodal.StorageType,
					BucketName: req.Multimodal.Minio.BucketName,
					PathPrefix: req.Multimodal.Minio.PathPrefix,
					SecretID:   os.Getenv("MINIO_ACCESS_KEY_ID"),
					SecretKey:  os.Getenv("MINIO_SECRET_ACCESS_KEY"),
				}
			}
		}
	} else {
		kb.VLMConfig = types.VLMConfig{}
		kb.SetStorageProvider("")
		kb.StorageConfig = types.StorageConfig{}
	}

	if req.NodeExtract.Enabled {
		kb.ExtractConfig = &types.ExtractConfig{
			Text:      req.NodeExtract.Text,
			Tags:      req.NodeExtract.Tags,
			Nodes:     make([]*types.GraphNode, 0),
			Relations: make([]*types.GraphRelation, 0),
		}
		for _, rnode := range req.NodeExtract.Nodes {
			node := &types.GraphNode{
				Name:       rnode.Name,
				Attributes: rnode.Attributes,
			}
			kb.ExtractConfig.Nodes = append(kb.ExtractConfig.Nodes, node)
		}
		for _, relation := range req.NodeExtract.Relations {
			kb.ExtractConfig.Relations = append(kb.ExtractConfig.Relations, &types.GraphRelation{
				Node1: relation.Node1,
				Node2: relation.Node2,
				Type:  relation.Type,
			})
		}
	}
}

func extractModelIDs(processedModels []*types.Model) (embeddingModelID, llmModelID, vlmModelID string) {
	for _, model := range processedModels {
		if model == nil {
			continue
		}
		switch model.Type {
		case types.ModelTypeEmbedding:
			embeddingModelID = model.ID
		case types.ModelTypeKnowledgeQA:
			llmModelID = model.ID
		case types.ModelTypeVLLM:
			vlmModelID = model.ID
		}
	}
	return
}

// GetCurrentConfigByKB godoc
// @Summary      Bilgi tabanı yapılandırmasını al
// @Description  Bilgi tabanı kimliğine göre geçerli yapılandırma bilgilerini al
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        kbId  path      string  true  "Bilgi tabanı kimliği"
// @Success      200   {object}  map[string]interface{}  "Yapılandırma bilgileri"
// @Failure      404   {object}  errors.AppError         "Bilgi tabanı bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/config/{kbId} [get]
func (h *InitializationHandler) GetCurrentConfigByKB(c *gin.Context) {
	ctx := c.Request.Context()
	kbIdStr := utils.SanitizeForLog(c.Param("kbId"))

	logger.Info(ctx, "Getting configuration for knowledge base")

	// Belirtilen bilgi tabanı bilgilerini al
	kb, err := h.kbService.GetKnowledgeBaseByID(ctx, kbIdStr)
	if err != nil {
		// Mirror getKnowledgeBaseForInitialization above: missing /
		// cross-tenant kb ids are 404, not 500.
		if stderrors.Is(err, repository.ErrKnowledgeBaseNotFound) {
			c.Error(errors.NewNotFoundError(types.LocalizedText(ctx, "Bilgi tabanı bulunamadı", "Knowledge base not found")))
			return
		}
		logger.Error(ctx, "Failed to get knowledge base", err)
		c.Error(errors.NewInternalServerError(types.LocalizedText(ctx, "Bilgi tabanı bilgileri alınamadı: ", "Failed to get knowledge base information: ") + err.Error()))
		return
	}

	if kb == nil {
		logger.Error(ctx, "Knowledge base not found")
		c.Error(errors.NewNotFoundError(types.LocalizedText(ctx, "Bilgi tabanı bulunamadı", "Knowledge base not found")))
		return
	}

	// Bilgi tabanının model kimliğine göre belirli modeli al
	var models []*types.Model
	modelIDs := []string{
		kb.EmbeddingModelID,
		kb.SummaryModelID,
		kb.VLMConfig.ModelID,
	}

	for _, modelID := range modelIDs {
		if modelID != "" {
			model, err := h.modelService.GetModelByID(ctx, modelID)
			if err != nil {
				logger.Warn(ctx, "Failed to get model", err)
				// Model mevcut değilse veya alınamazsa diğer modelleri işlemeye devam et
				continue
			}
			if model != nil {
				models = append(models, model)
			}
		}
	}

	// Bilgi tabanında dosya olup olmadığını kontrol et
	knowledgeList, err := h.knowledgeService.ListPagedKnowledgeByKnowledgeBaseID(ctx,
		kbIdStr, &types.Pagination{
			Page:     1,
			PageSize: 1,
		}, types.KnowledgeListFilter{})
	hasFiles := err == nil && knowledgeList != nil && knowledgeList.Total > 0

	// Yapılandırma yanıtını oluştur
	config := h.buildConfigResponse(ctx, models, kb, hasFiles)

	logger.Info(ctx, "Knowledge base configuration retrieved successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// buildConfigResponse yapılandırma yanıt verilerini oluşturur
func (h *InitializationHandler) buildConfigResponse(ctx context.Context, models []*types.Model,
	kb *types.KnowledgeBase, hasFiles bool,
) map[string]interface{} {
	config := map[string]interface{}{
		"hasFiles": hasFiles,
	}
	// Integration details describe the owning workspace's infrastructure. A
	// share receiver — even an admin of its own workspace — only learns
	// whether credentials are configured.
	ownWorkspace := kb != nil && kb.TenantID == types.CallerFromContext(ctx).TenantID
	includeIntegrationDetail := ownWorkspace && dto.CanViewIntegrationSecrets(ctx)

	// Modelleri türe göre grupla
	for _, model := range models {
		if model == nil {
			continue
		}
		// Hide sensitive information for builtin models and viewers.
		baseURL := model.Parameters.BaseURL
		if model.IsBuiltin || !includeIntegrationDetail {
			baseURL = ""
		}

		switch model.Type {
		case types.ModelTypeKnowledgeQA:
			config["llm"] = map[string]interface{}{
				"source":    string(model.Source),
				"modelName": model.Name,
				"baseUrl":   baseURL,
				"credentials": map[string]bool{
					"apiKey": model.Parameters.APIKey != "" && !model.IsBuiltin,
				},
			}
		case types.ModelTypeEmbedding:
			config["embedding"] = map[string]interface{}{
				"source":    string(model.Source),
				"modelName": model.Name,
				"baseUrl":   baseURL,
				"dimension": model.Parameters.EmbeddingParameters.Dimension,
				"credentials": map[string]bool{
					"apiKey": model.Parameters.APIKey != "" && !model.IsBuiltin,
				},
			}
		case types.ModelTypeRerank:
			config["rerank"] = map[string]interface{}{
				"enabled":   true,
				"modelName": model.Name,
				"baseUrl":   baseURL,
				"credentials": map[string]bool{
					"apiKey": model.Parameters.APIKey != "" && !model.IsBuiltin,
				},
			}
		case types.ModelTypeVLLM:
			if config["multimodal"] == nil {
				config["multimodal"] = map[string]interface{}{
					"enabled": true,
				}
			}
			multimodal := config["multimodal"].(map[string]interface{})
			multimodal["vlm"] = map[string]interface{}{
				"modelName":     model.Name,
				"baseUrl":       baseURL,
				"interfaceType": model.Parameters.InterfaceType,
				"modelId":       model.ID,
				"credentials": map[string]bool{
					"apiKey": model.Parameters.APIKey != "" && !model.IsBuiltin,
				},
			}
		}
	}

	// Çok modluluğun etkin olup olmadığını belirle: VLM model kimliği veya depolama yapılandırması var (yeni ve eski alanlarla uyumlu)
	storageProvider := kb.GetStorageProvider()
	hasMultimodal := (kb.VLMConfig.IsEnabled() ||
		kb.StorageConfig.SecretID != "" || kb.StorageConfig.BucketName != "" ||
		(storageProvider != "" && storageProvider != "local"))
	if config["multimodal"] == nil {
		config["multimodal"] = map[string]interface{}{
			"enabled": hasMultimodal,
		}
	} else {
		config["multimodal"].(map[string]interface{})["enabled"] = hasMultimodal
	}
	if kb.VLMConfig.DescriptionLanguage != "" || kb.VLMConfig.CustomInstructions != "" {
		if config["multimodal"] == nil {
			config["multimodal"] = map[string]interface{}{
				"enabled": hasMultimodal,
			}
		}
		multimodal := config["multimodal"].(map[string]interface{})
		if kb.VLMConfig.DescriptionLanguage != "" {
			multimodal["descriptionLanguage"] = kb.VLMConfig.DescriptionLanguage
		}
		if kb.VLMConfig.CustomInstructions != "" {
			multimodal["customInstructions"] = kb.VLMConfig.CustomInstructions
		}
	}

	// Rerank modeli yoksa rerank'i disabled olarak ayarla
	if config["rerank"] == nil {
		config["rerank"] = map[string]interface{}{
			"enabled":   false,
			"modelName": "",
			"baseUrl":   "",
			"credentials": map[string]bool{
				"apiKey": false,
			},
		}
	}

	// Bilgi tabanının belge bölme yapılandırmasını ekle
	if kb != nil {
		ds := map[string]interface{}{
			"chunkSize":    kb.ChunkingConfig.ChunkSize,
			"chunkOverlap": kb.ChunkingConfig.ChunkOverlap,
			"separators":   kb.ChunkingConfig.Separators,
		}
		if kb.ChunkingConfig.Strategy != "" {
			ds["strategy"] = kb.ChunkingConfig.Strategy
		}
		if kb.ChunkingConfig.TokenLimit > 0 {
			ds["tokenLimit"] = kb.ChunkingConfig.TokenLimit
		}
		if len(kb.ChunkingConfig.Languages) > 0 {
			ds["languages"] = kb.ChunkingConfig.Languages
		}
		if kb.ChunkingConfig.TableMetadataInstructions != "" {
			ds["tableMetadataInstructions"] = kb.ChunkingConfig.TableMetadataInstructions
		}
		config["documentSplitting"] = ds

		// Çok modlu depolama yapılandırması bilgilerini ekle (önce yeni alanları oku, eski cos_config ile uyumlu)
		effectiveProvider := kb.GetStorageProvider()
		if kb.StorageConfig.SecretID != "" || (effectiveProvider != "" && effectiveProvider != "local") {
			if config["multimodal"] == nil {
				config["multimodal"] = map[string]interface{}{
					"enabled": true,
				}
			}
			multimodal := config["multimodal"].(map[string]interface{})
			multimodal["storageType"] = effectiveProvider
			switch effectiveProvider {
			case "cos":
				multimodal["cos"] = map[string]interface{}{
					"region":     kb.StorageConfig.Region,
					"bucketName": kb.StorageConfig.BucketName,
					"appId":      kb.StorageConfig.AppID,
					"pathPrefix": kb.StorageConfig.PathPrefix,
					"credentials": map[string]bool{
						"secretId":  kb.StorageConfig.SecretID != "",
						"secretKey": kb.StorageConfig.SecretKey != "",
					},
				}
			case "minio":
				multimodal["minio"] = map[string]interface{}{
					"bucketName": kb.StorageConfig.BucketName,
					"pathPrefix": kb.StorageConfig.PathPrefix,
				}
			}
			if !ownWorkspace {
				// Bucket locations are the owner's infrastructure too.
				for _, provider := range []string{"cos", "minio"} {
					if detail, ok := multimodal[provider].(map[string]interface{}); ok {
						for _, field := range []string{"region", "bucketName", "appId", "pathPrefix"} {
							delete(detail, field)
						}
					}
				}
			}
		}
	}

	if kb.ExtractConfig != nil {
		nodeExtract := map[string]interface{}{
			"enabled":   kb.ExtractConfig.Enabled,
			"text":      kb.ExtractConfig.Text,
			"tags":      kb.ExtractConfig.Tags,
			"nodes":     kb.ExtractConfig.Nodes,
			"relations": kb.ExtractConfig.Relations,
		}
		if kb.ExtractConfig.CustomInstructions != "" {
			nodeExtract["customInstructions"] = kb.ExtractConfig.CustomInstructions
		}
		config["nodeExtract"] = nodeExtract
	} else {
		config["nodeExtract"] = map[string]interface{}{
			"enabled": false,
		}
	}

	if kb.QuestionGenerationConfig != nil {
		config["questionGeneration"] = map[string]interface{}{
			"enabled":            kb.QuestionGenerationConfig.Enabled,
			"questionCount":      kb.QuestionGenerationConfig.QuestionCount,
			"customInstructions": kb.QuestionGenerationConfig.CustomInstructions,
		}
	} else {
		config["questionGeneration"] = map[string]interface{}{
			"enabled": false,
		}
	}

	return config
}

// ModelTestRequest birleşik "bağlantıyı test et" istek gövdesi.
//
// Dört modelin (chat/embedding/rerank/asr) test arayüzleri aynı yapıyı paylaşır; böylece:
//   - Ön yüzün yalnızca tek bir form → arka uç eşlemesi sürdürmesi yeterlidir.
//   - Arka uç isteği doğrudan *types.Model'e dönüştürebilir, ardından her paketin ConfigFromModel işlevini çağırabilir,
//     ve üretim yoluyla (service.modelService.GetXxxModel) tamamen aynı birleştirme akışını kullanır,
//     böylece geçmişte her test uç noktasında Config'i elle oluşturmak için gereken şablon kod tamamen ortadan kalkar.
//
// Tüm provider/model ortak alanları burada merkezi olarak bildirilir; gelecekte yeni alanlar eklenirse (örneğin mevcut
// custom_headers), yalnızca tek bir yerin değiştirilmesi yeterlidir; üretim ve test yolları aynı anda etkili olur.
type ModelTestRequest struct {
	Spec                      *types.ModelSpecOverride `json:"spec,omitempty"`
	Source                    string                   `json:"source"` // Boş olduğunda gerektiğinde varsayılan olarak "remote" kullanılır
	ModelName                 string                   `json:"modelName" binding:"required"`
	BaseURL                   string                   `json:"baseUrl"`
	APIKey                    string                   `json:"apiKey"`
	Provider                  string                   `json:"provider"`
	InterfaceType             string                   `json:"interfaceType,omitempty"`
	Dimension                 int                      `json:"dimension,omitempty"`
	SupportsDimensionOverride bool                     `json:"supportsDimensionOverride,omitempty"`
	CustomHeaders             map[string]string        `json:"customHeaders,omitempty"`
	ExtraConfig               map[string]string        `json:"extraConfig,omitempty"`
	// AppSecret, ikinci bir anahtar bölümü gerektiren LKEAP / Volcengine Rerank gibi durumlar için kullanılır (model Parameters.AppSecret'e karşılık gelir).
	AppSecret string `json:"appSecret,omitempty"`
	// ModelID, when set, instructs the handler to substitute any missing
	// secrets (APIKey, AppSecret via ExtraConfig) from the stored model
	// record before assembling the test client. This lets the "Test
	// connection" button work on existing models without making the
	// frontend reload — and ship — the plaintext API key. Other fields
	// (BaseURL, ModelName, etc.) on this request still override the
	// stored values, so a user can validate a new endpoint against the
	// existing credentials in one click.
	ModelID string `json:"modelId,omitempty"`
}

// fillSecretsFromStoredModel mutates req in place: if req.ModelID is set
// and a secret field on the request is empty, the corresponding value from
// the stored (and decrypted) model is copied in. The stored ExtraConfig is
// filled in as well when the request does not carry one — provider-specific
// settings (thinking_control, api_version, remote_model_name, ...) must
// apply to the connection test exactly as they apply to real traffic, and
// the frontend only sends extraConfig when the user actively edits it.
// Non-empty request values are always preferred — they represent the user
// actively typing a new key they want to verify. Missing or inaccessible
// model is treated as a no-op (the connection test will fail downstream
// with a clearer "missing apiKey" error than we could produce here).
func (h *InitializationHandler) fillSecretsFromStoredModel(ctx context.Context, req *ModelTestRequest) {
	if req == nil || req.ModelID == "" {
		return
	}
	// A request that already carries every secret needs no lookup — but a
	// secret stored in extra_config (LKEAP / Volcengine secret_key) is
	// redacted by GET, so "extraConfig is present" does not mean it is
	// complete.
	if req.APIKey != "" && req.AppSecret != "" && req.ExtraConfig != nil && req.Spec != nil &&
		dto.HasAllSecretExtras(req.Provider, req.BaseURL, req.ExtraConfig) {
		return
	}
	stored, err := h.modelService.GetModelByID(ctx, req.ModelID)
	if err != nil || stored == nil {
		logger.Warnf(ctx, "test-connection: stored model %s not found, leaving secrets empty: %v",
			utils.SanitizeForLog(req.ModelID), err)
		return
	}
	if req.Spec == nil && req.Provider == stored.Parameters.Provider {
		req.Spec = stored.Parameters.Spec
	}
	if req.APIKey == "" {
		req.APIKey = stored.Parameters.APIKey
	}
	if req.AppSecret == "" {
		req.AppSecret = stored.Parameters.AppSecret
	}
	// Same contract as PUT /models/{id}: an absent or masked secret extra
	// falls back to the stored value, a real one the user just typed wins —
	// and a test against a different vendor gets no stored credential, which
	// belongs to the integration the row is being moved away from.
	req.ExtraConfig = dto.PreserveStoredSecretExtras(
		stored.Parameters.ExtraConfig, req.ExtraConfig,
		dto.VendorRef{Provider: stored.Parameters.Provider, BaseURL: stored.Parameters.BaseURL},
		dto.VendorRef{Provider: req.Provider, BaseURL: req.BaseURL},
	)
}

// RemoteModelCheckRequest eski swagger tanımlarıyla uyumludur.
//
// Deprecated: Oluşturulmuş API belgelerini bozmamak için korunmuştur; yeni kodda doğrudan ModelTestRequest kullanın.
type RemoteModelCheckRequest = ModelTestRequest

// decryptModelAppSecret, model Parameters içindeki AppSecret'i çözer (modelService davranışıyla tutarlı).
func decryptModelAppSecret(encrypted string) string {
	if encrypted == "" {
		return encrypted
	}
	if key := utils.GetAESKey(); key != nil {
		if plain, err := utils.DecryptAESGCM(encrypted, key); err == nil {
			return plain
		}
	}
	return encrypted
}

// buildTestModel, bağlantı testi isteğini geçici bir *types.Model'e dönüştürür (veritabanına kaydetmez),
// ConfigFromModel tarafından kullanılır. source boş olduğunda defaultSource ile yedeklenir (chat/rerank/asr
// varsayılan olarak remote kullanır; embedding, ön uçtan iletilen source değerine göre belirlenir).
func (h *InitializationHandler) buildTestModel(
	req *ModelTestRequest, modelType types.ModelType, defaultSource types.ModelSource,
) *types.Model {
	source := types.ModelSource(strings.ToLower(req.Source))
	if source == "" {
		source = defaultSource
	}
	return &types.Model{
		Name:   req.ModelName,
		Type:   modelType,
		Source: source,
		Parameters: types.ModelParameters{
			BaseURL:       req.BaseURL,
			APIKey:        req.APIKey,
			AppSecret:     req.AppSecret,
			Provider:      req.Provider,
			InterfaceType: req.InterfaceType,
			ExtraConfig:   req.ExtraConfig,
			Spec:          req.Spec,
			CustomHeaders: req.CustomHeaders,
			EmbeddingParameters: types.EmbeddingParameters{
				Dimension:                 req.Dimension,
				TruncatePromptTokens:      256,
				SupportsDimensionOverride: req.SupportsDimensionOverride,
			},
		},
	}
}

// CheckRemoteModel godoc
// @Summary      Uzak modeli kontrol et
// @Description  Uzak API model bağlantısının normal olup olmadığını kontrol et
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      RemoteModelCheckRequest  true  "Model kontrol isteği"
// @Success      200      {object}  map[string]interface{}   "Kontrol sonucu"
// @Failure      400      {object}  errors.AppError          "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/remote/check [post]
func (h *InitializationHandler) CheckRemoteModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Checking remote model connection")

	var req ModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse remote model check request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	h.fillSecretsFromStoredModel(ctx, &req)

	if req.ModelName == "" || req.BaseURL == "" {
		logger.Error(ctx, "Model name and base URL are required")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Model adı ve Base URL boş olamaz", "Model name and Base URL are required")))
		return
	}

	if err := utils.ValidateURLForSSRF(req.BaseURL); err != nil {
		logger.Warnf(ctx, "SSRF validation failed for remote model BaseURL: %v", err)
		c.Error(errors.NewBadRequestError(utils.FormatSSRFError("Base URL", req.BaseURL, err, types.LanguageFromContextOrDefault(ctx))))
		return
	}
	model := h.buildTestModel(&req, types.ModelTypeKnowledgeQA, types.ModelSourceRemote)
	available, message := h.checkChatModelConnection(ctx, model, "", decryptModelAppSecret(model.Parameters.AppSecret))

	logger.Infof(ctx, "Remote model check completed, available: %v, message: %s", available, message)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available": available,
			"message":   message,
		},
	})
}

// TestEmbeddingModel godoc
// @Summary      Embedding modelini test et
// @Description  Embedding arayüzünün kullanılabilir olup olmadığını test et ve vektör boyutunu döndür
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      handler.ModelTestRequest  true  "Embedding test isteği"
// @Success      200      {object}  map[string]interface{}  "Test sonucu"
// @Failure      400      {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/embedding/test [post]
func (h *InitializationHandler) TestEmbeddingModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Testing embedding model connectivity and functionality")

	var req ModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse embedding test request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	h.fillSecretsFromStoredModel(ctx, &req)
	if req.Source == "" {
		req.Source = string(types.ModelSourceRemote)
	}

	if req.BaseURL != "" {
		if err := utils.ValidateURLForSSRF(req.BaseURL); err != nil {
			logger.Warnf(ctx, "SSRF validation failed for embedding BaseURL: %v", err)
			c.Error(errors.NewBadRequestError(utils.FormatSSRFError("Base URL", req.BaseURL, err, types.LanguageFromContextOrDefault(ctx))))
			return
		}
	}

	// Alibaba Cloud çok modlu Embedding modeli henüz desteklenmiyor
	if strings.ToLower(req.Provider) == "aliyun" {
		modelNameLower := strings.ToLower(req.ModelName)
		if strings.Contains(modelNameLower, "vision") || strings.Contains(modelNameLower, "multimodal") {
			logger.Infof(ctx, "Aliyun multimodal embedding model not supported: %s", req.ModelName)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"available": false,
					"message":   types.LocalizedText(ctx, "Aliyun çok modlu Embedding modelleri henüz desteklenmiyor; text-embedding-v4 gibi bir metin Embedding modeli kullanın", "Aliyun multimodal Embedding models are not supported yet; use a text Embedding model such as text-embedding-v4"),
					"dimension": 0,
				},
			})
			return
		}
	}

	model := h.buildTestModel(&req, types.ModelTypeEmbedding, types.ModelSourceRemote)
	emb, err := embedding.NewEmbedder(embedding.ConfigFromModel(model, "", decryptModelAppSecret(model.Parameters.AppSecret)), h.pooler)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"model": utils.SanitizeForLog(req.ModelName)})
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{`available`: false, `message`: fmt.Sprintf(types.LocalizedText(ctx, "Embedder oluşturulamadı: %v", "Failed to create Embedder: %v"), err), `dimension`: 0},
		})
		return
	}

	vec, err := emb.Embed(ctx, "hello")
	if err != nil {
		logger.Error(ctx, "Failed to call embedder", err)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    gin.H{`available`: false, `message`: fmt.Sprintf(types.LocalizedText(ctx, "Embedding çağrısı başarısız: %v", "Embedding call failed: %v"), err), `dimension`: 0},
		})
		return
	}

	logger.Infof(ctx, "Embedding test succeeded, dimension: %d", len(vec))
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{`available`: true, `message`: fmt.Sprintf(types.LocalizedText(ctx, "Test başarılı, vektör boyutu=%d", "Test succeeded, vector dimension=%d"), len(vec)), `dimension`: len(vec)},
	})
}

// classifyConnectionError maps an upstream error string to a short
// localized hint. Callers should always combine the hint
// with the raw error message (e.g. fmt.Sprintf("%s：%v", hint, err)) so
// the operator can still see what URL / response body the SDK actually
// got — the hint is for "where to start looking", the raw error is for
// "what actually happened".
func classifyConnectionError(ctx context.Context, errMsg string) string {
	switch {
	case strings.Contains(errMsg, "401") || strings.Contains(errMsg, "unauthorized"):
		return types.LocalizedText(ctx, "Kimlik doğrulama başarısız; API Key değerini kontrol edin", "Authentication failed; check the API Key")
	case strings.Contains(errMsg, "403") || strings.Contains(errMsg, "forbidden"):
		return types.LocalizedText(ctx, "Yetki yetersiz; API Key izinlerini kontrol edin", "Insufficient permissions; check the API Key permissions")
	case strings.Contains(errMsg, "404") || strings.Contains(errMsg, "not found"):
		return types.LocalizedText(ctx, "API uç noktası bulunamadı; Base URL değerini kontrol edin", "API endpoint not found; check the Base URL")
	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "context deadline exceeded"):
		return types.LocalizedText(ctx, "Bağlantı zaman aşımına uğradı; ağı kontrol edin", "Connection timed out; check the network")
	case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dial tcp"):
		return types.LocalizedText(ctx, "Sunucuya bağlanılamadı; Base URL değerini kontrol edin", "Could not connect to the server; check the Base URL")
	default:
		return types.LocalizedText(ctx, "Bağlantı başarısız", "Connection failed")
	}
}

// checkChatModelConnection, bağlantıyı ve kimlik doğrulamayı test etmek için chat modülüyle en küçük çağrıyı yapar.
// Üretim yoluyla tamamen aynı ConfigFromModel → NewChat akışını izler; bu nedenle CustomHeaders,
// ExtraConfig, Provider ve diğer alanlar doğru şekilde iletilir.
func (h *InitializationHandler) checkChatModelConnection(
	ctx context.Context, model *types.Model, appID, appSecret string,
) (bool, string) {
	chatInstance, err := chat.NewChat(chat.ConfigFromModel(model, appID, appSecret))
	if err != nil {
		return false, fmt.Sprintf(types.LocalizedText(ctx, "Sohbet örneği oluşturulamadı: %v", "Failed to create chat instance: %v"), err)
	}

	testMessages := []chat.Message{{Role: "user", Content: "test"}}
	testOptions := &chat.ChatOptions{
		MaxTokens: 1,
		Thinking:  &[]bool{false}[0], // for dashscope.aliyuncs qwen3-32b
	}

	_, err = chatInstance.Chat(ctx, testMessages, testOptions)
	if err != nil {
		errMsg := err.Error()
		// 400 = endpoint reachable + auth ok, just a parameter mismatch
		// (e.g. max_tokens vs max_completion_tokens). Treat as success.
		if strings.Contains(errMsg, "status code: 400") {
			return true, types.LocalizedText(ctx, "Bağlantı başarılı; model kullanılabilir", "Connection successful; model is available")
		}
		// For every other failure mode we surface a human-readable hint
		// AND the upstream error verbatim. Swallowing the underlying
		// message used to hide things like the actual URL the SDK
		// tried, response body, etc. — making remote debugging nearly
		// impossible. Format: "<hint>：<raw err>".
		return false, fmt.Sprintf("%s: %v", classifyConnectionError(ctx, errMsg), err)
	}

	// Bağlantı başarılı, model kullanılabilir
	return true, types.LocalizedText(ctx, "Bağlantı başarılı; model kullanılabilir", "Connection successful; model is available")
}

// checkRerankModelConnection, bağlantıyı ve kimlik doğrulamayı test etmek için rerank modülüyle en küçük çağrıyı yapar.
// Üretim yoluyla ConfigFromModel'i paylaşır; tüm alanlar (CustomHeaders vb.) iletilir.
func (h *InitializationHandler) checkRerankModelConnection(
	ctx context.Context, model *types.Model, appID, appSecret string,
) (bool, string) {
	reranker, err := rerank.NewReranker(rerank.ConfigFromModel(model, appID, appSecret))
	if err != nil {
		return false, fmt.Sprintf(types.LocalizedText(ctx, "Reranker oluşturulamadı: %v", "Failed to create Reranker: %v"), err)
	}

	results, err := reranker.Rerank(ctx, "ping", []string{"pong"})
	if err != nil {
		return false, fmt.Sprintf(types.LocalizedText(ctx, "Yeniden sıralama testi başarısız: %v", "Rerank test failed: %v"), err)
	}
	if len(results) > 0 {
		return true, fmt.Sprintf(types.LocalizedText(ctx, "Yeniden sıralama çalışıyor; %d sonuç döndü", "Reranking works; returned %d results"), len(results))
	}
	return false, types.LocalizedText(ctx, "Yeniden sıralama arayüzüne bağlanıldı, ancak sonuç dönmedi", "Rerank endpoint connected but returned no results")
}

// CheckRerankModel godoc
// @Summary      Rerank modelini kontrol et
// @Description  Rerank model bağlantısının ve işlevselliğinin normal olup olmadığını kontrol et
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      handler.ModelTestRequest  true  "Rerank kontrol isteği"
// @Success      200      {object}  map[string]interface{}  "Kontrol sonucu"
// @Failure      400      {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/rerank/check [post]
func (h *InitializationHandler) CheckRerankModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Checking rerank model connection and functionality")

	var req ModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse rerank model check request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	h.fillSecretsFromStoredModel(ctx, &req)

	if req.ModelName == "" || req.BaseURL == "" {
		logger.Error(ctx, "Model name and base URL are required")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Model adı ve Base URL boş olamaz", "Model name and Base URL are required")))
		return
	}

	if err := utils.ValidateURLForSSRF(req.BaseURL); err != nil {
		logger.Warnf(ctx, "SSRF validation failed for rerank BaseURL: %v", err)
		c.Error(errors.NewBadRequestError(utils.FormatSSRFError("Base URL", req.BaseURL, err, types.LanguageFromContextOrDefault(ctx))))
		return
	}

	model := h.buildTestModel(&req, types.ModelTypeRerank, types.ModelSourceRemote)
	available, message := h.checkRerankModelConnection(ctx, model, "", decryptModelAppSecret(model.Parameters.AppSecret))

	logger.Infof(ctx, "Rerank model check completed, available: %v, message: %s", available, message)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available": available,
			"message":   message,
		},
	})
}

// CheckASRModel godoc
// @Summary      ASR modelini kontrol et
// @Description  ASR (ses tanıma) model bağlantısının normal olup olmadığını, bir sessiz ses kaydı göndererek /v1/audio/transcriptions uç noktasını test eder
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      handler.ModelTestRequest  true  "ASR kontrol isteği"
// @Success      200      {object}  map[string]interface{}  "Kontrol sonucu"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/asr/check [post]
func (h *InitializationHandler) CheckASRModel(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Checking ASR model connection")

	var req ModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to parse ASR model check request", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	h.fillSecretsFromStoredModel(ctx, &req)

	if req.ModelName == "" || req.BaseURL == "" {
		logger.Error(ctx, "Model name and base URL are required for ASR check")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Model adı ve Base URL boş olamaz", "Model name and Base URL are required")))
		return
	}

	if err := utils.ValidateURLForSSRF(req.BaseURL); err != nil {
		logger.Warnf(ctx, "SSRF validation failed for ASR BaseURL: %v", err)
		c.Error(errors.NewBadRequestError(utils.FormatSSRFError("Base URL", req.BaseURL, err, types.LanguageFromContextOrDefault(ctx))))
		return
	}

	// Birleşik oluşturucu kullanılarak test için *types.Model oluşturulur (ASR ek kimlik bilgileri gerektirmez),
	// /v1/audio/transcriptions uç noktasının erişilebilir olduğunu doğrulamak için çok kısa bir sessiz WAV ses kaydı gönderilir.
	model := h.buildTestModel(&req, types.ModelTypeASR, types.ModelSourceRemote)
	asrInstance, err := asr.NewASR(asr.ConfigFromModel(model))
	if err != nil {
		logger.Errorf(ctx, "Failed to create ASR instance for check: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"available": false,
				"message":   fmt.Sprintf(types.LocalizedText(ctx, "ASR örneği oluşturulamadı: %v", "Failed to create ASR instance: %v"), err),
			},
		})
		return
	}

	res, err := asrInstance.Transcribe(ctx, assets.ASRTestWAV, "asr_test.wav")
	var text string
	if res != nil {
		text = res.Text
	}
	available := true
	message := types.LocalizedText(ctx, "ASR bağlantısı başarılı", "ASR connected successfully")

	if err != nil {
		errMsg := err.Error()
		// Always include the raw upstream error after the hint — see
		// classifyConnectionError comment for rationale.
		switch {
		case strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Unauthorized") || strings.Contains(errMsg, "authentication"):
			available = false
			message = fmt.Sprintf(types.LocalizedText(ctx, "Kimlik doğrulama başarısız; API Key değerini kontrol edin: %s", "Authentication failed; check the API Key: %s"), errMsg)
		case strings.Contains(errMsg, "404") || strings.Contains(errMsg, "Not Found"):
			available = false
			message = fmt.Sprintf(types.LocalizedText(ctx, "API uç noktası bulunamadı; Base URL değerini kontrol edin: %s", "API endpoint not found; check the Base URL: %s"), errMsg)
		case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dial tcp"):
			available = false
			message = fmt.Sprintf(types.LocalizedText(ctx, "Sunucuya bağlanılamadı; Base URL değerini kontrol edin: %s", "Could not connect to the server; check the Base URL: %s"), errMsg)
		case strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found"):
			available = false
			message = fmt.Sprintf(types.LocalizedText(ctx, "Model bulunamadı; model adını kontrol edin: %s", "Model not found; check the model name: %s"), errMsg)
		default:
			logger.Infof(ctx, "ASR check got non-fatal error (endpoint reachable): %v", err)
			available = true
			message = fmt.Sprintf(types.LocalizedText(ctx, "ASR uç noktasına erişildi (kritik olmayan hata: %s)", "ASR endpoint reached (non-fatal error: %s)"), errMsg)
		}
	} else if text != "" {
		message = fmt.Sprintf(types.LocalizedText(ctx, "ASR bağlantısı başarılı; transkripsiyon: %s", "ASR connected successfully; transcription: %s"), text)
	}

	logger.Infof(ctx, "ASR model check completed, available: %v, message: %s", available, message)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available": available,
			"message":   message,
		},
	})
}

// Form verilerini ayrıştırmak için struct kullanılır
type testMultimodalForm struct {
	VLMModel         string `form:"vlm_model"`
	VLMBaseURL       string `form:"vlm_base_url"`
	VLMAPIKey        string `form:"vlm_api_key"`
	VLMInterfaceType string `form:"vlm_interface_type"`

	StorageType string `form:"storage_type"`

	// COS yapılandırması
	COSSecretID   string `form:"cos_secret_id"`
	COSSecretKey  string `form:"cos_secret_key"`
	COSRegion     string `form:"cos_region"`
	COSBucketName string `form:"cos_bucket_name"`
	COSAppID      string `form:"cos_app_id"`
	COSPathPrefix string `form:"cos_path_prefix"`

	// MinIO yapılandırması (depolama minio olduğunda)
	MinioBucketName string `form:"minio_bucket_name"`
	MinioPathPrefix string `form:"minio_path_prefix"`

	// Belge bölme yapılandırması (tür bağlama hatalarını önlemek için dize daha sonra ayrıştırılır)
	ChunkSize     string `form:"chunk_size"`
	ChunkOverlap  string `form:"chunk_overlap"`
	SeparatorsRaw string `form:"separators"`
}

// TestMultimodalFunction godoc
// @Summary      Çok modlu işlevi test et
// @Description  Çok modlu işleme işlevini test etmek için resim yükler
// @Tags         Başlatma
// @Accept       multipart/form-data
// @Produce      json
// @Param        image             formData  file    true   "Test resmi"
// @Param        vlm_model         formData  string  true   "VLM model adı"
// @Param        vlm_base_url      formData  string  true   "VLM Base URL"
// @Param        vlm_api_key       formData  string  false  "VLM API Key"
// @Param        vlm_interface_type formData string  false  "VLM arayüz türü"
// @Param        storage_type      formData  string  true   "Depolama türü(cos/minio)"
// @Success      200               {object}  map[string]interface{}  "Test sonucu"
// @Failure      400               {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/multimodal/test [post]
func (h *InitializationHandler) TestMultimodalFunction(c *gin.Context) {
	ctx := c.Request.Context()

	logger.Info(ctx, "Testing multimodal functionality")

	var req testMultimodalForm
	if err := c.ShouldBind(&req); err != nil {
		logger.Error(ctx, "Failed to parse form data", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Form parametreleri okunamadı", "Failed to parse form parameters")))
		return
	}

	req.StorageType = strings.ToLower(req.StorageType)

	if req.VLMModel == "" || req.VLMBaseURL == "" {
		logger.Error(ctx, "VLM model name and base URL are required")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "VLM model adı ve Base URL boş olamaz", "VLM model name and Base URL are required")))
		return
	}

	// SSRF validation for VLM BaseURL
	if err := utils.ValidateURLForSSRF(req.VLMBaseURL); err != nil {
		logger.Warnf(ctx, "SSRF validation failed for VLM BaseURL: %v", err)
		c.Error(errors.NewBadRequestError(utils.FormatSSRFError("VLM Base URL", req.VLMBaseURL, err, types.LanguageFromContextOrDefault(ctx))))
		return
	}

	switch req.StorageType {
	case "cos":
		// Zorunlu: SecretID/SecretKey/Region/BucketName/AppID; PathPrefix isteğe bağlı
		if req.COSSecretID == "" || req.COSSecretKey == "" ||
			req.COSRegion == "" || req.COSBucketName == "" ||
			req.COSAppID == "" {
			logger.Error(ctx, "COS configuration is required")
			c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "COS yapılandırması boş olamaz", "COS configuration is required")))
			return
		}
	case "minio":
		if req.MinioBucketName == "" {
			logger.Error(ctx, "MinIO configuration is required")
			c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "MinIO yapılandırması boş olamaz", "MinIO configuration is required")))
			return
		}
	default:
		logger.Error(ctx, "Invalid storage type")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Geçersiz depolama türü", "Invalid storage type")))
		return
	}

	// Dosya boyutunu doğrular — MAX_FILE_SIZE_MB env (varsayılan 50MB).
	// utils/filesize.go yorumuna bakın: dağıtım zamanı env olarak kasıtlı şekilde korunur, runtime setting yapılmaz.
	maxSizeMB := utils.GetMaxFileSizeMB()
	maxSize := maxSizeMB * 1024 * 1024
	// Üst sınır multipart ayrıştırmasından önce geçerli olmalıdır: FormFile önce tüm body'yi arabelleğe alır
	// (belleği aşan kısım geçici dosyaya yazılır), ardından yapılan header.Size kontrolü yüklemenin zaten tamamlandığını görür.
	// nginx'in location /api/ ayarı hâlâ MAX_FILE_SIZE; burası app'e doğrudan bağlanıldığında aynı üst sınırdır.
	limitUploadBody(c, maxSize)

	// Yüklenen resim dosyasını al
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		if isRequestBodyTooLarge(err) {
			logger.Error(ctx, "File size too large")
			c.Error(errors.NewBadRequestError(fmt.Sprintf(types.LocalizedText(ctx, "Görsel dosyası %d MB sınırını aşamaz", "Image file must not exceed %d MB"), maxSizeMB)))
			return
		}
		logger.Error(ctx, "Failed to get uploaded image", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Yüklenen görsel alınamadı", "Failed to get uploaded image")))
		return
	}
	defer file.Close()

	// Dosya türünü doğrula
	if !strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {
		logger.Error(ctx, "Invalid file type, only images are allowed")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Yalnızca görsel dosyaları yüklenebilir", "Only image files can be uploaded")))
		return
	}

	if header.Size > maxSize {
		logger.Error(ctx, "File size too large")
		c.Error(errors.NewBadRequestError(fmt.Sprintf(types.LocalizedText(ctx, "Görsel dosyası %d MB sınırını aşamaz", "Image file must not exceed %d MB"), maxSizeMB)))
		return
	}
	logger.Infof(ctx, "Processing image: %s", utils.SanitizeForLog(header.Filename))

	// Belge bölme yapılandırmasını ayrıştır
	chunkSizeInt32, err := strconv.ParseInt(req.ChunkSize, 10, 32)
	if err != nil {
		logger.Error(ctx, "Failed to parse chunk size", err)
		c.Error(errors.NewBadRequestError("Failed to parse chunk size"))
		return
	}
	chunkSize := int32(chunkSizeInt32)
	if chunkSize < 100 || chunkSize > 10000 {
		chunkSize = 1000
	}

	chunkOverlapInt32, err := strconv.ParseInt(req.ChunkOverlap, 10, 32)
	if err != nil {
		logger.Error(ctx, "Failed to parse chunk overlap", err)
		c.Error(errors.NewBadRequestError("Failed to parse chunk overlap"))
		return
	}
	chunkOverlap := int32(chunkOverlapInt32)
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		chunkOverlap = 200
	}

	var separators []string
	if req.SeparatorsRaw != "" {
		if err := json.Unmarshal([]byte(req.SeparatorsRaw), &separators); err != nil {
			separators = []string{"\n\n", "\n", "。", "！", "？", ";", "；"}
		}
	} else {
		separators = []string{"\n\n", "\n", "。", "！", "？", ";", "；"}
	}

	// Görüntü dosyası içeriğini oku
	imageContent, err := io.ReadAll(file)
	if err != nil {
		logger.Error(ctx, "Failed to read image file", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Görsel dosyası okunamadı", "Failed to read image file")))
		return
	}

	// Çok modlu testi çağır
	startTime := time.Now()
	result, err := h.testMultimodalWithDocReader(
		ctx,
		imageContent, header.Filename,
		chunkSize, chunkOverlap, separators, &req,
	)
	processingTime := time.Since(startTime).Milliseconds()

	if err != nil {
		logger.Error(ctx, "Failed to test multimodal", err)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"success":         false,
				"message":         err.Error(),
				"processing_time": processingTime,
			},
		})
		return
	}

	logger.Infof(ctx, "Multimodal test completed successfully in %dms", processingTime)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"success":         true,
			"caption":         result["caption"],
			"ocr":             result["ocr"],
			"processing_time": processingTime,
		},
	})
}

// testMultimodalWithDocReader uses DocumentReader.Read for document reading,
// then returns basic information about the result.
func (h *InitializationHandler) testMultimodalWithDocReader(
	ctx context.Context,
	imageContent []byte, filename string,
	chunkSize, chunkOverlap int32, separators []string,
	req *testMultimodalForm,
) (map[string]string, error) {
	fileExt := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		fileExt = strings.ToLower(filename[idx+1:])
	}

	if h.documentReader == nil {
		return nil, fmt.Errorf("DocReader service not configured")
	}

	requestID, _ := types.RequestIDFromContext(ctx)

	readResult, err := h.documentReader.Read(ctx, &types.ReadRequest{
		FileContent: imageContent,
		FileName:    filename,
		FileType:    fileExt,
		RequestID:   requestID,
	})
	if err != nil {
		return nil, fmt.Errorf(types.LocalizedText(ctx, "DocReader hizmeti çağrılamadı: %v", "DocReader service call failed: %v"), err)
	}
	if readResult.Error != "" {
		return nil, fmt.Errorf(types.LocalizedText(ctx, "DocReader hizmeti hata döndürdü: %s", "DocReader service returned an error: %s"), readResult.Error)
	}

	result := map[string]string{
		"markdown": readResult.MarkdownContent,
		"caption":  "",
		"ocr":      "",
	}
	return result, nil
}

// TextRelationExtractionRequest metin ilişki çıkarma isteği yapısı
type TextRelationExtractionRequest struct {
	Text    string   `json:"text"     binding:"required"`
	Tags    []string `json:"tags"     binding:"required"`
	ModelID string   `json:"model_id" binding:"required"`
}

// TextRelationExtractionResponse metin ilişki çıkarma yanıtı yapısı
type TextRelationExtractionResponse struct {
	Nodes     []*types.GraphNode     `json:"nodes"`
	Relations []*types.GraphRelation `json:"relations"`
}

// ExtractTextRelations godoc
// @Summary      Metin ilişkilerini çıkar
// @Description  Metinden varlıkları ve ilişkileri çıkar
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      TextRelationExtractionRequest  true  "Çıkarma isteği"
// @Success      200      {object}  map[string]interface{}         "Çıkarma sonucu"
// @Failure      400      {object}  errors.AppError                "Hatalı istek parametresi"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/extract/text-relation [post]
func (h *InitializationHandler) ExtractTextRelations(c *gin.Context) {
	ctx := c.Request.Context()

	var req TextRelationExtractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Invalid text relation extraction request")
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Metinden ilişki çıkarma isteği geçersiz", "Invalid text relationship extraction request")))
		return
	}

	// Metin içeriğini doğrula
	if len(req.Text) == 0 {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Metin boş olamaz", "Text is required")))
		return
	}

	if len(req.Text) > 5000 {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Metin 5000 karakteri aşamaz", "Text must not exceed 5000 characters")))
		return
	}

	// Etiketleri doğrula
	if len(req.Tags) == 0 {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "En az bir ilişki etiketi seçin", "Select at least one relationship label")))
		return
	}

	// Model kimliğine göre chat modelini al
	chatModel, err := h.modelService.GetChatModel(ctx, req.ModelID)
	if err != nil {
		logger.Error(ctx, "Failed to get model", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Model alınamadı: ", "Failed to get model: ") + err.Error()))
		return
	}

	// Metin ilişkilerini çıkarmak için model hizmetini çağır
	result, err := h.extractRelationsFromText(ctx, req.Text, req.Tags, chatModel)
	if err != nil {
		logger.Error(ctx, "Text relation extraction failed", err)
		c.Error(errors.NewInternalServerError(types.LocalizedText(ctx, "Metinden ilişki çıkarılamadı: ", "Failed to extract text relationships: ") + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// extractRelationsFromText Metinden ilişkileri çıkar
func (h *InitializationHandler) extractRelationsFromText(
	ctx context.Context,
	text string,
	tags []string,
	chatModel chat.Chat,
) (*TextRelationExtractionResponse, error) {
	template := &types.PromptTemplateStructured{
		Description: h.config.ExtractManager.ExtractGraph.Description,
		Tags:        tags,
		Examples:    h.config.ExtractManager.ExtractGraph.Examples,
	}

	extractor := chatpipeline.NewExtractor(chatModel, template)
	graph, err := extractor.Extract(ctx, text)
	if err != nil {
		logger.Error(ctx, "Text relation extraction failed", err)
		return nil, err
	}
	extractor.RemoveUnknownRelation(ctx, graph)

	result := &TextRelationExtractionResponse{
		Nodes:     graph.Node,
		Relations: graph.Relation,
	}

	return result, nil
}

// FabriTextRequest is a request for generating example text
type FabriTextRequest struct {
	Tags    []string `json:"tags"`
	ModelID string   `json:"model_id" binding:"required"`
}

// FabriTextResponse is a response for generating example text
type FabriTextResponse struct {
	Text string `json:"text"`
}

// FabriText godoc
// @Summary      Örnek metin oluştur
// @Description  Etiketlere göre örnek metin oluştur
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Param        request  body      FabriTextRequest  true  "Oluşturma isteği"
// @Success      200      {object}  map[string]interface{}  "Oluşturulan metin"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametresi"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /initialization/extract/fabri-text [post]
func (h *InitializationHandler) FabriText(c *gin.Context) {
	ctx := c.Request.Context()

	var req FabriTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "failed to parse fabri text request")
		c.Error(errors.NewBadRequestError("invalid fabri text request parameters"))
		return
	}

	chatModel, err := h.modelService.GetChatModel(ctx, req.ModelID)
	if err != nil {
		logger.Error(ctx, "Failed to get model", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Model alınamadı: ", "Failed to get model: ") + err.Error()))
		return
	}

	result, err := h.fabriText(ctx, req.Tags, chatModel)
	if err != nil {
		logger.Error(ctx, "failed to generate fabri text", err)
		c.Error(errors.NewInternalServerError("failed to generate fabri text: " + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    FabriTextResponse{Text: result},
	})
}

// fabriText generates example text
func (h *InitializationHandler) fabriText(ctx context.Context, tags []string, chatModel chat.Chat) (string, error) {
	content := h.config.ExtractManager.FabriText.WithNoTag
	if len(tags) > 0 {
		tagStr, _ := json.Marshal(tags)
		content = fmt.Sprintf(h.config.ExtractManager.FabriText.WithTag, string(tagStr))
	}

	think := false
	result, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "user", Content: content},
	}, &chat.ChatOptions{
		Temperature: 0.3,
		MaxTokens:   4096,
		Thinking:    &think,
	})
	if err != nil {
		logger.Error(ctx, "Failed to generate example text", err)
		return "", err
	}
	return result.Content, nil
}

// FabriTagRequest is a request for generating tags
type FabriTagRequest struct{}

// FabriTagResponse is a response for generating tags
type FabriTagResponse struct {
	Tags []string `json:"tags"`
}

var tagOptions = []string{
	"Content", "Culture", "Person", "Event", "Time", "Location",
	"Work", "Author", "Relation", "Attribute",
}

// FabriTag godoc
// @Summary      Rastgele etiketler oluştur
// @Description  Rastgele bir etiket grubu oluştur
// @Tags         Başlatma
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Oluşturulan etiketler"
// @Router       /initialization/extract/fabri-tag [post]
func (h *InitializationHandler) FabriTag(c *gin.Context) {
	tagRandom := RandomSelect(tagOptions, rand.Intn(len(tagOptions)-1)+1)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    FabriTagResponse{Tags: tagRandom},
	})
}

// RandomSelect selects random strings
func RandomSelect(strs []string, n int) []string {
	if n <= 0 {
		return []string{}
	}
	result := make([]string, len(strs))
	copy(result, strs)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})

	if n > len(strs) {
		n = len(strs)
	}
	return result[:n]
}
