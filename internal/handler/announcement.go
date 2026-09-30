package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/acrbaran/rag/internal/logger"
	"github.com/acrbaran/rag/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AnnouncementHandler serves system announcements: the system-admin editor
// (drafts, publishing, audience targeting, viewers) and every user's
// notification feed and top banner state.
type AnnouncementHandler struct {
	db *gorm.DB
}

func NewAnnouncementHandler(db *gorm.DB) *AnnouncementHandler {
	return &AnnouncementHandler{db: db}
}

type announcementRow struct {
	ID           string  `gorm:"column:id;primaryKey"`
	Status       string  `gorm:"column:status"`
	Version      int     `gorm:"column:version"`
	TypeID       string  `gorm:"column:type_id"`
	Title        string  `gorm:"column:title"`
	Body         string  `gorm:"column:body"`
	StartsAt     *int64  `gorm:"column:starts_at"`
	EndsAt       *int64  `gorm:"column:ends_at"`
	ShowBanner   bool    `gorm:"column:show_banner"`
	BannerColor  string  `gorm:"column:banner_color"`
	AudienceMode string  `gorm:"column:audience_mode"`
	AudienceIDs  string  `gorm:"column:audience_ids"`
	CreatedBy    string  `gorm:"column:created_by"`
	PublishedBy  *string `gorm:"column:published_by"`
	PublishedAt  *int64  `gorm:"column:published_at"`
	CreatedAt    int64   `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt    int64   `gorm:"column:updated_at;autoUpdateTime:milli"`
}

func (announcementRow) TableName() string { return "announcements" }

type announcementTypeRow struct {
	ID             string `gorm:"column:id;primaryKey"`
	Name           string `gorm:"column:name"`
	NormalizedName string `gorm:"column:normalized_name"`
	CreatedBy      string `gorm:"column:created_by"`
	CreatedAt      int64  `gorm:"column:created_at;autoCreateTime:milli"`
}

func (announcementTypeRow) TableName() string { return "announcement_types" }

type announcementStateRow struct {
	ID               string `gorm:"column:id;primaryKey"`
	AnnouncementID   string `gorm:"column:announcement_id"`
	UserID           string `gorm:"column:user_id"`
	Phase            string `gorm:"column:phase"`
	IsRead           bool   `gorm:"column:is_read"`
	Deleted          bool   `gorm:"column:deleted"`
	DismissedVersion *int   `gorm:"column:dismissed_version"`
	FirstSeenAt      *int64 `gorm:"column:first_seen_at"`
}

func (announcementStateRow) TableName() string { return "announcement_notification_states" }

type announcementIdempotencyRow struct {
	ID             string `gorm:"column:id;primaryKey"`
	ActorID        string `gorm:"column:actor_id"`
	IdempotencyKey string `gorm:"column:idempotency_key"`
	Action         string `gorm:"column:action"`
	RequestHash    string `gorm:"column:request_hash"`
	ResourceID     string `gorm:"column:resource_id"`
	CreatedAt      int64  `gorm:"column:created_at;autoCreateTime:milli"`
}

func (announcementIdempotencyRow) TableName() string { return "announcement_idempotency" }

type announcementAudience struct {
	Mode string   `json:"mode"`
	IDs  []string `json:"ids"`
}

type announcementTypePayload struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Builtin bool   `json:"builtin"`
}

type announcementCounts struct {
	Users           int64 `json:"users"`
	SubscribedUsers int64 `json:"subscribedUsers"`
	Devices         int64 `json:"devices"`
}

type announcementDelivery struct {
	Pending   int `json:"pending"`
	Accepted  int `json:"accepted"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

type announcementState struct {
	Read             bool `json:"read"`
	Deleted          bool `json:"deleted"`
	DismissedVersion *int `json:"dismissedVersion"`
}

type announcementPayload struct {
	ID                  string                       `json:"id"`
	TypeID              string                       `json:"typeId"`
	Type                announcementTypePayload      `json:"type"`
	Status              string                       `json:"status"`
	Version             int                          `json:"version"`
	Title               string                       `json:"title"`
	Body                string                       `json:"body"`
	ShowBanner          bool                         `json:"showBanner"`
	BannerColor         string                       `json:"bannerColor"`
	StartsAt            *int64                       `json:"startsAt"`
	EndsAt              *int64                       `json:"endsAt"`
	PublishedAt         *int64                       `json:"publishedAt"`
	ResolvedAt          *int64                       `json:"resolvedAt"`
	ExpiresAt           *int64                       `json:"expiresAt"`
	CreatedAt           int64                        `json:"createdAt"`
	CreatedBy           string                       `json:"createdBy"`
	CreatedByName       *string                      `json:"createdByName"`
	CreatedByEmail      *string                      `json:"createdByEmail"`
	PublishedBy         *string                      `json:"publishedBy"`
	PublishedByName     *string                      `json:"publishedByName"`
	PublishedByEmail    *string                      `json:"publishedByEmail"`
	ResolvedBy          *string                      `json:"resolvedBy"`
	ResolutionTitle     *string                      `json:"resolutionTitle"`
	ResolutionBody      *string                      `json:"resolutionBody"`
	Audience            *announcementAudience        `json:"audience,omitempty"`
	AudienceCounts      *announcementCounts          `json:"audienceCounts,omitempty"`
	Delivery            *announcementDelivery        `json:"delivery,omitempty"`
	MaintenanceDelivery *announcementDelivery        `json:"maintenanceDelivery,omitempty"`
	ResolutionDelivery  *announcementDelivery        `json:"resolutionDelivery,omitempty"`
	ViewerCount         *int64                       `json:"viewerCount,omitempty"`
	NotificationState   map[string]announcementState `json:"notificationState,omitempty"`
}

type announcementError struct {
	status  int
	message string
}

func (e *announcementError) Error() string { return e.message }

func announcementInvalid(message string) error {
	return &announcementError{status: http.StatusBadRequest, message: message}
}

func announcementConflict(message string) error {
	return &announcementError{status: http.StatusConflict, message: message}
}

var errAnnouncementNotFound = &announcementError{status: http.StatusNotFound, message: "Announcement not found."}

var (
	announcementBuiltinTypes = []announcementTypePayload{
		{ID: "maintenance", Name: "Maintenance", Builtin: true},
		{ID: "info", Name: "Information", Builtin: true},
		{ID: "update", Name: "Update", Builtin: true},
		{ID: "warning", Name: "Warning", Builtin: true},
	}
	announcementReservedTypeNames = []string{"bakım", "bilgilendirme", "güncelleme", "uyarı"}
	announcementBannerColors      = map[string]bool{"neutral": true, "blue": true, "green": true, "amber": true, "red": true, "violet": true}
	announcementAudienceModes     = map[string]bool{"all": true, "workspaces": true, "roles": true, "users": true}
	announcementRoles             = []string{"system_admin", string(types.TenantRoleOwner), string(types.TenantRoleAdmin), string(types.TenantRoleContributor), string(types.TenantRoleViewer)}
	announcementEditableFields    = map[string]bool{"title": true, "body": true, "startsAt": true, "endsAt": true, "typeId": true, "audience": true, "showBanner": true, "bannerColor": true}
	announcementIdempotencyRE     = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
	announcementPageRE            = regexp.MustCompile(`^[1-9][0-9]{0,5}$`)
	announcementLikeEscaper       = strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)
)

const (
	announcementMaxTimestamp = 8_640_000_000_000_000
	announcementViewerPage   = 50
	announcementUserName     = "COALESCE(NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''), u.username)"
)

func announcementNow() int64 { return time.Now().UnixMilli() }

func (h *AnnouncementHandler) ready(c *gin.Context) bool {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "announcements unavailable"})
		return false
	}
	return true
}

func (h *AnnouncementHandler) fail(c *gin.Context, err error) {
	var known *announcementError
	if errors.As(err, &known) {
		c.JSON(known.status, gin.H{"error": known.message})
		return
	}
	logger.Errorf(c.Request.Context(), "announcement request failed: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "The platform service could not complete the request."})
}

func announcementActor(c *gin.Context) string {
	id, _ := types.UserIDFromContext(c.Request.Context())
	return id
}

func announcementBody(c *gin.Context) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := c.ShouldBindJSON(&fields); err != nil || fields == nil {
		return nil, announcementInvalid("Request body must be a JSON object.")
	}
	return fields, nil
}

func announcementRawString(raw json.RawMessage) (string, bool) {
	trimmed := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(trimmed, `"`) {
		return "", false
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func announcementRawInt(raw json.RawMessage) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	return value, err == nil
}

func announcementRawBool(raw json.RawMessage) (bool, bool) {
	switch strings.TrimSpace(string(raw)) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

func announcementIsNull(raw json.RawMessage) bool {
	return raw == nil || strings.TrimSpace(string(raw)) == "null"
}

func announcementText(raw json.RawMessage, field string, limit int) (string, error) {
	value, ok := announcementRawString(raw)
	value = strings.TrimSpace(value)
	if !ok || value == "" || utf8.RuneCountInString(value) > limit {
		return "", announcementInvalid(field + " is invalid.")
	}
	return value, nil
}

func announcementTimestamp(raw json.RawMessage, field string) (*int64, error) {
	if announcementIsNull(raw) {
		return nil, nil
	}
	value, ok := announcementRawInt(raw)
	if !ok || value < 0 || value > announcementMaxTimestamp {
		return nil, announcementInvalid(field + " is invalid.")
	}
	return &value, nil
}

func announcementVersion(fields map[string]json.RawMessage) (int, error) {
	value, ok := announcementRawInt(fields["version"])
	if !ok || value < 0 || value > 1<<31-1 {
		return 0, announcementInvalid("version is required.")
	}
	return int(value), nil
}

func announcementDecodeAudience(value string) announcementAudience {
	var audience announcementAudience
	_ = json.Unmarshal([]byte(value), &audience.IDs)
	return audience
}

func (row announcementRow) audience() announcementAudience {
	audience := announcementDecodeAudience(row.AudienceIDs)
	audience.Mode = row.AudienceMode
	if audience.Mode == "" {
		audience.Mode = "all"
	}
	if audience.IDs == nil {
		audience.IDs = []string{}
	}
	return audience
}

func announcementInWindow(row announcementRow, now int64) bool {
	return (row.StartsAt == nil || *row.StartsAt <= now) && (row.EndsAt == nil || now < *row.EndsAt)
}

// announcementWorkspaceIDs parses tenant ids; audiences store them as strings
// so every mode shares one shape.
func announcementWorkspaceIDs(ids []string) ([]uint64, bool) {
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		value, err := strconv.ParseUint(id, 10, 64)
		if err != nil {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

// audienceUsers is the set of active accounts an audience selection targets.
// Workspaces and tenant roles count only active memberships of live workspaces.
func audienceUsers(tx *gorm.DB, audience announcementAudience) *gorm.DB {
	query := tx.Table("users").Where("users.deleted_at IS NULL AND users.is_active = ?", true)
	switch audience.Mode {
	case "all":
		return query
	case "users":
		return query.Where("users.id IN ?", audience.IDs)
	case "roles":
		parts, args := []string{}, []interface{}{}
		tenantRoles := []string{}
		for _, role := range audience.IDs {
			if role == "system_admin" {
				parts = append(parts, "users.is_system_admin = ?")
				args = append(args, true)
			} else {
				tenantRoles = append(tenantRoles, role)
			}
		}
		if len(tenantRoles) > 0 {
			parts = append(parts, "users.id IN (SELECT tm.user_id FROM tenant_members tm JOIN tenants t ON t.id = tm.tenant_id WHERE tm.role IN ? AND tm.status = 'active' AND tm.deleted_at IS NULL AND t.deleted_at IS NULL AND t.status = 'active')")
			args = append(args, tenantRoles)
		}
		if len(parts) == 0 {
			return query.Where("1 = 0")
		}
		return query.Where("("+strings.Join(parts, " OR ")+")", args...)
	case "workspaces":
		ids, ok := announcementWorkspaceIDs(audience.IDs)
		if !ok || len(ids) == 0 {
			return query.Where("1 = 0")
		}
		return query.Where("users.id IN (SELECT tm.user_id FROM tenant_members tm JOIN tenants t ON t.id = tm.tenant_id WHERE tm.tenant_id IN ? AND tm.status = 'active' AND tm.deleted_at IS NULL AND t.deleted_at IS NULL AND t.status = 'active')", ids)
	}
	return query.Where("1 = 0")
}

func audienceCounts(tx *gorm.DB, audience announcementAudience) (announcementCounts, error) {
	var total int64
	err := audienceUsers(tx, audience).Count(&total).Error
	return announcementCounts{Users: total}, err
}

func validateAnnouncementAudience(tx *gorm.DB, raw json.RawMessage) (announcementAudience, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 2 || fields["mode"] == nil || fields["ids"] == nil {
		return announcementAudience{}, announcementInvalid("Select a target audience.")
	}
	mode, ok := announcementRawString(fields["mode"])
	var items []json.RawMessage
	if !ok || !announcementAudienceModes[mode] || json.Unmarshal(fields["ids"], &items) != nil || items == nil || len(items) > 200 {
		return announcementAudience{}, announcementInvalid("Target audience is invalid.")
	}
	unique := map[string]bool{}
	for _, item := range items {
		id, ok := announcementRawString(item)
		if !ok || len(id) < 1 || len(id) > 64 {
			return announcementAudience{}, announcementInvalid("Target audience is invalid.")
		}
		unique[id] = true
	}
	ids := make([]string, 0, len(unique))
	for id := range unique {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if (mode == "all") != (len(ids) == 0) {
		return announcementAudience{}, announcementInvalid("Select at least one target for this audience.")
	}
	valid := map[string]bool{}
	switch mode {
	case "roles":
		for _, role := range announcementRoles {
			valid[role] = true
		}
	case "users":
		var found []string
		if err := audienceUsers(tx, announcementAudience{Mode: "all"}).Where("users.id IN ?", ids).Pluck("users.id", &found).Error; err != nil {
			return announcementAudience{}, err
		}
		for _, id := range found {
			valid[id] = true
		}
	case "workspaces":
		if parsed, ok := announcementWorkspaceIDs(ids); ok {
			var found []uint64
			if err := tx.Table("tenants").Where("id IN ? AND deleted_at IS NULL AND status = 'active'", parsed).Pluck("id", &found).Error; err != nil {
				return announcementAudience{}, err
			}
			for _, id := range found {
				valid[strconv.FormatUint(id, 10)] = true
			}
		}
	}
	for _, id := range ids {
		if !valid[id] {
			return announcementAudience{}, announcementInvalid("Some audience targets are no longer available. Update your selection.")
		}
	}
	return announcementAudience{Mode: mode, IDs: ids}, nil
}

type announcementValues struct {
	Title       string
	Body        string
	ShowBanner  bool
	BannerColor string
	TypeID      string
	Audience    announcementAudience
	StartsAt    *int64
	EndsAt      *int64
}

func (v announcementValues) hashable() map[string]interface{} {
	return map[string]interface{}{
		"title": v.Title, "body": v.Body, "show_banner": v.ShowBanner, "banner_color": v.BannerColor,
		"type_id": v.TypeID, "audience": v.Audience, "starts_at": v.StartsAt, "ends_at": v.EndsAt,
	}
}

func (v announcementValues) columns() map[string]interface{} {
	ids, _ := json.Marshal(v.Audience.IDs)
	return map[string]interface{}{
		"title": v.Title, "body": v.Body, "show_banner": v.ShowBanner, "banner_color": v.BannerColor,
		"type_id": v.TypeID, "audience_mode": v.Audience.Mode, "audience_ids": string(ids),
		"starts_at": v.StartsAt, "ends_at": v.EndsAt,
	}
}

func (h *AnnouncementHandler) typeExists(tx *gorm.DB, id string) (bool, error) {
	for _, builtin := range announcementBuiltinTypes {
		if builtin.ID == id {
			return true, nil
		}
	}
	var count int64
	err := tx.Model(&announcementTypeRow{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (h *AnnouncementHandler) parseValues(tx *gorm.DB, fields map[string]json.RawMessage) (announcementValues, error) {
	unknown := []string{}
	for key := range fields {
		if !announcementEditableFields[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		return announcementValues{}, announcementInvalid("Unsupported announcement field.")
	}
	var values announcementValues
	var err error
	if values.Title, err = announcementText(fields["title"], "title", 120); err != nil {
		return values, err
	}
	if values.Body, err = announcementText(fields["body"], "body", 1000); err != nil {
		return values, err
	}
	if raw, ok := fields["showBanner"]; ok {
		value, valid := announcementRawBool(raw)
		if !valid {
			return values, announcementInvalid("showBanner must be a boolean.")
		}
		values.ShowBanner = value
	}
	values.BannerColor = "neutral"
	if raw, ok := fields["bannerColor"]; ok {
		if values.BannerColor, err = announcementText(raw, "bannerColor", 16); err != nil {
			return values, err
		}
	}
	if !announcementBannerColors[values.BannerColor] {
		return values, announcementInvalid("Select a supported banner color.")
	}
	values.TypeID = "maintenance"
	if raw, ok := fields["typeId"]; ok {
		if values.TypeID, err = announcementText(raw, "typeId", 64); err != nil {
			return values, err
		}
	}
	exists, err := h.typeExists(tx, values.TypeID)
	if err != nil {
		return values, err
	}
	if !exists {
		return values, announcementInvalid("Unknown notification type.")
	}
	audience := fields["audience"]
	if audience == nil {
		audience = json.RawMessage(`{"mode":"all","ids":[]}`)
	}
	if values.Audience, err = validateAnnouncementAudience(tx, audience); err != nil {
		return values, err
	}
	if values.StartsAt, err = announcementTimestamp(fields["startsAt"], "startsAt"); err != nil {
		return values, err
	}
	if values.EndsAt, err = announcementTimestamp(fields["endsAt"], "endsAt"); err != nil {
		return values, err
	}
	if values.StartsAt != nil && values.EndsAt != nil && *values.EndsAt <= *values.StartsAt {
		return values, announcementInvalid("endsAt must be later than startsAt.")
	}
	return values, nil
}

func announcementRequestHash(action string, values interface{}) string {
	encoded, _ := json.Marshal([]interface{}{action, values})
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// replay returns the announcement an earlier request with the same key
// produced, or nil when the key is new.
func (h *AnnouncementHandler) replay(tx *gorm.DB, actor string, raw json.RawMessage, action string, values interface{}) (string, string, *announcementRow, error) {
	key, ok := announcementRawString(raw)
	if !ok || !announcementIdempotencyRE.MatchString(key) {
		return "", "", nil, announcementInvalid("idempotencyKey is invalid.")
	}
	digest := announcementRequestHash(action, values)
	var existing announcementIdempotencyRow
	err := tx.Where("actor_id = ? AND idempotency_key = ?", actor, key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return key, digest, nil, nil
	}
	if err != nil {
		return "", "", nil, err
	}
	if existing.Action != action || subtle.ConstantTimeCompare([]byte(existing.RequestHash), []byte(digest)) != 1 {
		return "", "", nil, announcementConflict("The idempotency key was already used for another request.")
	}
	var row announcementRow
	if err := tx.Where("id = ?", existing.ResourceID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, announcementConflict("This announcement has been deleted.")
		}
		return "", "", nil, err
	}
	return key, digest, &row, nil
}

func (h *AnnouncementHandler) types(tx *gorm.DB) ([]announcementTypePayload, error) {
	var rows []announcementTypeRow
	if err := tx.Order("name ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := append([]announcementTypePayload{}, announcementBuiltinTypes...)
	for _, row := range rows {
		result = append(result, announcementTypePayload{ID: row.ID, Name: row.Name, Builtin: false})
	}
	return result, nil
}

type announcementPerson struct {
	ID    string
	Name  string
	Email string
}

// payloads renders rows for the API. Admin payloads carry the audience,
// its size and the viewer count; user payloads carry notification state.
func (h *AnnouncementHandler) payloads(tx *gorm.DB, rows []announcementRow, admin bool) ([]announcementPayload, error) {
	result := make([]announcementPayload, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	typeList, err := h.types(tx)
	if err != nil {
		return nil, err
	}
	typeByID := map[string]announcementTypePayload{}
	for _, item := range typeList {
		typeByID[item.ID] = item
	}
	personIDs, rowIDs := []string{}, []string{}
	for _, row := range rows {
		rowIDs = append(rowIDs, row.ID)
		personIDs = append(personIDs, row.CreatedBy)
		if row.PublishedBy != nil {
			personIDs = append(personIDs, *row.PublishedBy)
		}
	}
	var people []announcementPerson
	if err := tx.Table("users AS u").Select("u.id AS id, "+announcementUserName+" AS name, COALESCE(u.email, '') AS email").Where("u.id IN ?", personIDs).Scan(&people).Error; err != nil {
		return nil, err
	}
	personByID := map[string]announcementPerson{}
	for _, person := range people {
		personByID[person.ID] = person
	}
	viewers := map[string]int64{}
	if admin {
		var counts []struct {
			AnnouncementID string
			Count          int64
		}
		if err := tx.Table("announcement_notification_states AS s").Select("s.announcement_id AS announcement_id, COUNT(s.user_id) AS count").
			Joins("JOIN users AS u ON u.id = s.user_id").
			Where("s.announcement_id IN ? AND s.phase = 'published' AND s.first_seen_at IS NOT NULL", rowIDs).
			Group("s.announcement_id").Scan(&counts).Error; err != nil {
			return nil, err
		}
		for _, count := range counts {
			viewers[count.AnnouncementID] = count.Count
		}
	}
	for _, row := range rows {
		kind, ok := typeByID[row.TypeID]
		if !ok {
			kind = announcementTypePayload{ID: row.TypeID, Name: row.TypeID, Builtin: false}
		}
		item := announcementPayload{
			ID: row.ID, TypeID: row.TypeID, Type: kind, Status: row.Status, Version: row.Version,
			Title: row.Title, Body: row.Body, ShowBanner: row.ShowBanner, BannerColor: row.BannerColor,
			StartsAt: row.StartsAt, EndsAt: row.EndsAt, PublishedAt: row.PublishedAt,
			CreatedAt: row.CreatedAt, CreatedBy: row.CreatedBy, PublishedBy: row.PublishedBy,
		}
		if person, ok := personByID[row.CreatedBy]; ok {
			item.CreatedByName, item.CreatedByEmail = announcementOptional(person.Name), announcementOptional(person.Email)
		}
		if row.PublishedBy != nil {
			if person, ok := personByID[*row.PublishedBy]; ok {
				item.PublishedByName, item.PublishedByEmail = announcementOptional(person.Name), announcementOptional(person.Email)
			}
		}
		if admin {
			audience := row.audience()
			counts, err := audienceCounts(tx, audience)
			if err != nil {
				return nil, err
			}
			viewerCount := viewers[row.ID]
			item.Audience, item.AudienceCounts = &audience, &counts
			item.Delivery, item.MaintenanceDelivery, item.ResolutionDelivery = &announcementDelivery{}, &announcementDelivery{}, &announcementDelivery{}
			item.ViewerCount = &viewerCount
		}
		result = append(result, item)
	}
	return result, nil
}

func announcementOptional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (h *AnnouncementHandler) payload(tx *gorm.DB, row announcementRow) (announcementPayload, error) {
	items, err := h.payloads(tx, []announcementRow{row}, true)
	if err != nil {
		return announcementPayload{}, err
	}
	return items[0], nil
}

// announcementViewer is the requesting user with the memberships that
// decide which audiences include them.
type announcementViewer struct {
	ID          string
	SystemAdmin bool
	Roles       map[string]bool
	Workspaces  map[string]bool
}

func (v announcementViewer) targeted(audience announcementAudience) bool {
	if audience.Mode == "all" {
		return true
	}
	for _, id := range audience.IDs {
		switch audience.Mode {
		case "users":
			if id == v.ID {
				return true
			}
		case "roles":
			if (id == "system_admin" && v.SystemAdmin) || (id != "system_admin" && v.Roles[id]) {
				return true
			}
		case "workspaces":
			if v.Workspaces[id] {
				return true
			}
		}
	}
	return false
}

func (h *AnnouncementHandler) viewer(tx *gorm.DB, c *gin.Context) (announcementViewer, error) {
	id := announcementActor(c)
	var user struct {
		ID            string
		IsSystemAdmin bool
	}
	if id == "" {
		return announcementViewer{}, errAnnouncementNotFound
	}
	if err := tx.Table("users").Select("id, is_system_admin").Where("id = ? AND deleted_at IS NULL AND is_active = ?", id, true).Scan(&user).Error; err != nil {
		return announcementViewer{}, err
	}
	if user.ID == "" {
		return announcementViewer{}, errAnnouncementNotFound
	}
	var memberships []struct {
		TenantID uint64
		Role     string
	}
	if err := tx.Table("tenant_members AS tm").Select("tm.tenant_id AS tenant_id, tm.role AS role").
		Joins("JOIN tenants AS t ON t.id = tm.tenant_id").
		Where("tm.user_id = ? AND tm.status = 'active' AND tm.deleted_at IS NULL AND t.deleted_at IS NULL AND t.status = 'active'", id).
		Scan(&memberships).Error; err != nil {
		return announcementViewer{}, err
	}
	viewer := announcementViewer{ID: id, SystemAdmin: user.IsSystemAdmin, Roles: map[string]bool{}, Workspaces: map[string]bool{}}
	for _, membership := range memberships {
		viewer.Roles[membership.Role] = true
		viewer.Workspaces[strconv.FormatUint(membership.TenantID, 10)] = true
	}
	return viewer, nil
}

func (h *AnnouncementHandler) withStates(tx *gorm.DB, viewer announcementViewer, items []announcementPayload, phases ...string) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var rows []announcementStateRow
	if err := tx.Where("user_id = ? AND announcement_id IN ?", viewer.ID, ids).Find(&rows).Error; err != nil {
		return err
	}
	states := map[string]announcementState{}
	for _, row := range rows {
		states[row.AnnouncementID+":"+row.Phase] = announcementState{Read: row.IsRead, Deleted: row.Deleted, DismissedVersion: row.DismissedVersion}
	}
	for index := range items {
		items[index].NotificationState = map[string]announcementState{}
		for _, phase := range phases {
			items[index].NotificationState[phase] = states[items[index].ID+":"+phase]
		}
	}
	return nil
}

// notifications is the user's feed: the latest 50 announcements that were
// ever published to them, including ones taken down since.
func (h *AnnouncementHandler) notifications(tx *gorm.DB, viewer announcementViewer) ([]announcementPayload, error) {
	var rows []announcementRow
	if err := tx.Where("status IN ?", []string{"maintenance", "resolved", "withdrawn"}).Order("published_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	eligible := []announcementRow{}
	for _, row := range rows {
		if viewer.targeted(row.audience()) {
			eligible = append(eligible, row)
			if len(eligible) == 50 {
				break
			}
		}
	}
	items, err := h.payloads(tx, eligible, false)
	if err != nil {
		return nil, err
	}
	return items, h.withStates(tx, viewer, items, "published", "resolved")
}

// active lists the live announcements inside their schedule window.
func (h *AnnouncementHandler) active(tx *gorm.DB, viewer announcementViewer) ([]announcementPayload, error) {
	var rows []announcementRow
	if err := tx.Where("status = ?", "maintenance").Order("published_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	now := announcementNow()
	eligible := []announcementRow{}
	for _, row := range rows {
		if announcementInWindow(row, now) && viewer.targeted(row.audience()) {
			eligible = append(eligible, row)
		}
	}
	items, err := h.payloads(tx, eligible, false)
	if err != nil {
		return nil, err
	}
	return items, h.withStates(tx, viewer, items, "published")
}

// Current returns the caller's live announcements and notification feed.
func (h *AnnouncementHandler) Current(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	tx := h.db.WithContext(c.Request.Context())
	viewer, err := h.viewer(tx, c)
	if err != nil {
		h.fail(c, err)
		return
	}
	active, err := h.active(tx, viewer)
	if err != nil {
		h.fail(c, err)
		return
	}
	items, err := h.notifications(tx, viewer)
	if err != nil {
		h.fail(c, err)
		return
	}
	var first interface{}
	if len(active) > 0 {
		first = active[0]
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"announcement": first, "announcements": active, "items": items, "pushConfigured": false, "publicKey": nil,
	}})
}

type announcementRef struct {
	ID    string
	Phase string
}

// UpdateNotifications marks the caller's notifications read, unread, seen,
// dismissed from the banner or deleted from their own list.
func (h *AnnouncementHandler) UpdateNotifications(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	if len(fields) != 2 || fields["action"] == nil || fields["items"] == nil {
		h.fail(c, announcementInvalid("action and items are required."))
		return
	}
	action, ok := announcementRawString(fields["action"])
	if !ok || (action != "read" && action != "unread" && action != "delete" && action != "seen" && action != "dismiss") {
		h.fail(c, announcementInvalid("Invalid notification action."))
		return
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(fields["items"], &items) != nil || len(items) < 1 || len(items) > 100 {
		h.fail(c, announcementInvalid("Select between 1 and 100 notifications."))
		return
	}
	unique := map[announcementRef]bool{}
	for _, item := range items {
		id, idOK := announcementRawString(item["announcementId"])
		phase, phaseOK := announcementRawString(item["phase"])
		if len(item) != 2 || !idOK || !phaseOK || len(id) < 1 || len(id) > 64 || (phase != "published" && phase != "resolved") {
			h.fail(c, announcementInvalid("Invalid notification reference."))
			return
		}
		unique[announcementRef{ID: id, Phase: phase}] = true
	}
	refs := make([]announcementRef, 0, len(unique))
	ids := []string{}
	for ref := range unique {
		refs = append(refs, ref)
		ids = append(ids, ref.ID)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].ID != refs[j].ID {
			return refs[i].ID < refs[j].ID
		}
		return refs[i].Phase < refs[j].Phase
	})
	var result []announcementPayload
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		viewer, err := h.viewer(tx, c)
		if err != nil {
			return err
		}
		var rows []announcementRow
		if err := tx.Where("id IN ?", ids).Find(&rows).Error; err != nil {
			return err
		}
		byID := map[string]announcementRow{}
		for _, row := range rows {
			byID[row.ID] = row
		}
		for _, ref := range refs {
			row, ok := byID[ref.ID]
			if !ok || (row.Status != "maintenance" && row.Status != "resolved" && row.Status != "withdrawn") ||
				(ref.Phase == "resolved" && row.Status != "resolved") || !viewer.targeted(row.audience()) {
				return errAnnouncementNotFound
			}
			if action == "dismiss" && (ref.Phase != "published" || row.Status != "maintenance") {
				return errAnnouncementNotFound
			}
		}
		now := announcementNow()
		for _, ref := range refs {
			state := announcementStateRow{ID: uuid.NewString(), AnnouncementID: ref.ID, UserID: viewer.ID, Phase: ref.Phase}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "announcement_id"}, {Name: "user_id"}, {Name: "phase"}}, DoNothing: true}).Create(&state).Error; err != nil {
				return err
			}
			scope := tx.Model(&announcementStateRow{}).Where("announcement_id = ? AND user_id = ? AND phase = ?", ref.ID, viewer.ID, ref.Phase)
			switch action {
			case "seen":
				err = scope.Where("first_seen_at IS NULL").Update("first_seen_at", now).Error
			case "dismiss":
				err = scope.Update("dismissed_version", byID[ref.ID].Version).Error
			case "delete":
				err = scope.Where("deleted = ?", false).Update("deleted", true).Error
			default:
				err = scope.Where("deleted = ?", false).Update("is_read", action == "read").Error
			}
			if err != nil {
				return err
			}
		}
		result, err = h.notifications(tx, viewer)
		return err
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": result}})
}

// List returns live announcements first, then the 50 most recent ones.
func (h *AnnouncementHandler) List(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	tx := h.db.WithContext(c.Request.Context())
	var live, recent []announcementRow
	if err := tx.Where("status = ?", "maintenance").Order("published_at DESC, id DESC").Find(&live).Error; err != nil {
		h.fail(c, err)
		return
	}
	if err := tx.Order("created_at DESC, id DESC").Limit(50).Find(&recent).Error; err != nil {
		h.fail(c, err)
		return
	}
	now := announcementNow()
	seen := map[string]bool{}
	rows := []announcementRow{}
	for _, row := range live {
		if announcementInWindow(row, now) {
			seen[row.ID] = true
			rows = append(rows, row)
		}
	}
	limit := len(rows)
	if limit < 50 {
		limit = 50
	}
	for _, row := range recent {
		if !seen[row.ID] && len(rows) < limit {
			rows = append(rows, row)
		}
	}
	items, err := h.payloads(tx, rows, true)
	if err != nil {
		h.fail(c, err)
		return
	}
	counts, err := audienceCounts(tx, announcementAudience{Mode: "all", IDs: []string{}})
	if err != nil {
		h.fail(c, err)
		return
	}
	typeList, err := h.types(tx)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items, "audience": counts, "pushConfigured": false, "types": typeList}})
}

// CreateType adds a custom notification type next to the built-in ones.
func (h *AnnouncementHandler) CreateType(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	if len(fields) != 1 || fields["name"] == nil {
		h.fail(c, announcementInvalid("A notification type name is required."))
		return
	}
	name, err := announcementText(fields["name"], "name", 60)
	if err != nil {
		h.fail(c, err)
		return
	}
	name = strings.Join(strings.Fields(name), " ")
	normalized := strings.ToLower(name)
	duplicate := announcementInvalid("This notification type already exists.")
	for _, builtin := range announcementBuiltinTypes {
		if strings.ToLower(builtin.Name) == normalized {
			h.fail(c, duplicate)
			return
		}
	}
	for _, reserved := range announcementReservedTypeNames {
		if reserved == normalized {
			h.fail(c, duplicate)
			return
		}
	}
	tx := h.db.WithContext(c.Request.Context())
	var count int64
	if err := tx.Model(&announcementTypeRow{}).Where("normalized_name = ?", normalized).Count(&count).Error; err != nil {
		h.fail(c, err)
		return
	}
	if count > 0 {
		h.fail(c, duplicate)
		return
	}
	row := announcementTypeRow{ID: uuid.NewString(), Name: name, NormalizedName: normalized, CreatedBy: announcementActor(c), CreatedAt: announcementNow()}
	if err := tx.Create(&row).Error; err != nil {
		if tx.Model(&announcementTypeRow{}).Where("normalized_name = ?", normalized).Count(&count).Error == nil && count > 0 {
			h.fail(c, duplicate)
			return
		}
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": announcementTypePayload{ID: row.ID, Name: row.Name, Builtin: false}})
}

// Options lists the users, workspaces or roles an audience can target.
func (h *AnnouncementHandler) Options(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	mode, query := c.Query("mode"), c.Query("q")
	ids, hasIDs := c.GetQueryArray("id")
	if (mode != "users" && mode != "workspaces" && mode != "roles") || utf8.RuneCountInString(query) > 100 {
		h.fail(c, announcementInvalid("Invalid audience search."))
		return
	}
	if hasIDs {
		if len(ids) > 200 {
			h.fail(c, announcementInvalid("Invalid audience targets."))
			return
		}
		for _, id := range ids {
			if len(id) > 64 {
				h.fail(c, announcementInvalid("Invalid audience targets."))
				return
			}
		}
	}
	type option struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Detail string `json:"detail,omitempty"`
	}
	result := []option{}
	limit := 50
	if hasIDs {
		limit = 200
	}
	tx := h.db.WithContext(c.Request.Context())
	pattern := "%" + announcementLikeEscaper.Replace(strings.ToLower(query)) + "%"
	switch mode {
	case "roles":
		for _, role := range announcementRoles {
			result = append(result, option{ID: role, Label: role})
		}
	case "users":
		var rows []announcementPerson
		scope := tx.Table("users AS u").Where("u.deleted_at IS NULL AND u.is_active = ?", true)
		if hasIDs {
			scope = scope.Where("u.id IN ?", ids)
		} else if query != "" {
			scope = scope.Where(`(LOWER(`+announcementUserName+`) LIKE ? ESCAPE '\' OR LOWER(COALESCE(u.email, '')) LIKE ? ESCAPE '\')`, pattern, pattern)
		}
		if err := scope.Select("u.id AS id, " + announcementUserName + " AS name, COALESCE(u.email, '') AS email").
			Order("name ASC, u.id ASC").Limit(limit).Scan(&rows).Error; err != nil {
			h.fail(c, err)
			return
		}
		for _, row := range rows {
			label := row.Name
			if label == "" {
				label = row.Email
			}
			result = append(result, option{ID: row.ID, Label: label, Detail: row.Email})
		}
	case "workspaces":
		var rows []struct {
			ID   uint64
			Name string
		}
		scope := tx.Table("tenants").Where("deleted_at IS NULL AND status = 'active'")
		if hasIDs {
			parsed := []uint64{}
			for _, id := range ids {
				if value, err := strconv.ParseUint(id, 10, 64); err == nil {
					parsed = append(parsed, value)
				}
			}
			if len(parsed) == 0 {
				c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
				return
			}
			scope = scope.Where("id IN ?", parsed)
		} else if query != "" {
			scope = scope.Where(`LOWER(name) LIKE ? ESCAPE '\'`, pattern)
		}
		if err := scope.Select("id, name").Order("name ASC, id ASC").Limit(limit).Scan(&rows).Error; err != nil {
			h.fail(c, err)
			return
		}
		for _, row := range rows {
			result = append(result, option{ID: strconv.FormatUint(row.ID, 10), Label: row.Name})
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// PreviewAudience counts the users an audience selection reaches.
func (h *AnnouncementHandler) PreviewAudience(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		h.fail(c, announcementInvalid("Request body must be a JSON object."))
		return
	}
	tx := h.db.WithContext(c.Request.Context())
	audience, err := validateAnnouncementAudience(tx, raw)
	if err != nil {
		h.fail(c, err)
		return
	}
	counts, err := audienceCounts(tx, audience)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": counts})
}

// Create stores a new draft. The idempotency key makes retries safe.
func (h *AnnouncementHandler) Create(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	key := fields["idempotencyKey"]
	delete(fields, "idempotencyKey")
	actor := announcementActor(c)
	var row announcementRow
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		values, err := h.parseValues(tx, fields)
		if err != nil {
			return err
		}
		idempotencyKey, digest, existing, err := h.replay(tx, actor, key, "create", values.hashable())
		if err != nil {
			return err
		}
		if existing != nil {
			row = *existing
			return nil
		}
		now := announcementNow()
		ids, _ := json.Marshal(values.Audience.IDs)
		row = announcementRow{
			ID: uuid.NewString(), Status: "draft", Version: 1, TypeID: values.TypeID, Title: values.Title, Body: values.Body,
			StartsAt: values.StartsAt, EndsAt: values.EndsAt, ShowBanner: values.ShowBanner, BannerColor: values.BannerColor,
			AudienceMode: values.Audience.Mode, AudienceIDs: string(ids), CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&announcementIdempotencyRow{
			ID: uuid.NewString(), ActorID: actor, IdempotencyKey: idempotencyKey, Action: "create",
			RequestHash: digest, ResourceID: row.ID, CreatedAt: now,
		}).Error
	})
	h.respondRow(c, http.StatusCreated, row, err)
}

func (h *AnnouncementHandler) respondRow(c *gin.Context, status int, row announcementRow, err error) {
	if err != nil {
		h.fail(c, err)
		return
	}
	item, err := h.payload(h.db.WithContext(c.Request.Context()), row)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(status, gin.H{"success": true, "data": item})
}

func (h *AnnouncementHandler) load(tx *gorm.DB, id string) (*announcementRow, error) {
	var row announcementRow
	err := tx.Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (h *AnnouncementHandler) reload(tx *gorm.DB, id string) (announcementRow, error) {
	row, err := h.load(tx, id)
	if err != nil {
		return announcementRow{}, err
	}
	if row == nil {
		return announcementRow{}, errAnnouncementNotFound
	}
	return *row, nil
}

// Update edits a draft when the caller holds its latest version.
func (h *AnnouncementHandler) Update(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	expected, err := announcementVersion(fields)
	if err != nil {
		h.fail(c, err)
		return
	}
	delete(fields, "version")
	id := c.Param("id")
	var row announcementRow
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		values, err := h.parseValues(tx, fields)
		if err != nil {
			return err
		}
		current, err := h.load(tx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return errAnnouncementNotFound
		}
		if current.Status != "draft" || current.Version != expected {
			return announcementConflict("Announcement version or state changed.")
		}
		columns := values.columns()
		columns["version"] = expected + 1
		columns["updated_at"] = announcementNow()
		result := tx.Model(&announcementRow{}).Where("id = ? AND status = ? AND version = ?", id, "draft", expected).Updates(columns)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return announcementConflict("Announcement version or state changed.")
		}
		row, err = h.reload(tx, id)
		return err
	})
	h.respondRow(c, http.StatusOK, row, err)
}

// Delete removes an announcement and every user's state for it.
func (h *AnnouncementHandler) Delete(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	expected, err := announcementVersion(fields)
	if err != nil {
		h.fail(c, err)
		return
	}
	id := c.Param("id")
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		current, err := h.load(tx, id)
		if err != nil || current == nil {
			return err
		}
		if current.Version != expected {
			return announcementConflict("Announcement version or state changed.")
		}
		result := tx.Where("id = ? AND version = ?", id, expected).Delete(&announcementRow{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return announcementConflict("Announcement version or state changed.")
		}
		return tx.Where("announcement_id = ?", id).Delete(&announcementStateRow{}).Error
	})
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": true}})
}

// Unpublish takes a live announcement down; it stays in the history.
func (h *AnnouncementHandler) Unpublish(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	expected, err := announcementVersion(fields)
	if err != nil {
		h.fail(c, err)
		return
	}
	id := c.Param("id")
	var row announcementRow
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		current, err := h.load(tx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return errAnnouncementNotFound
		}
		if current.Status == "withdrawn" && current.Version == expected+1 {
			row = *current
			return nil
		}
		if current.Status != "maintenance" || current.Version != expected {
			return announcementConflict("Announcement version or state changed.")
		}
		result := tx.Model(&announcementRow{}).Where("id = ? AND status = ? AND version = ?", id, "maintenance", expected).
			Updates(map[string]interface{}{"status": "withdrawn", "version": expected + 1, "updated_at": announcementNow()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return announcementConflict("Announcement version or state changed.")
		}
		row, err = h.reload(tx, id)
		return err
	})
	h.respondRow(c, http.StatusOK, row, err)
}

// Publish makes a draft live. With keepDraft the draft stays editable and
// a published copy is created instead.
func (h *AnnouncementHandler) Publish(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	fields, err := announcementBody(c)
	if err != nil {
		h.fail(c, err)
		return
	}
	expected, err := announcementVersion(fields)
	if err != nil {
		h.fail(c, err)
		return
	}
	keepDraft := false
	if raw, ok := fields["keepDraft"]; ok {
		value, valid := announcementRawBool(raw)
		if !valid {
			h.fail(c, announcementInvalid("keepDraft must be a boolean."))
			return
		}
		keepDraft = value
	}
	id, actor := c.Param("id"), announcementActor(c)
	var row announcementRow
	err = h.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		idempotencyKey, digest, existing, err := h.replay(tx, actor, fields["idempotencyKey"], "publish",
			map[string]interface{}{"announcement_id": id, "version": expected, "keep_draft": keepDraft})
		if err != nil {
			return err
		}
		if existing != nil {
			row = *existing
			return nil
		}
		current, err := h.load(tx, id)
		if err != nil {
			return err
		}
		if current == nil {
			return errAnnouncementNotFound
		}
		if current.Status != "draft" || current.Version != expected {
			return announcementConflict("Announcement version or state changed.")
		}
		audience, _ := json.Marshal(current.audience())
		if _, err := validateAnnouncementAudience(tx, audience); err != nil {
			return err
		}
		now := announcementNow()
		if keepDraft {
			row = *current
			row.ID, row.Status, row.Version = uuid.NewString(), "maintenance", expected+1
			row.PublishedAt, row.PublishedBy = &now, &actor
			row.CreatedAt, row.UpdatedAt = now, now
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		} else {
			result := tx.Model(&announcementRow{}).Where("id = ? AND status = ? AND version = ?", id, "draft", expected).
				Updates(map[string]interface{}{"status": "maintenance", "version": expected + 1, "published_at": now, "published_by": actor, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return announcementConflict("Announcement version or state changed.")
			}
			if row, err = h.reload(tx, id); err != nil {
				return err
			}
		}
		return tx.Create(&announcementIdempotencyRow{
			ID: uuid.NewString(), ActorID: actor, IdempotencyKey: idempotencyKey, Action: "publish",
			RequestHash: digest, ResourceID: row.ID, CreatedAt: now,
		}).Error
	})
	h.respondRow(c, http.StatusOK, row, err)
}

// Viewers lists who saw an announcement title for at least a second.
func (h *AnnouncementHandler) Viewers(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	phase, page := c.DefaultQuery("phase", "published"), c.DefaultQuery("page", "1")
	if (phase != "published" && phase != "resolved") || !announcementPageRE.MatchString(page) {
		h.fail(c, announcementInvalid("Invalid viewer phase or page."))
		return
	}
	tx := h.db.WithContext(c.Request.Context())
	current, err := h.load(tx, c.Param("id"))
	if err != nil {
		h.fail(c, err)
		return
	}
	if current == nil {
		h.fail(c, errAnnouncementNotFound)
		return
	}
	scope := func() *gorm.DB {
		return tx.Table("announcement_notification_states AS s").Joins("JOIN users AS u ON u.id = s.user_id").
			Where("s.announcement_id = ? AND s.phase = ? AND s.first_seen_at IS NOT NULL", current.ID, phase)
	}
	var total int64
	if err := scope().Count(&total).Error; err != nil {
		h.fail(c, err)
		return
	}
	number, _ := strconv.Atoi(page)
	rows := []struct {
		UserID string `json:"userId"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		SeenAt int64  `json:"seenAt"`
	}{}
	if err := scope().Select("s.user_id AS user_id, COALESCE(NULLIF(" + announcementUserName + ", ''), u.email) AS name, COALESCE(u.email, '') AS email, s.first_seen_at AS seen_at").
		Order("s.first_seen_at DESC, s.user_id ASC").Limit(announcementViewerPage).Offset((number - 1) * announcementViewerPage).
		Scan(&rows).Error; err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"total": total, "items": rows}})
}

// Test would queue a browser push; this server has no push delivery.
func (h *AnnouncementHandler) Test(c *gin.Context) {
	c.JSON(http.StatusConflict, gin.H{"error": "Push notifications are not configured on this server."})
}
