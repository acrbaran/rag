package handler

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"strconv"

	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
)

// TagHandler handles knowledge base tag operations.
//
// All KB-access checks (own / org-shared / via shared agent) are now
// performed by the route-level g.KBAccessRead / g.KBAccessWrite
// guards in router.go — the guard rewrites c.Request.Context() to
// carry the effective tenant ID, so handlers below just use
// c.Request.Context() the way they always did.
type TagHandler struct {
	tagService interfaces.KnowledgeTagService
	tagRepo    interfaces.KnowledgeTagRepository
	chunkRepo  interfaces.ChunkRepository
}

// DeleteTagRequest represents the request body for deleting a tag
type DeleteTagRequest struct {
	ExcludeIDs []int64 `json:"exclude_ids"` // Chunk seq_ids to exclude from deletion
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(
	tagService interfaces.KnowledgeTagService,
	tagRepo interfaces.KnowledgeTagRepository,
	chunkRepo interfaces.ChunkRepository,
) *TagHandler {
	return &TagHandler{tagService: tagService, tagRepo: tagRepo, chunkRepo: chunkRepo}
}

// resolveTagID resolves tag_id parameter which can be either UUID or seq_id (integer).
// Uses tenant from c's context — which the route-level KB-access guard
// has already rewritten to the effective tenant for shared KBs.
func (h *TagHandler) resolveTagID(c *gin.Context) (string, error) {
	return h.resolveTagIDWithCtx(c, c.Request.Context())
}

// resolveTagIDWithCtx resolves tag_id using the given context for tenant.
func (h *TagHandler) resolveTagIDWithCtx(c *gin.Context, ctx context.Context) (string, error) {
	tagIDParam := secutils.SanitizeForLog(c.Param("tag_id"))

	if seqID, err := strconv.ParseInt(tagIDParam, 10, 64); err == nil {
		tenantID := types.MustTenantIDFromContext(ctx)
		tag, err := h.tagRepo.GetBySeqID(ctx, tenantID, seqID)
		if err != nil {
			return "", errors.NewNotFoundError(types.LocalizedText(ctx, "Etiket bulunamadı", "Tag not found"))
		}
		return tag.ID, nil
	}
	return tagIDParam, nil
}

// getChunksBySeqIDs retrieves chunks by their seq_ids.
func (h *TagHandler) getChunksBySeqIDs(ctx context.Context, tenantID uint64, seqIDs []int64) ([]*types.Chunk, error) {
	return h.chunkRepo.ListChunksBySeqID(ctx, tenantID, seqIDs)
}

// ListTags godoc
// @Summary      Etiket listesini al
// @Description  Bilgi tabanı altındaki tüm etiketleri ve istatistik bilgilerini alır
// @Tags         Etiket Yönetimi
// @Accept       json
// @Produce      json
// @Param        id         path      string  true   "Bilgi tabanı kimliği"
// @Param        page       query     int     false  "Sayfa numarası"
// @Param        page_size  query     int     false  "Sayfa başına öğe sayısı"
// @Param        keyword    query     string  false  "Anahtar kelime araması"
// @Success      200        {object}  map[string]interface{}  "Etiket listesi"
// @Failure      400        {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/tags [get]
func (h *TagHandler) ListTags(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var page types.Pagination
	if err := c.ShouldBindQuery(&page); err != nil {
		logger.Error(ctx, "Failed to bind pagination query", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Sayfalama parametreleri geçersiz", "Invalid pagination parameters")).WithDetails(err.Error()))
		return
	}

	keyword := secutils.SanitizeForLog(c.Query("keyword"))

	tags, err := h.tagService.ListTags(ctx, kbID, &page, keyword)
	if err != nil {
		logger.ErrorWithFields(ctx, err, nil)
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tags,
	})
}

type createTagRequest struct {
	Name      string `json:"name"       binding:"required"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// CreateTag godoc
// @Summary      Etiket oluştur
// @Description  Bilgi tabanı altında yeni bir etiket oluşturur
// @Tags         Etiket Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Bilgi tabanı kimliği"
// @Param        request  body      object{name=string,color=string,sort_order=int}  true  "Etiket bilgisi"
// @Success      200      {object}  map[string]interface{}  "Oluşturulan etiket"
// @Failure      400      {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/tags [post]
func (h *TagHandler) CreateTag(c *gin.Context) {
	ctx := c.Request.Context()
	kbID := secutils.SanitizeForLog(c.Param("id"))

	var req createTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind create tag payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	tag, err := h.tagService.CreateTag(ctx, kbID,
		secutils.SanitizeForLog(req.Name), secutils.SanitizeForLog(req.Color), req.SortOrder)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"kb_id": kbID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

type updateTagRequest struct {
	Name      *string `json:"name"`
	Color     *string `json:"color"`
	SortOrder *int    `json:"sort_order"`
}

// UpdateTag godoc
// @Summary      Etiketi güncelle
// @Description  Etiket bilgilerini günceller
// @Tags         Etiket Yönetimi
// @Accept       json
// @Produce      json
// @Param        id       path      string  true  "Bilgi tabanı kimliği"
// @Param        tag_id   path      string  true  "Etiket kimliği (UUID veya seq_id)"
// @Param        request  body      object  true  "Etiket güncelleme bilgisi"
// @Success      200      {object}  map[string]interface{}  "Güncellenmiş etiket"
// @Failure      400      {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/tags/{tag_id} [put]
func (h *TagHandler) UpdateTag(c *gin.Context) {
	ctx := c.Request.Context()

	tagID, err := h.resolveTagID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req updateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(ctx, "Failed to bind update tag payload", err)
		c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "İstek parametreleri geçersiz", "Invalid request parameters")).WithDetails(err.Error()))
		return
	}

	tag, err := h.tagService.UpdateTag(ctx, tagID, req.Name, req.Color, req.SortOrder)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tag_id": tagID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tag,
	})
}

// DeleteTag godoc
// @Summary      Etiketi sil
// @Description  Etiketi siler; başvurulan etiketleri force=true ile zorla silebilir, content_only=true ile yalnızca etiket altındaki içerikleri silip etiketin kendisini koruyabilirsiniz
// @Tags         Etiket Yönetimi
// @Accept       json
// @Produce      json
// @Param        id            path      string              true   "Bilgi tabanı kimliği"
// @Param        tag_id        path      string              true   "Etiket kimliği (UUID veya seq_id)"
// @Param        force         query     bool                false  "Zorla sil"
// @Param        content_only  query     bool                false  "Yalnızca içeriği sil, etiketi koru"
// @Param        body          body      DeleteTagRequest    false  "Silme seçenekleri"
// @Success      200           {object}  map[string]interface{}  "Başarıyla silindi"
// @Failure      400           {object}  errors.AppError         "İstek parametresi hatası"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /knowledge-bases/{id}/tags/{tag_id} [delete]
func (h *TagHandler) DeleteTag(c *gin.Context) {
	ctx := c.Request.Context()

	tagID, err := h.resolveTagID(c)
	if err != nil {
		c.Error(err)
		return
	}

	force := c.Query("force") == "true"
	contentOnly := c.Query("content_only") == "true"

	var req DeleteTagRequest
	if err := c.ShouldBindJSON(&req); err != nil && !stderrors.Is(err, io.EOF) {
		_ = c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Silme seçenekleri geçersiz", "Invalid deletion options")))
		return
	}

	var excludeUUIDs []string
	if len(req.ExcludeIDs) > 0 {
		tenantID := types.MustTenantIDFromContext(ctx)
		wanted := make(map[int64]bool, len(req.ExcludeIDs))
		for _, id := range req.ExcludeIDs {
			if id <= 0 {
				_ = c.Error(errors.NewBadRequestError(types.LocalizedText(ctx, "Hariç tutulacak kayıt kimliği pozitif tam sayı olmalıdır", "Excluded entry ID must be a positive integer")))
				return
			}
			wanted[id] = true
		}
		chunks, err := h.getChunksBySeqIDs(ctx, tenantID, req.ExcludeIDs)
		if err != nil {
			_ = c.Error(err)
			return
		}
		for _, chunk := range chunks {
			if chunk == nil || !wanted[chunk.SeqID] {
				continue
			}
			if chunk.TenantID != tenantID || chunk.KnowledgeBaseID != c.Param("id") ||
				chunk.ChunkType != types.ChunkTypeFAQ {
				_ = c.Error(errors.NewForbiddenError(types.LocalizedText(ctx, "Hariç tutulan kayıt bu bilgi tabanına ait değil", "Excluded entry does not belong to this knowledge base")))
				return
			}
			excludeUUIDs = append(excludeUUIDs, chunk.ID)
			delete(wanted, chunk.SeqID)
		}
		if len(wanted) != 0 {
			_ = c.Error(errors.NewNotFoundError(types.LocalizedText(ctx, "Hariç tutulacak kayıt bulunamadı", "Excluded entry not found")))
			return
		}
	}

	if err := h.tagService.DeleteTag(ctx, tagID, force, contentOnly, excludeUUIDs); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tag_id": tagID,
		})
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// NOTE: TagHandler currently exposes CRUD for tags and statistics.
// Knowledge / Chunk tagging is handled via dedicated knowledge and FAQ APIs.
