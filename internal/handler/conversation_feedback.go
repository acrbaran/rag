package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ConversationFeedbackHandler struct {
	db       *gorm.DB
	sessions interfaces.SessionService
}

func NewConversationFeedbackHandler(db *gorm.DB, sessions interfaces.SessionService) *ConversationFeedbackHandler {
	return &ConversationFeedbackHandler{db: db, sessions: sessions}
}

type ConversationFeedback struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"uniqueIndex"`
	TenantID  uint64    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Helpful   *bool     `json:"helpful"`
	Category  *string   `json:"category"`
	Comment   *string   `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ConversationFeedback) TableName() string { return "conversation_feedback" }

func (h *ConversationFeedbackHandler) ownSession(c *gin.Context) (string, bool) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback unavailable"})
		return "", false
	}
	id := c.Param("id")
	session, err := h.sessions.GetSession(c.Request.Context(), id)
	if err != nil || session == nil || session.UserID != types.SessionOwnerIDFromContext(c.Request.Context()) {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return "", false
	}
	return id, true
}

func (h *ConversationFeedbackHandler) GetOwn(c *gin.Context) {
	id, ok := h.ownSession(c)
	if !ok {
		return
	}
	var row ConversationFeedback
	err := h.db.WithContext(c.Request.Context()).Where("session_id = ?", id).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feedback load failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": row})
}

func (h *ConversationFeedbackHandler) SaveOwn(c *gin.Context) {
	id, ok := h.ownSession(c)
	if !ok {
		return
	}
	var fields map[string]json.RawMessage
	if err := c.ShouldBindJSON(&fields); err != nil || len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feedback"})
		return
	}
	updates := map[string]interface{}{}
	for key, raw := range fields {
		switch key {
		case "helpful":
			if string(raw) == "null" {
				updates[key] = nil
				continue
			}
			var value bool
			if json.Unmarshal(raw, &value) != nil || (string(raw) != "true" && string(raw) != "false") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rating"})
				return
			}
			updates[key] = value
		case "category":
			var value string
			if json.Unmarshal(raw, &value) != nil || (value != "suggestion" && value != "complaint") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
				return
			}
			updates[key] = value
		case "comment":
			var value string
			if json.Unmarshal(raw, &value) != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment"})
				return
			}
			value = strings.TrimSpace(value)
			if value == "" || utf8.RuneCountInString(value) > 2000 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "comment must be 1-2000 characters"})
				return
			}
			updates[key] = value
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown feedback field"})
			return
		}
	}
	if _, exists := updates["category"]; exists {
		if _, ok := updates["comment"]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "comment required"})
			return
		}
	}
	if _, exists := updates["comment"]; exists {
		if _, ok := updates["category"]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category required"})
			return
		}
	}
	userID := types.SessionOwnerIDFromContext(c.Request.Context())
	tenantID := types.MustTenantIDFromContext(c.Request.Context())
	row := ConversationFeedback{ID: uuid.NewString(), SessionID: id, TenantID: tenantID, UserID: userID}
	err := h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}}, DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		updates["updated_at"] = time.Now()
		return tx.Model(&ConversationFeedback{}).Where("session_id = ? AND tenant_id = ? AND user_id = ?", id, tenantID, userID).Updates(updates).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feedback save failed"})
		return
	}
	var saved ConversationFeedback
	if err := h.db.WithContext(c.Request.Context()).Where("session_id = ?", id).First(&saved).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feedback load failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": saved})
}

type feedbackListRow struct {
	ConversationFeedback `gorm:"embedded"`
	SessionTitle         string `json:"session_title"`
	UserName             string `json:"user_name"`
	UserEmail            string `json:"user_email"`
	WorkspaceName        string `json:"workspace_name"`
	MessageExcerpt       string `json:"message_excerpt"`
}

func (h *ConversationFeedbackHandler) feedbackQuery(c *gin.Context, global bool) *gorm.DB {
	query := h.db.WithContext(c.Request.Context()).Table("conversation_feedback AS f").
		Joins("JOIN sessions AS s ON s.id = f.session_id").
		Joins("LEFT JOIN users AS u ON u.id = f.user_id").
		Joins("LEFT JOIN tenants AS t ON t.id = f.tenant_id")
	if !global {
		query = query.Where("f.tenant_id = ?", types.MustTenantIDFromContext(c.Request.Context()))
	}
	if value := c.Query("helpful"); value == "true" || value == "false" {
		query = query.Where("f.helpful = ?", value == "true")
	}
	if value := c.Query("category"); value == "suggestion" || value == "complaint" {
		query = query.Where("f.category = ?", value)
	}
	if value := c.Query("workspace_id"); global && value != "" {
		query = query.Where("f.tenant_id = ?", value)
	}
	if value := strings.TrimSpace(c.Query("q")); value != "" {
		value = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(value))
		pattern := "%" + value + "%"
		query = query.Where(`(LOWER(COALESCE(f.comment, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(s.title, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(u.username, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(u.email, '')) LIKE ? ESCAPE '\')`,
			pattern, pattern, pattern, pattern, pattern)
	}
	if value, _, ok := parseFeedbackDate(c.Query("from")); ok {
		query = query.Where("f.updated_at >= ?", value)
	}
	if value, dateOnly, ok := parseFeedbackDate(c.Query("to")); ok {
		if dateOnly {
			value = value.AddDate(0, 0, 1)
		}
		query = query.Where("f.updated_at < ?", value)
	}
	return query
}

// parseFeedbackDate accepts an RFC3339 instant (the UI sends the user's local
// day boundaries this way) or a plain YYYY-MM-DD date in server time.
func parseFeedbackDate(value string) (time.Time, bool, bool) {
	if value == "" {
		return time.Time{}, false, false
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, false, true
	}
	if parsed, err := time.ParseInLocation(time.DateOnly, value, time.Local); err == nil {
		return parsed, true, true
	}
	return time.Time{}, false, false
}

func validFeedbackDateFilters(c *gin.Context) bool {
	for _, key := range []string{"from", "to"} {
		if value := c.Query(key); value != "" {
			if _, _, ok := parseFeedbackDate(value); !ok {
				return false
			}
		}
	}
	return true
}

func (h *ConversationFeedbackHandler) ListTenant(c *gin.Context) { h.list(c, false) }
func (h *ConversationFeedbackHandler) ListGlobal(c *gin.Context) { h.list(c, true) }

func (h *ConversationFeedbackHandler) list(c *gin.Context, global bool) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback unavailable"})
		return
	}
	if !validFeedbackDateFilters(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date filter"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if size < 1 || size > 100 {
		size = 20
	}
	query := h.feedbackQuery(c, global)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feedback load failed"})
		return
	}
	items := make([]feedbackListRow, 0)
	err := h.feedbackQuery(c, global).Select("f.*, s.title AS session_title, COALESCE(NULLIF(u.first_name || ' ' || u.last_name, ' '), u.username, f.user_id) AS user_name, COALESCE(u.email, '') AS user_email, COALESCE(t.name, '') AS workspace_name, '' AS message_excerpt").
		Order("f.updated_at DESC, f.id DESC").Limit(size).Offset((page - 1) * size).Scan(&items).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feedback load failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "total": total, "page": page, "page_size": size}})
}

func (h *ConversationFeedbackHandler) DetailTenant(c *gin.Context) { h.detail(c, false) }
func (h *ConversationFeedbackHandler) DetailGlobal(c *gin.Context) { h.detail(c, true) }

func (h *ConversationFeedbackHandler) detail(c *gin.Context, global bool) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "feedback unavailable"})
		return
	}
	var row feedbackListRow
	err := h.feedbackQuery(c, global).Select("f.*, s.title AS session_title, COALESCE(NULLIF(u.first_name || ' ' || u.last_name, ' '), u.username, f.user_id) AS user_name, COALESCE(u.email, '') AS user_email, COALESCE(t.name, '') AS workspace_name").Where("f.id = ?", c.Param("id")).Scan(&row).Error
	if err != nil || row.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found"})
		return
	}
	messages := make([]struct {
		ID        string    `json:"id"`
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
	}, 0)
	if err := h.db.WithContext(c.Request.Context()).Table("messages").Select("id, role, content, created_at").Where("session_id = ? AND deleted_at IS NULL", row.SessionID).Order("created_at ASC, id ASC").Scan(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "conversation load failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"feedback": row, "messages": messages}})
}
