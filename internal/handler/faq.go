package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
)

// FAQHandler handles FAQ knowledge base operations.
//
// All KB-access checks (own / org-shared / via shared agent) are now
// performed by the route-level g.KBAccessRead / g.KBAccessWrite
// guards in router.go — the guard rewrites c.Request.Context() to
// carry the effective tenant ID for the duration of the handler, so
// the handler reads tenant from context the way it always did.
type FAQHandler struct {
	knowledgeService interfaces.KnowledgeService
	kbService        interfaces.KnowledgeBaseService
}

// NewFAQHandler creates a new FAQ handler.
func NewFAQHandler(
	knowledgeService interfaces.KnowledgeService,
	kbService interfaces.KnowledgeBaseService,
) *FAQHandler {
	return &FAQHandler{
		knowledgeService: knowledgeService,
		kbService:        kbService,
	}
}

// faqDeleteRequest is a request for deleting FAQ entries in batch
type faqDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

// faqEntryTagBatchRequest is a request for updating tags for FAQ entries in batch
// key: entry seq_id, value: tag seq_id (nil to remove tag)
type faqEntryTagBatchRequest struct {
	Updates map[int64]*int64 `json:"updates" binding:"required,min=1"`
}

// addSimilarQuestionsRequest is a request for adding similar questions to a FAQ entry
type addSimilarQuestionsRequest struct {
	SimilarQuestions []string `json:"similar_questions" binding:"required,min=1"`
}

// updateLastFAQImportResultDisplayStatusRequest is the request payload for UpdateLastImportResultDisplayStatus
type updateLastFAQImportResultDisplayStatusRequest struct {
	DisplayStatus string `json:"display_status" binding:"required,oneof=open close"`
}

// ListEntries godoc
// @Summary      FAQ giriş listesini getir
// @Description  Bilgi bankası altındaki FAQ girişlerinin listesini getirir, sayfalama ve filtrelemeyi destekler
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id           path      string  true   "Bilgi bankası ID'si"
// @Param        page         query     int     false  "Sayfa numarası"
// @Param        page_size    query     int     false  "Sayfa başına sayı"
// @Param        tag_id       query     int     false  "Etiket ID filtresi(seq_id), eski tek etiket sürümüyle uyumlu"
// @Param        tag_ids      query     string  false  "Etiket UUID filtresi, virgülle ayrılmıştır (OR anlamı)"
// @Param        keyword      query     string  false  "Anahtar kelime araması"
// @Param        search_field query     string  false  "Arama alanı: standard_question(standart soru), similar_questions(benzer soru biçimleri), answers(yanıtlar), varsayılan olarak tümünde arar"
// @Param        sort_order   query     string  false  "Sıralama yöntemi: asc(güncelleme zamanına göre artan), varsayılan olarak güncelleme zamanına göre azalan"
// @Param        is_enabled   query     bool    false  "Etkin durum filtresi; gönderilmezse tümü döndürülür"
// @Success      200        {object}  map[string]interface{}  "FAQ listesi"
// @Failure      400        {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries [get]
func (h *FAQHandler) ListEntries(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var page types.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		logger.Error(ctx, "Failed to bind pagination query", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Sayfalama parametreleri geçersiz", "Invalid pagination parameters")).WithDetails(err.Error()))
		return
	}

	tagUUIDs := parseCommaSeparatedTagIDs(c.Query("tag_ids"))
	var legacyTagSeqID int64
	tagIDStr := c.Query("tag_id")
	if tagIDStr != "" {
		var err error
		legacyTagSeqID, err = strconv.ParseInt(tagIDStr, 10, 64)
		if err != nil {
			c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "tag_id tam sayı olmalıdır", "tag_id must be an integer")))
			return
		}
	}
	keyword := secutils.SanitizeForLog(c.Query("keyword"))
	searchField := secutils.SanitizeForLog(c.Query("search_field"))
	sortOrder := secutils.SanitizeForLog(c.Query("sort_order"))
	isEnabled, err := parseOptionalFAQEnabled(c)
	if err != nil {
		c.Error(err)
		return
	}

	result, err := h.knowledgeService.ListFAQEntries(ctx, kbID, &page, tagUUIDs, legacyTagSeqID, keyword, searchField, sortOrder, isEnabled)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// parseOptionalFAQEnabled preserves the distinction between an omitted filter
// and an explicit false value so the management endpoint remains backward compatible.
func parseOptionalFAQEnabled(c *gin.Context) (*bool, error) {
	raw, exists := c.GetQuery("is_enabled")
	if !exists {
		return nil, nil
	}

	var value bool
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		value = true
	case "false":
		value = false
	default:
		return nil, errors.NewBadRequestError("is_enabled must be true or false")
	}
	return &value, nil
}

// UpsertEntries godoc
// @Summary      FAQ girişlerini toplu güncelle/ekle
// @Description  SSS girdilerini eşzamansız olarak toplu günceller veya ekler. dry_run modunu destekler (dry_run=true); bu modda içe aktarma yapılmadan yalnızca eşzamansız doğrulama yapılır.
// @Description  dry_run modu eşzamansız bir işlemdir; task_id döndürür, ilerleme ve sonuçlar /faq/import/progress/{task_id} üzerinden sorgulanır.
// @Description  Doğrulama şunları içerir: 1) girişlerin temel biçimi 2) yinelenen sorular (toplu işlem içinde ve bilgi bankasında mevcut olanlar) 3) içerik güvenliği denetimi.
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "Bilgi bankası ID'si"
// @Param        request  body      types.FAQBatchUpsertPayload  true  "Toplu işlem isteği"
// @Success      200      {object}  map[string]interface{}    "Görev ID'si"
// @Failure      400      {object}  errors.AppError           "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries [post]
func (h *FAQHandler) UpsertEntries(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req types.FAQBatchUpsertPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ upsert payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	taskID, err := h.knowledgeService.UpsertFAQEntries(ctx, kbID, &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"task_id": taskID,
		},
	})
}

// CreateEntry godoc
// @Summary      Tek bir FAQ girişi oluştur
// @Description  Tek bir FAQ girişini eşzamanlı olarak oluşturur
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "Bilgi bankası kimliği"
// @Param        request  body      types.FAQEntryPayload true  "FAQ girdisi"
// @Success      200      {object}  map[string]interface{}  "Oluşturulan FAQ girdisi"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entry [post]
func (h *FAQHandler) CreateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req types.FAQEntryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	entry, err := h.knowledgeService.CreateFAQEntry(ctx, kbID, &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// UpdateEntry godoc
// @Summary      FAQ girdisini güncelle
// @Description  Belirtilen FAQ girdisini güncelle
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id        path      string                true  "Bilgi bankası kimliği"
// @Param        entry_id  path      int                   true  "FAQ girdi kimliği(seq_id)"
// @Param        request   body      types.FAQEntryPayload true  "FAQ girdisi"
// @Success      200       {object}  map[string]interface{}  "Başarıyla güncellendi"
// @Failure      400       {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/{entry_id} [put]
func (h *FAQHandler) UpdateEntry(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req types.FAQEntryPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	entrySeqID, err := strconv.ParseInt(c.Param("entry_id"), 10, 64)
	if err != nil {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "entry_id tam sayı olmalıdır", "entry_id must be an integer")))
		return
	}

	entry, err := h.knowledgeService.UpdateFAQEntry(ctx, kbID, entrySeqID, &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// UpdateEntryTagBatch godoc
// @Summary      FAQ etiketlerini toplu güncelle
// @Description  FAQ girdilerinin etiketlerini toplu güncelle
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Bilgi bankası kimliği"
// @Param        request  body      object  true  "Etiket güncelleme isteği"
// @Success      200      {object}  map[string]interface{}  "Başarıyla güncellendi"
// @Failure      400      {object}  errors.AppError         "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/tags [put]
func (h *FAQHandler) UpdateEntryTagBatch(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req faqEntryTagBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry tag batch payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}
	if err := h.knowledgeService.UpdateFAQEntryTagBatch(ctx, kbID, req.Updates); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// UpdateEntryFieldsBatch godoc
// @Summary      FAQ alanlarını toplu güncelle
// @Description  FAQ girdilerinin birden fazla alanını toplu güncelle（is_enabled, is_recommended, tag_id）
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string                        true  "Bilgi bankası kimliği"
// @Param        request  body      types.FAQEntryFieldsBatchUpdate  true  "Alan güncelleme isteği"
// @Success      200      {object}  map[string]interface{}        "Başarıyla güncellendi"
// @Failure      400      {object}  errors.AppError               "Hatalı istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/fields [put]
func (h *FAQHandler) UpdateEntryFieldsBatch(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req types.FAQEntryFieldsBatchUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ entry fields batch payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}
	if err := h.knowledgeService.UpdateFAQEntryFieldsBatch(ctx, kbID, &req); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// DeleteEntries godoc
// @Summary      FAQ girdilerini toplu sil
// @Description  Belirtilen FAQ girdilerini toplu sil
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Bilgi tabanı kimliği"
// @Param        request  body      object{ids=[]int}  true  "Silinecek SSS kimlikleri listesi (seq_id)"
// @Success      200      {object}  map[string]interface{}  "Başarıyla silindi"
// @Failure      400      {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries [delete]
func (h *FAQHandler) DeleteEntries(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req faqDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(ctx, "Failed to bind FAQ delete payload: %s", secutils.SanitizeForLog(err.Error()))
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	if err := h.knowledgeService.DeleteFAQEntries(ctx, kbID, req.IDs); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// SearchFAQ godoc
// @Summary      SSS ara
// @Description  SSS'lerde karma arama kullanarak arama yapar; iki seviyeli öncelik etiketi geri çağırmasını destekler: first_priority_tag_ids en yüksek önceliğe, second_priority_tag_ids ise ikinci önceliğe sahiptir
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "Bilgi tabanı kimliği"
// @Param        request  body      types.FAQSearchRequest  true  "Arama isteği"
// @Success      200      {object}  map[string]interface{}  "Arama sonuçları"
// @Failure      400      {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/search [post]
func (h *FAQHandler) SearchFAQ(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req types.FAQSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind FAQ search payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}
	req.QueryText = secutils.SanitizeForLog(req.QueryText)
	if req.MatchCount <= 0 {
		req.MatchCount = 10
	}
	if req.MatchCount > 200 {
		req.MatchCount = 200
	}
	entries, err := h.knowledgeService.SearchFAQEntries(ctx, kbID, &req)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entries,
	})
}

// ExportEntries godoc
// @Summary      SSS kayıtlarını dışa aktar
// @Description  Tüm SSS kayıtlarını CSV (varsayılan) veya JSON olarak dışa aktarır. ?format=json, FAQEntryPayload yapısıyla uyumlu bir dizi döndürür.
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      text/csv
// @Produce      application/json
// @Param        id      path      string  true   "Bilgi tabanı kimliği"
// @Param        format  query     string  false  "Dışa aktarma biçimi: csv (varsayılan) veya json"
// @Success      200     {file}    file    "Dışa aktarılan dosya"
// @Failure      400     {object}  errors.AppError  "Geçersiz istek parametreleri"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/export [get]
func (h *FAQHandler) ExportEntries(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))
	format := strings.ToLower(strings.TrimSpace(c.Query("format")))

	if format == "json" {
		jsonData, err := h.knowledgeService.ExportFAQEntriesJSON(ctx, kbID)
		if err != nil {
			logger.ErrorWithFields(ctx, err, nil)
			c.Error(err)
			return
		}
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=faq_export.json")
		c.Data(http.StatusOK, "application/json; charset=utf-8", jsonData)
		return
	}

	csvData, err := h.knowledgeService.ExportFAQEntries(ctx, kbID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	// Set response headers for CSV download
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=faq_export.csv")
	// Add BOM for Excel compatibility with UTF-8
	bom := []byte{0xEF, 0xBB, 0xBF}
	c.Data(http.StatusOK, "text/csv; charset=utf-8", append(bom, csvData...))
}

// GetEntry godoc
// @Summary      SSS kaydı ayrıntılarını al
// @Description  Kimliğe göre tek bir SSS kaydının ayrıntılarını alır
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id        path      string  true  "Bilgi tabanı kimliği"
// @Param        entry_id  path      int     true  "SSS kayıt kimliği (seq_id)"
// @Success      200       {object}  map[string]interface{}  "SSS kayıt ayrıntıları"
// @Failure      400       {object}  errors.AppError         "Geçersiz istek parametreleri"
// @Failure      404       {object}  errors.AppError         "Kayıt bulunamadı"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/{entry_id} [get]
func (h *FAQHandler) GetEntry(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	entrySeqID, err := strconv.ParseInt(c.Param("entry_id"), 10, 64)
	if err != nil {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "entry_id tam sayı olmalıdır", "entry_id must be an integer")))
		return
	}

	entry, err := h.knowledgeService.GetFAQEntry(ctx, kbID, entrySeqID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}

// GetImportProgress godoc
// @Summary      SSS içe aktarma ilerlemesini al
// @Description  SSS içe aktarma görevinin ilerlemesini alır
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        task_id  path      string  true  "Görev ID'si"
// @Success      200      {object}  map[string]interface{}  "İçe aktarma ilerlemesi"
// @Failure      404      {object}  errors.AppError         "Görev mevcut değil"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /faq/import/progress/{task_id} [get]
func (h *FAQHandler) GetImportProgress(c *gin.Context) {
	ctx := c.Request.Context()
	taskID := secutils.SanitizeForLog(c.Param("task_id"))
	if err := requireTaskProgressTenant(ctx, taskID); err != nil {
		c.Error(err)
		return
	}

	progress, err := h.knowledgeService.GetFAQImportProgress(ctx, taskID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    progress,
	})
}

// UpdateLastImportResultDisplayStatus godoc
// @Summary      FAQ son içe aktarma sonucu görüntüleme durumunu güncelle
// @Description  FAQ bilgi tabanı içe aktarma sonuç istatistik kartının gösterilme veya gizlenme durumunu güncelle
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id      path      string                                         true  "Bilgi tabanı ID'si"
// @Param        request body      updateLastFAQImportResultDisplayStatusRequest  true  "Durum güncelleme isteği"
// @Success      200     {object}  map[string]interface{}                         "Güncelleme başarılı"
// @Failure      400     {object}  errors.AppError                                "İstek parametresi hatası"
// @Failure      404     {object}  errors.AppError                                "Bilgi tabanı mevcut değil veya içe aktarma kaydı yok"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/import/last-result/display [put]
func (h *FAQHandler) UpdateLastImportResultDisplayStatus(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req updateLastFAQImportResultDisplayStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind display status update payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	if err := h.knowledgeService.UpdateLastFAQImportResultDisplayStatus(ctx, kbID, req.DisplayStatus); err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// AddSimilarQuestions godoc
// @Summary      Benzer soru ekle
// @Description  Belirtilen FAQ girdisine benzer sorular ekle
// @Tags         SSS Yönetimi
// @Accept       json
// @Produce      json
// @Param        id        path      string                      true  "Bilgi tabanı ID'si"
// @Param        entry_id  path      int                         true  "FAQ girdi ID'si(seq_id)"
// @Param        request   body      addSimilarQuestionsRequest  true  "Benzer soru listesi"
// @Success      200       {object}  map[string]interface{}      "Güncellenmiş FAQ girdisi"
// @Failure      400       {object}  errors.AppError             "İstek parametresi hatası"
// @Failure      404       {object}  errors.AppError             "Girdi mevcut değil"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/faq/entries/{entry_id}/similar-questions [post]
func (h *FAQHandler) AddSimilarQuestions(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	entrySeqID, err := strconv.ParseInt(c.Param("entry_id"), 10, 64)
	if err != nil {
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "entry_id tam sayı olmalıdır", "entry_id must be an integer")))
		return
	}

	var req addSimilarQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind add similar questions payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	entry, err := h.knowledgeService.AddSimilarQuestions(ctx, kbID, entrySeqID, req.SimilarQuestions)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    entry,
	})
}
