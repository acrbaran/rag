package handler

import (
	"net/http"

	"github.com/acrbaran/rag/internal/types"
	"github.com/gin-gonic/gin"
)

// WebSearchHandler handles legacy web search related requests
type WebSearchHandler struct{}

// NewWebSearchHandler creates a new web search handler
func NewWebSearchHandler() *WebSearchHandler {
	return &WebSearchHandler{}
}

// GetProviders returns the list of available web search provider types.
//
// GetProviders godoc
// @Summary      Kullanılabilir ağ arama Provider listesini alır
// @Description  Kaydedilmiş tüm ağ arama provider'larını (meta veriler dahil) döndürür
// @Tags         Web Araması
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "provider listesi"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /web-search/providers [get]
func (h *WebSearchHandler) GetProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    types.GetWebSearchProviderTypes(),
	})
}
