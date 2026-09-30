package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/application/service/memory"
	apperrors "github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
)

// MemoryHandler exposes the caller's own long-term memory.
//
// Every route operates on the memory space derived from the request context,
// so no endpoint takes a subject id. That is deliberate: it removes the entire
// class of "can I read another user's memories by changing an id" bugs instead
// of relying on a per-route ownership check.
type MemoryHandler struct {
	memoryService interfaces.MemoryService
}

func NewMemoryHandler(memoryService interfaces.MemoryService) *MemoryHandler {
	return &MemoryHandler{memoryService: memoryService}
}

// GetSettings godoc
// @Summary      Bellek ayarlarımı al
// @Description  Birleştirilmiş bellek anahtarı durumunu (alan düzeyi + kişisel düzey) ve bellek öğesi sayısını döndürür
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Bellek ayarları"
// @Security     Bearer
// @Router       /memory/settings [get]
func (h *MemoryHandler) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settings, err := h.memoryService.GetSettings(ctx)
	if err != nil {
		h.fail(c, err, "Failed to load memory settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

type updateMemorySettingsRequest struct {
	Enabled *bool `json:"enabled"`
}

// UpdateSettings godoc
// @Summary      Bellek ayarlarımı güncelle
// @Description  Mevcut kullanıcının kendi uzun süreli belleğini açar veya kapatır
// @Tags         Uzun Süreli Bellek
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "Ayarlar"
// @Success      200      {object}  map[string]interface{}  "Güncellenmiş ayarlar"
// @Security     Bearer
// @Router       /memory/settings [put]
func (h *MemoryHandler) UpdateSettings(c *gin.Context) {
	ctx := c.Request.Context()
	var req updateMemorySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request data").WithDetails(err.Error()))
		return
	}
	if req.Enabled == nil {
		c.Error(apperrors.NewBadRequestError("enabled is required"))
		return
	}
	if err := h.memoryService.SetEnabled(ctx, *req.Enabled); err != nil {
		h.fail(c, err, "Failed to update memory settings")
		return
	}
	settings, err := h.memoryService.GetSettings(ctx)
	if err != nil {
		h.fail(c, err, "Failed to load memory settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

// ListItems godoc
// @Summary      Belleklerimi listele
// @Description  Mevcut kullanıcının bellek öğelerini sayfalı olarak döndürür, duruma göre filtrelenebilir
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        status  query     string  false  "Durum filtresi"  Enums(active, superseded, archived, pending)
// @Param        limit   query     int     false  "Sayfa başına öğe sayısı"  default(50)
// @Param        offset  query     int     false  "Ofset"
// @Success      200     {object}  map[string]interface{}  "Bellek listesi"
// @Security     Bearer
// @Router       /memory/items [get]
func (h *MemoryHandler) ListItems(c *gin.Context) {
	ctx := c.Request.Context()
	status := c.Query("status")
	switch status {
	case "", types.MemoryStatusActive, types.MemoryStatusSuperseded,
		types.MemoryStatusArchived, types.MemoryStatusPending:
	default:
		c.Error(apperrors.NewBadRequestError("unsupported status"))
		return
	}
	limit, offset := memoryListPaging(c)

	items, total, err := h.memoryService.ListItems(ctx, status, limit, offset)
	if err != nil {
		h.fail(c, err, "Failed to list memories")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    items,
		"total":   total,
	})
}

const (
	// memoryExportPageSize is how many rows one export page reads.
	memoryExportPageSize = 500
	// memoryExportMaxItems bounds a single export so one enormous store cannot
	// turn a download into an unbounded read.
	memoryExportMaxItems = 20000
)

func memoryListPaging(c *gin.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// ListTopics godoc
// @Summary      İzlenen konuları listele
// @Description  Sayımı yapılmış, henüz uzun süreli ilgi alanına yükseltilmemiş konuları ve eşiğe ulaşmak için kaç kez daha gerektiğini döndürür
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        limit   query     int  false  "Sayfa başına öğe sayısı"  default(50)
// @Param        offset  query     int  false  "Ofset"
// @Success      200     {object}  map[string]interface{}  "Konu listesi"
// @Security     Bearer
// @Router       /memory/topics [get]
func (h *MemoryHandler) ListTopics(c *gin.Context) {
	ctx := c.Request.Context()
	limit, offset := memoryListPaging(c)
	topics, total, err := h.memoryService.ListTopics(ctx, limit, offset)
	if err != nil {
		h.fail(c, err, "Failed to list topics")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    topics,
		"total":   total,
	})
}

// PromoteTopic godoc
// @Summary      Hemen uzun süreli ilgi alanı olarak kaydet
// @Description  Kalan sayıları beklemeden, izlenen konuyu uzun süreli ilgi belleği olarak yükseltir
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id   path      string  true  "Konu ID"
// @Success      200  {object}  map[string]interface{}  "Yeni eklenen bellek"
// @Security     Bearer
// @Router       /memory/topics/{id}/promote [post]
func (h *MemoryHandler) PromoteTopic(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := h.memoryService.PromoteTopic(ctx, c.Param("id"))
	if err != nil {
		h.fail(c, err, "Failed to promote topic")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

// DeleteTopic godoc
// @Summary      Bir konuyu izlemeyi bırak
// @Description  Henüz yükseltilmemiş konu sayacını siler ve bu reddi hatırlar; daha sonra otomatik olarak uzun vadeli ilgi olarak işaretlenmez
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id   path      string  true  "Konu ID'si"
// @Success      200  {object}  map[string]interface{}  "Başarıyla silindi"
// @Security     Bearer
// @Router       /memory/topics/{id} [delete]
func (h *MemoryHandler) DeleteTopic(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.memoryService.DeleteTopic(ctx, c.Param("id")); err != nil {
		h.fail(c, err, "Failed to delete topic")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListDocuments godoc
// @Summary      Sık kullanılan belgeleri listeler
// @Description  Mevcut kullanıcının yanıtlarında tekrar tekrar başvurulan belgeleri döndürür; alışkanlık eşiğine ulaşmayanlar gösterilmez
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        limit   query     int  false  "Sayfa başına öğe sayısı"  default(50)
// @Param        offset  query     int  false  "Ofset"
// @Success      200     {object}  map[string]interface{}  "Belge listesi"
// @Security     Bearer
// @Router       /memory/documents [get]
func (h *MemoryHandler) ListDocuments(c *gin.Context) {
	ctx := c.Request.Context()
	limit, offset := memoryListPaging(c)
	docs, total, err := h.memoryService.ListDocuments(ctx, limit, offset)
	if err != nil {
		h.fail(c, err, "Failed to list documents")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    docs,
		"total":   total,
	})
}

// DeleteDocument godoc
// @Summary      Belirli bir belgeyi kişiselleştirilmiş aramada kullanmayı durdurur
// @Description  Bir belge yakınlık sayacını siler; bundan sonra arama bu belge nedeniyle ağırlıklandırılmaz
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id   path      string  true  "Yakınlık ID'si"
// @Success      200  {object}  map[string]interface{}  "Başarıyla silindi"
// @Security     Bearer
// @Router       /memory/documents/{id} [delete]
func (h *MemoryHandler) DeleteDocument(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.memoryService.DeleteDocument(ctx, c.Param("id")); err != nil {
		h.fail(c, err, "Failed to delete document affinity")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type createMemoryItemRequest struct {
	Kind       string `json:"kind"`
	Content    string `json:"content"`
	Importance int    `json:"importance"`
}

// CreateItem godoc
// @Summary      Yeni bir bellek ekler
// @Description  Manuel olarak uzun vadeli bir bellek ekler
// @Tags         Uzun Süreli Bellek
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "Bellek içeriği"
// @Success      200      {object}  map[string]interface{}  "Eklenen bellek"
// @Security     Bearer
// @Router       /memory/items [post]
func (h *MemoryHandler) CreateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req createMemoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request data").WithDetails(err.Error()))
		return
	}
	item, err := h.memoryService.CreateItem(ctx, req.Kind, req.Content, req.Importance)
	if err != nil {
		h.fail(c, err, "Failed to create memory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

type updateMemoryItemRequest struct {
	Content    string `json:"content"`
	Importance int    `json:"importance"`
}

// UpdateItem godoc
// @Summary      Bir belleği değiştirir
// @Description  Bellek içeriğini ve önem derecesini değiştirir; değişiklikten sonra bu bellek arka plan çıkarımı tarafından üzerine yazılmaz
// @Tags         Uzun Süreli Bellek
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Bellek ID'si"
// @Param        request  body      object  true  "Bellek içeriği"
// @Success      200      {object}  map[string]interface{}  "Güncellenmiş bellek"
// @Security     Bearer
// @Router       /memory/items/{id} [put]
func (h *MemoryHandler) UpdateItem(c *gin.Context) {
	ctx := c.Request.Context()
	var req updateMemoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request data").WithDetails(err.Error()))
		return
	}
	item, err := h.memoryService.UpdateItem(ctx, c.Param("id"), req.Content, req.Importance)
	if err != nil {
		h.fail(c, err, "Failed to update memory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

// DeleteItem godoc
// @Summary      Bir belleği siler
// @Description  Bir belleği kalıcı olarak siler
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id  path      string  true  "Bellek ID'si"
// @Success      200  {object}  map[string]interface{}  "Başarıyla silindi"
// @Security     Bearer
// @Router       /memory/items/{id} [delete]
func (h *MemoryHandler) DeleteItem(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.memoryService.DeleteItem(ctx, c.Param("id")); err != nil {
		h.fail(c, err, "Failed to delete memory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ConfirmItem godoc
// @Summary      Çıkarımı yapılmış bir anıyı onayla
// @Description  Sistem tarafından çıkarımı yapılan anıyı kabul ederek etkinleşmesini sağlar
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id   path      string  true  "Anı ID"
// @Success      200  {object}  map[string]interface{}  "Onay başarılı"
// @Security     Bearer
// @Router       /memory/items/{id}/confirm [post]
//
// Inferred memories are the ones worth having and the ones most likely to be
// wrong, so they wait here rather than taking effect silently.
func (h *MemoryHandler) ConfirmItem(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := h.memoryService.ConfirmItem(ctx, c.Param("id"))
	if err != nil {
		h.fail(c, err, "Failed to confirm memory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}

// RejectItem godoc
// @Summary      Çıkarımı yapılmış bir anıyı reddet
// @Description  Sistem tarafından çıkarımı yapılan anıyı reddeder ve bu reddi hatırlar
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Param        id   path      string  true  "Anı ID"
// @Success      200  {object}  map[string]interface{}  "Reddetme başarılı"
// @Security     Bearer
// @Router       /memory/items/{id}/reject [post]
func (h *MemoryHandler) RejectItem(c *gin.Context) {
	ctx := c.Request.Context()
	if err := h.memoryService.RejectItem(ctx, c.Param("id")); err != nil {
		h.fail(c, err, "Failed to reject memory")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Clear godoc
// @Summary      Anılarımı temizle
// @Description  Geçerli kullanıcının tüm anılarını kalıcı olarak siler
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Temizleme başarılı"
// @Security     Bearer
// @Router       /memory/items [delete]
func (h *MemoryHandler) Clear(c *gin.Context) {
	ctx := c.Request.Context()
	removed, err := h.memoryService.Clear(ctx)
	if err != nil {
		h.fail(c, err, "Failed to clear memories")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "removed": removed})
}

// Export godoc
// @Summary      Anılarımı dışa aktar
// @Description  Geçerli kullanıcının tüm anılarını JSON olarak dışa aktarır
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Anı dışa aktarımı"
// @Security     Bearer
// @Router       /memory/export [get]
func (h *MemoryHandler) Export(c *gin.Context) {
	ctx := c.Request.Context()
	// Export is a snapshot, not a page, so it walks every status to the end.
	//
	// A single fixed page used to serve this on the grounds that it matched the
	// largest capacity a workspace can configure. It does not: max_items caps
	// active memories only, while superseded and archived rows accumulate
	// without limit, so a long-lived store holds far more than its capacity and
	// the export quietly returned a prefix of it.
	var items []*types.MemoryItem
	var total int64
	for {
		page, pageTotal, err := h.memoryService.ListItems(ctx, "", memoryExportPageSize, len(items))
		if err != nil {
			h.fail(c, err, "Failed to export memories")
			return
		}
		total = pageTotal
		items = append(items, page...)
		if len(page) < memoryExportPageSize || int64(len(items)) >= total {
			break
		}
		if len(items) >= memoryExportMaxItems {
			break
		}
	}
	c.Header("Content-Disposition", `attachment; filename="rethra-memories.json"`)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   total,
		// Say so rather than letting a partial file look complete. Only the
		// safety ceiling can trigger this, so it stays false in practice.
		"truncated": int64(len(items)) < total,
		"data":      items,
	})
}

// Consolidate godoc
// @Summary      Anılarımı hemen düzenle
// @Description  Anlamları yakın kayıtları birleştirir, süresi dolan öğeleri arşivler; günlük arka plan düzenlemesini beklemez
// @Tags         Uzun Süreli Bellek
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "Düzenleme sonucu"
// @Security     Bearer
// @Router       /memory/consolidate [post]
func (h *MemoryHandler) Consolidate(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := h.memoryService.ConsolidateNow(ctx)
	if err != nil {
		h.fail(c, err, "Failed to consolidate memories")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// fail maps service errors onto HTTP responses. A missing item and an item
// belonging to someone else produce the same 404 on purpose.
func (h *MemoryHandler) fail(c *gin.Context, err error, message string) {
	switch {
	case errors.Is(err, memory.ErrNoMemoryScope):
		c.Error(apperrors.NewUnauthorizedError("no principal in request"))
	case errors.Is(err, memory.ErrItemNotFound):
		c.Error(apperrors.NewNotFoundError("memory not found"))
	case errors.Is(err, types.ErrMemoryConflict):
		c.Error(apperrors.NewConflictError(err.Error()))
	case errors.Is(err, memory.ErrSensitiveContent):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	case errors.Is(err, memory.ErrMemoryDisabled):
		c.Error(apperrors.NewBadRequestError("memory is disabled"))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(message).WithDetails(err.Error()))
	}
}
