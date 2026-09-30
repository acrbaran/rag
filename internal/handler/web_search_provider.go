package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/handler/dto"
	infra_web_search "github.com/acrbaran/rag/internal/infrastructure/web_search"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
)

// WebSearchProviderHandler handles HTTP requests for web search provider CRUD
type WebSearchProviderHandler struct {
	repo     interfaces.WebSearchProviderRepository
	service  interfaces.WebSearchProviderService
	registry *infra_web_search.Registry
}

// NewWebSearchProviderHandler creates a new handler
func NewWebSearchProviderHandler(
	repo interfaces.WebSearchProviderRepository,
	service interfaces.WebSearchProviderService,
	registry *infra_web_search.Registry,
) *WebSearchProviderHandler {
	return &WebSearchProviderHandler{repo: repo, service: service, registry: registry}
}

// --- request DTOs ---

// CreateProviderRequest defines the request body for creating a provider
type CreateProviderRequest struct {
	Name        string                            `json:"name" binding:"required"`
	Provider    types.WebSearchProviderType       `json:"provider" binding:"required"`
	Description string                            `json:"description"`
	Parameters  types.WebSearchProviderParameters `json:"parameters"`
	IsDefault   bool                              `json:"is_default"`
}

// UpdateProviderRequest defines the request body for updating a provider
type UpdateProviderRequest struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Parameters  types.WebSearchProviderParameters `json:"parameters"`
	IsDefault   bool                              `json:"is_default"`
}

// --- helpers ---

// getTenantID extracts tenant ID from gin context (set by auth middleware).
func (h *WebSearchProviderHandler) getTenantID(c *gin.Context) uint64 {
	return c.GetUint64(types.TenantIDContextKey.String())
}

// getOwnedProvider loads a provider and verifies it belongs to the given tenant.
// Returns (nil, status, msg) on failure so callers can respond immediately.
func (h *WebSearchProviderHandler) getOwnedProvider(
	ctx context.Context, tenantID uint64, id string,
) (*types.WebSearchProviderEntity, int, string) {
	provider, err := h.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, http.StatusInternalServerError, "failed to query provider"
	}
	if provider == nil {
		return nil, http.StatusNotFound, "web search provider not found"
	}
	return provider, http.StatusOK, ""
}

// --- endpoints ---

// CreateProvider creates a new web search provider
func (h *WebSearchProviderHandler) CreateProvider(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warnf(ctx, "Invalid create provider request: %v", err)
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	logger.Infof(ctx, "Creating web search provider: tenant=%d, name=%s, type=%s",
		tenantID, secutils.SanitizeForLog(req.Name), secutils.SanitizeForLog(string(req.Provider)))

	provider := &types.WebSearchProviderEntity{
		TenantID:    tenantID,
		Name:        secutils.SanitizeForLog(req.Name),
		Provider:    req.Provider,
		Description: secutils.SanitizeForLog(req.Description),
		Parameters:  req.Parameters,
		IsDefault:   req.IsDefault,
	}

	if err := h.service.CreateProvider(ctx, provider); err != nil {
		logger.Warnf(ctx, "Failed to create web search provider: %v", err)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    dto.NewWebSearchProviderResponse(ctx, provider),
	})
}

// ListProviders lists all web search providers for the current tenant
func (h *WebSearchProviderHandler) ListProviders(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	providers, err := h.repo.List(ctx, tenantID)
	if err != nil {
		logger.Warnf(ctx, "Failed to list web search providers: %v", err)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto.NewWebSearchProviderResponses(ctx, providers),
	})
}

// GetProvider retrieves a single web search provider by ID.
//
// GetProvider godoc
// @Summary      Web arama Provider ayrıntılarını al
// @Description  ID'ye göre belirtilen provider yapılandırmasını al
// @Tags         Web Araması
// @Produce      json
// @Param        id   path      string                          true  "Provider ID"
// @Success      200  {object}  types.WebSearchProviderEntity   "Provider ayrıntıları"
// @Failure      404  {object}  map[string]interface{}          "Provider bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/{id} [get]
func (h *WebSearchProviderHandler) GetProvider(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	id := c.Param("id")
	provider, status, msg := h.getOwnedProvider(ctx, tenantID, id)
	if status != http.StatusOK {
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dto.NewWebSearchProviderResponse(ctx, provider),
	})
}

// UpdateProvider updates a web search provider.
//
// UpdateProvider godoc
// @Summary      Web arama Provider'ını güncelle
// @Description  Belirtilen provider'ın adını/açıklamasını/parametrelerini/varsayılan olup olmadığını güncelle
// @Tags         Web Araması
// @Accept       json
// @Produce      json
// @Param        id       path      string                          true  "Provider ID"
// @Param        request  body      handler.UpdateProviderRequest   true  "Güncellenecek alanlar"
// @Success      200      {object}  types.WebSearchProviderEntity   "Güncellenmiş Provider"
// @Failure      400      {object}  map[string]interface{}          "Hatalı istek parametreleri"
// @Failure      404      {object}  map[string]interface{}          "Provider bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/{id} [put]
func (h *WebSearchProviderHandler) UpdateProvider(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	id := c.Param("id")

	// Ownership check
	existing, status, msg := h.getOwnedProvider(ctx, tenantID, id)
	if status != http.StatusOK {
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	// Credentials (api_key) NEVER flow through this endpoint — they live
	// behind the /credentials subresource. Force-preserve the stored key
	// regardless of what the body says; log a warning if a stale caller
	// passes one so we can spot them.
	if req.Parameters.APIKey != "" && req.Parameters.APIKey != existing.Parameters.APIKey {
		logger.Warnf(ctx,
			"deprecated: api_key in PUT /web-search-providers/%s body is ignored; use PUT /credentials instead",
			secutils.SanitizeForLog(id))
	}
	mergedParams := req.Parameters
	mergedParams.APIKey = existing.Parameters.APIKey
	// Preserve ExtraConfig when the request omits it (nil); otherwise a
	// partial PUT would silently drop tenant-configured extras.
	if mergedParams.ExtraConfig == nil {
		mergedParams.ExtraConfig = existing.Parameters.ExtraConfig
	}

	// Preserve existing values for top-level metadata fields when the
	// request omits them (empty string from the JSON decoder). Without this,
	// a partial update that only flips IsDefault would clobber Name and
	// Description on the stored record.
	mergedName := req.Name
	if mergedName == "" {
		mergedName = existing.Name
	}
	mergedDescription := req.Description
	if mergedDescription == "" {
		mergedDescription = existing.Description
	}

	// Build updated entity, keeping immutable fields from existing
	provider := &types.WebSearchProviderEntity{
		ID:          id,
		TenantID:    tenantID,
		Name:        secutils.SanitizeForLog(mergedName),
		Provider:    existing.Provider, // Provider type is immutable after creation
		Description: secutils.SanitizeForLog(mergedDescription),
		Parameters:  mergedParams,
		IsDefault:   req.IsDefault,
	}

	if err := h.service.UpdateProvider(ctx, provider); err != nil {
		logger.Warnf(ctx, "Failed to update web search provider %s: %v", id, err)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	// Re-fetch to get the full stored state
	updated, _ := h.repo.GetByID(ctx, tenantID, id)
	if updated != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": dto.NewWebSearchProviderResponse(ctx, updated)})
	} else {
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// DeleteProvider deletes a web search provider.
//
// DeleteProvider godoc
// @Summary      Ağ arama Provider'ını sil
// @Description  Belirtilen provider yapılandırmasını sil
// @Tags         Web Araması
// @Produce      json
// @Param        id   path      string                  true  "Provider ID"
// @Success      200  {object}  map[string]interface{}  "success: true"
// @Failure      404  {object}  map[string]interface{}  "Provider mevcut değil"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/{id} [delete]
func (h *WebSearchProviderHandler) DeleteProvider(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	id := c.Param("id")

	// Ownership check
	if _, status, msg := h.getOwnedProvider(ctx, tenantID, id); status != http.StatusOK {
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	if err := h.service.DeleteProvider(ctx, tenantID, id); err != nil {
		logger.Warnf(ctx, "Failed to delete web search provider %s: %v", id, err)
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListProviderTypes returns available provider types and their parameter requirements.
//
// ListProviderTypes godoc
// @Summary      Ağ arama Provider türü meta verilerini al
// @Description  UI formunun ihtiyaç duyduğu provider türlerini ve parametre tanımlarını döndür
// @Tags         Web Araması
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "provider türü listesi"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/types [get]
func (h *WebSearchProviderHandler) ListProviderTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    types.GetWebSearchProviderTypes(),
	})
}

// TestProviderByID tests an existing saved provider by performing a sample search.
//
// TestProviderByID godoc
// @Summary      Kaydedilmiş Provider'ı test et
// @Description  Veritabanında kaydedilmiş kimlik bilgilerini kullanarak bağlantıyı test et
// @Tags         Web Araması
// @Produce      json
// @Param        id   path      string                  true  "Provider ID"
// @Success      200  {object}  map[string]interface{}  "Test sonucu"
// @Failure      404  {object}  map[string]interface{}  "Provider mevcut değil"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/{id}/test [post]
func (h *WebSearchProviderHandler) TestProviderByID(c *gin.Context) {
	ctx := c.Request.Context()

	tenantID := h.getTenantID(c)
	if tenantID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "unauthorized: workspace context missing"})
		return
	}

	id := c.Param("id")
	provider, status, msg := h.getOwnedProvider(ctx, tenantID, id)
	if status != http.StatusOK {
		c.JSON(status, gin.H{"success": false, "error": msg})
		return
	}

	if err := h.doTestSearch(ctx, string(provider.Provider), provider.Parameters); err != nil {
		logger.Warnf(ctx, "Web search provider test failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// TestProviderRequest defines the body for testing raw credentials
type TestProviderRequest struct {
	Provider   string                            `json:"provider" binding:"required"`
	Parameters types.WebSearchProviderParameters `json:"parameters"`
}

// TestProviderRaw tests a provider with raw credentials (no persistence).
//
// TestProviderRaw godoc
// @Summary      Provider'ı ham kimlik bilgileriyle test et (veritabanına kaydetmeden)
// @Description  "Bağlantıyı test et" düğmesi için ön uç formunda henüz kaydedilmemiş kimlik bilgilerini kullanarak bağlantıyı test et
// @Tags         Web Araması
// @Accept       json
// @Produce      json
// @Param        request  body      handler.TestProviderRequest  true  "{provider, parameters}"
// @Success      200      {object}  map[string]interface{}  "Test sonucu"
// @Failure      400      {object}  map[string]interface{}  "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search-providers/test [post]
func (h *WebSearchProviderHandler) TestProviderRaw(c *gin.Context) {
	ctx := c.Request.Context()

	var req TestProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.NewBadRequestError(err.Error()))
		return
	}

	if err := h.doTestSearch(ctx, req.Provider, req.Parameters); err != nil {
		logger.Warnf(ctx, "Web search provider test failed: %v", err)
		c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// doTestSearch creates a temporary provider and runs a simple test query.
//
// The provider would otherwise try to authenticate against the upstream API
// with the redacted placeholder (which is guaranteed to fail with a
// confusing error). Reject it up front with an actionable message so the
// user knows they should type a real key or test against the saved config
// via /test instead.
func (h *WebSearchProviderHandler) doTestSearch(ctx context.Context, providerType string, params types.WebSearchProviderParameters) error {
	logger.Infof(ctx, "[WebSearch][Test] testing provider type=%s", providerType)
	searchProvider, err := h.registry.CreateProvider(providerType, params)
	if err != nil {
		logger.Warnf(ctx, "[WebSearch][Test] failed to create provider: %v", err)
		return fmt.Errorf("failed to create provider: %w", err)
	}
	results, err := searchProvider.Search(ctx, "test", 1, false)
	if err != nil {
		logger.Warnf(ctx, "[WebSearch][Test] search failed: %v", err)
		return err
	}
	if len(results) == 0 {
		err := infra_web_search.EmptyTestResultsError(providerType, searchProvider)
		logger.Warnf(ctx, "[WebSearch][Test] %v", err)
		return err
	}
	logger.Infof(ctx, "[WebSearch][Test] succeeded: type=%s, results=%d", providerType, len(results))
	return nil
}
