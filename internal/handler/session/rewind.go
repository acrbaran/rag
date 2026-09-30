package session

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"

	"github.com/acrbaran/rag/internal/application/service"
	"github.com/acrbaran/rag/internal/errors"
	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/gin-gonic/gin"
)

// sessionRewinder is the rewind surface the handler needs. Declaring it here
// rather than depending on *service.SessionRewindService keeps the handler
// testable with a stub.
type sessionRewinder interface {
	Rewind(
		ctx context.Context,
		tenantID uint64,
		userID, sessionID, messageID string,
	) (*service.RewindResult, error)
}

// RewindSessionRequest is the rewind endpoint's body.
type RewindSessionRequest struct {
	// MessageID is the message to rewind to. A user message deletes itself and
	// everything after (the client prefills that question). An assistant
	// message keeps itself and deletes everything after.
	MessageID string `json:"message_id" binding:"required"`
}

// RewindSession godoc
// @Summary      Oturumu geri al
// @Description  Mevcut oturumu belirtilen kullanıcı veya asistan mesajına geri alır: sonraki mesajları siler ve mümkünse çalışma alanında git reset uygular.
// @Tags         Oturumlar
// @Accept       json
// @Produce      json
// @Param        session_id  path      string                true  "Oturum ID"
// @Param        request     body      RewindSessionRequest  true  "Geri alma isteği"
// @Success      200         {object}  map[string]interface{}  "Geri alma sonucu"
// @Failure      400         {object}  errors.AppError         "Geçersiz istek parametreleri / geri alma noktası rolü desteklenmiyor"
// @Failure      404         {object}  errors.AppError         "Oturum veya mesaj bulunamadı"
// @Failure      409         {object}  errors.AppError         "Oturum şu anda oluşturuluyor / kontrol noktası yok / sandbox değiştirildi"
// @Failure      500         {object}  errors.AppError         "Çalışma alanı geri alma başarısız"
// @Security     Bearer
// @Security     ApiKeyAuth
// @Router       /sessions/{session_id}/rewind [post]
func (h *Handler) RewindSession(c *gin.Context) {
	ctx := c.Request.Context()

	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		sessionID = strings.TrimSpace(c.Param("id"))
	}
	if sessionID == "" {
		_ = c.Error(errors.NewBadRequestError("session ID is required"))
		return
	}

	var req RewindSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError("message_id is required"))
		return
	}
	if strings.TrimSpace(req.MessageID) == "" {
		_ = c.Error(errors.NewBadRequestError("message_id is required"))
		return
	}

	if h.rewindService == nil {
		_ = c.Error(errors.NewServiceUnavailableError("session rewind is not available"))
		return
	}

	tenantID, _ := types.TenantIDFromContext(ctx)
	userID := types.SessionOwnerIDFromContext(ctx)

	result, err := h.rewindService.Rewind(
		ctx, tenantID, userID, sessionID, strings.TrimSpace(req.MessageID),
	)
	if err != nil {
		if stderrors.Is(err, service.ErrRewindSourceBusy) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "session has an active turn",
				"code":    "REWIND_SOURCE_BUSY",
			})
			return
		}
		if stderrors.Is(err, service.ErrRewindNoCheckpoint) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "no reachable workspace checkpoint",
				"code":    "REWIND_NO_CHECKPOINT",
			})
			return
		}
		if stderrors.Is(err, service.ErrRewindSandboxReplaced) {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "sandbox was replaced",
				"code":    "REWIND_SANDBOX_REPLACED",
			})
			return
		}
		if stderrors.Is(err, service.ErrRewindSessionNotFound) {
			_ = c.Error(errors.NewNotFoundError("session not found"))
			return
		}
		if stderrors.Is(err, service.ErrRewindMessageNotFound) {
			_ = c.Error(errors.NewNotFoundError("message not found"))
			return
		}
		if stderrors.Is(err, service.ErrRewindMessageRole) {
			_ = c.Error(errors.NewBadRequestError("rewind point must be a user or assistant message"))
			return
		}
		logger.Errorf(ctx, "rewind session %s failed: %v", sessionID, err)
		_ = c.Error(errors.NewInternalServerError("rewind session failed"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
