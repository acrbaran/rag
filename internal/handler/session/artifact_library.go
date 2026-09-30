package session

import (
	"net/http"
	"strings"

	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/gin-gonic/gin"
)

// ListArtifactLibrary godoc
// @Summary      Ürünlerimi listele
// @Description  Geçerli kullanıcının görebildiği oturumlardaki beceriler tarafından oluşturulan dosyaları oturumlar arasında listeler; her dosya için yalnızca en güncel sürüm döndürülür (depolama URL'si dahil değildir)
// @Tags         Oturumlar
// @Produce      json
// @Param        keyword     query  string  false  "Dosya adına göre filtrele"
// @Param        file_types  query  string  false  "Virgülle ayrılmış uzantılar, ör. .pdf,.pptx"
// @Param        page        query  int     false  "Sayfa numarası"
// @Param        page_size   query  int     false  "Sayfa başına öğe sayısı（en fazla 100）"
// @Success      200  {object}  map[string]interface{}
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /artifacts [get]
//
// Items carry session_id, message_id and index so the client downloads
// through the per-session endpoint, which re-runs the ownership check.
func (h *Handler) ListArtifactLibrary(c *gin.Context) {
	ctx := c.Request.Context()

	var pagination types.Pagination
	if err := c.ShouldBindQuery(&pagination); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	var fileTypes []string
	if raw := c.Query("file_types"); raw != "" {
		fileTypes = strings.Split(raw, ",")
	}

	result, err := h.messageService.ListArtifactLibrary(ctx, &types.ArtifactLibraryQuery{
		Keyword:   c.Query("keyword"),
		FileTypes: fileTypes,
		Page:      pagination.Page,
		PageSize:  pagination.PageSize,
	})
	if err != nil {
		logger.Errorf(ctx, "list artifact library failed: %v", err)
		_ = c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      result.Data,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}
