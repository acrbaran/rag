package handler

import (
	"context"
	stderrors "errors"
	"net/http"

	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
)

// GetMCPMetadata godoc
// @Summary      Kaydedilmiş MCP araç kataloğunu okur
// @Description  Yalnızca veritabanını okur, üst sisteme bağlanmaz. Senkronize edilmemişse data null olur; bağlantı yapılandırması değiştiğinde stale true olur. OAuth kataloğu, geçerli yetkilendirme öznesine göre ayrılır.
// @Tags         MCP Hizmetleri
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "MCP hizmet ID'si"
// @Success      200  {object}  map[string]interface{}  "Katalog anlık görüntüsü"
// @Failure      400  {object}  errors.AppError         "İstek parametresi hatası"
// @Failure      401  {object}  errors.AppError         "OAuth kataloğunda yetkilendirme öznesi eksik"
// @Failure      404  {object}  errors.AppError         "Hizmet bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /mcp-services/{id}/metadata [get]
func (h *MCPServiceHandler) GetMCPMetadata(c *gin.Context) { h.mcpMetadata(c, false) }

// RefreshMCPMetadata godoc
// @Summary      MCP araç kataloğunu senkronize eder
// @Description  Üst sisteme açıkça bağlanır ve tam kataloğu atomik olarak değiştirir. OAuth hizmeti geçerli kullanıcının anlık görüntüsünü yazar; Viewer ve üzeri çağırabilir. Statik kimlik doğrulama, kiracı tarafından paylaşılan anlık görüntüyü yazar ve Admin gerektirir.
// @Tags         MCP Hizmetleri
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "MCP hizmet ID'si"
// @Success      200  {object}  map[string]interface{}  "Senkronizasyon sonrası katalog anlık görüntüsü"
// @Failure      400  {object}  errors.AppError         "Katalog eksik veya doğrulama başarısız"
// @Failure      401  {object}  errors.AppError         "OAuth kataloğunda yetkilendirme öznesi eksik"
// @Failure      403  {object}  errors.AppError         "Statik kimlik doğrulama kataloğu için yönetici yenilemesi gerekir"
// @Failure      404  {object}  errors.AppError         "Hizmet bulunamadı"
// @Failure      409  {object}  errors.AppError         "Yenileme sırasında bağlantı yapılandırması değişti"
// @Failure      503  {object}  errors.AppError         "Meta veri deposu kullanılamıyor"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /mcp-services/{id}/metadata/refresh [post]
func (h *MCPServiceHandler) RefreshMCPMetadata(c *gin.Context) { h.mcpMetadata(c, true) }

func (h *MCPServiceHandler) mcpMetadata(c *gin.Context, refresh bool) {
	ctx := c.Request.Context()
	tenant := c.GetUint64(types.TenantIDContextKey.String())
	if tenant == 0 {
		_ = c.Error(errors.NewBadRequestError("Workspace ID cannot be empty"))
		return
	}
	svc, ok := h.mcpServiceService.(interfaces.MCPMetadataService)
	if !ok {
		_ = c.Error(errors.NewServiceUnavailableError("MCP metadata storage is unavailable"))
		return
	}
	id := c.Param("id")
	var snapshot *types.MCPMetadata
	var err error
	if refresh {
		service, getErr := h.mcpServiceService.GetMCPServiceByID(ctx, tenant, id)
		if getErr != nil || service == nil {
			logger.ErrorWithFields(ctx, getErr, map[string]interface{}{
				"service_id": secutils.SanitizeForLog(id),
				"refresh":    true,
			})
			_ = c.Error(mcpMetadataAppError(types.ErrMCPServiceNotFound, true))
			return
		}
		if !service.AuthConfig.IsOAuth() && !mayWriteSharedMCPMetadata(ctx) {
			_ = c.Error(errors.NewForbiddenError("Refreshing a shared MCP directory requires an administrator"))
			return
		}
		snapshot, err = svc.RefreshMCPMetadata(ctx, tenant, id)
	} else {
		snapshot, err = svc.GetMCPMetadata(ctx, tenant, id)
	}
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"service_id": secutils.SanitizeForLog(id),
			"refresh":    refresh,
		})
		_ = c.Error(mcpMetadataAppError(err, refresh))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshot})
}

// mayWriteSharedMCPMetadata is the extra gate for static-auth catalogs. The
// route stays Viewer+ so OAuth users can persist their own snapshot after
// authorizing in chat. API keys already passed manage-MCP; JWT callers need Admin.
func mayWriteSharedMCPMetadata(ctx context.Context) bool {
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		return true
	}
	if types.IsSystemAdminFromContext(ctx) {
		return true
	}
	return types.CallerFromContext(ctx).Role.HasPermission(types.TenantRoleAdmin)
}

func mcpMetadataAppError(err error, refresh bool) *errors.AppError {
	switch {
	case stderrors.Is(err, types.ErrMCPServiceNotFound):
		return errors.NewNotFoundError("MCP service not found")
	case stderrors.Is(err, types.ErrMCPOAuthPrincipalRequired):
		return errors.NewUnauthorizedError("OAuth metadata requires an authenticated user")
	case stderrors.Is(err, types.ErrMCPMetadataStorage):
		return errors.NewServiceUnavailableError("MCP metadata storage is unavailable")
	case stderrors.Is(err, types.ErrMCPMetadataConnectionChanged):
		return errors.NewConflictError("MCP connection changed during refresh; save the configuration and sync again")
	case stderrors.Is(err, types.ErrMCPMetadataTooLarge), stderrors.Is(err, types.ErrMCPMetadataInvalidTools):
		return errors.NewBadRequestError("MCP directory is invalid or too large")
	default:
		if refresh {
			return errors.NewBadRequestError("Failed to refresh MCP tools. Check the connection and try again.")
		}
		return errors.NewInternalServerError("Failed to read MCP metadata")
	}
}
