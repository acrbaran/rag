package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/acrbaran/rag/internal/types"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type announcementTestResponse struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

func newAnnouncementTestRouter(t *testing.T) (*gin.Engine, func(user, method, path, body string) (int, announcementTestResponse)) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if raw, err := db.DB(); err == nil {
			_ = raw.Close()
		}
	})
	migration, err := os.ReadFile("../../migrations/sqlite/000037_announcements.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		"CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, first_name TEXT, last_name TEXT, email TEXT, is_active BOOLEAN, is_system_admin BOOLEAN, deleted_at DATETIME)",
		"CREATE TABLE tenants (id INTEGER PRIMARY KEY, name TEXT, status TEXT, deleted_at DATETIME)",
		"CREATE TABLE tenant_members (id INTEGER PRIMARY KEY, user_id TEXT, tenant_id INTEGER, role TEXT, status TEXT, deleted_at DATETIME)",
		"INSERT INTO users VALUES ('admin', 'admin', 'Sistem', 'Yöneticisi', 'admin@example.com', 1, 1, NULL)",
		"INSERT INTO users VALUES ('ayse', 'ayse', 'Ayşe', 'Yılmaz', 'ayse@example.com', 1, 0, NULL)",
		"INSERT INTO users VALUES ('mehmet', 'mehmet', '', '', 'mehmet@example.com', 1, 0, NULL)",
		"INSERT INTO tenants VALUES (7, 'Genel', 'active', NULL)",
		"INSERT INTO tenant_members VALUES (1, 'ayse', 7, 'viewer', 'active', NULL)",
	}
	for _, statement := range strings.Split(string(migration), ";") {
		if strings.TrimSpace(statement) != "" {
			statements = append(statements, statement)
		}
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("%s: %v", statement, err)
		}
	}
	h := NewAnnouncementHandler(db)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.UserIDContextKey, c.GetHeader("X-Test-User"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/announcements/current", h.Current)
	r.PATCH("/announcements/notifications", h.UpdateNotifications)
	r.GET("/admin", h.List)
	r.POST("/admin", h.Create)
	r.POST("/admin/types", h.CreateType)
	r.GET("/admin/options", h.Options)
	r.POST("/admin/audience", h.PreviewAudience)
	r.PATCH("/admin/:id", h.Update)
	r.DELETE("/admin/:id", h.Delete)
	r.POST("/admin/:id/publish", h.Publish)
	r.POST("/admin/:id/unpublish", h.Unpublish)
	r.GET("/admin/:id/viewers", h.Viewers)
	request := func(user, method, path, body string) (int, announcementTestResponse) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", user)
		r.ServeHTTP(w, req)
		var response announcementTestResponse
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		return w.Code, response
	}
	return r, request
}

func decodeAnnouncement(t *testing.T, raw json.RawMessage, target interface{}) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func TestAnnouncementLifecycle(t *testing.T) {
	_, request := newAnnouncementTestRouter(t)

	code, response := request("admin", http.MethodPost, "/admin", `{"title":"Bakım","body":"Gece **bakım** var.","typeId":"maintenance","showBanner":true,"bannerColor":"amber","audience":{"mode":"workspaces","ids":["7"]},"startsAt":null,"endsAt":null,"idempotencyKey":"create-1"}`)
	if code != http.StatusCreated {
		t.Fatalf("create: %d %s", code, response.Error)
	}
	var draft announcementPayload
	decodeAnnouncement(t, response.Data, &draft)
	if draft.Status != "draft" || draft.Version != 1 || draft.AudienceCounts.Users != 1 || *draft.CreatedByName != "Sistem Yöneticisi" {
		t.Fatalf("unexpected draft: %+v", draft)
	}

	code, response = request("admin", http.MethodPost, "/admin", `{"title":"Bakım","body":"Gece **bakım** var.","typeId":"maintenance","showBanner":true,"bannerColor":"amber","audience":{"mode":"workspaces","ids":["7"]},"startsAt":null,"endsAt":null,"idempotencyKey":"create-1"}`)
	var replayed announcementPayload
	decodeAnnouncement(t, response.Data, &replayed)
	if code != http.StatusCreated || replayed.ID != draft.ID {
		t.Fatalf("idempotent replay: %d %s", code, response.Error)
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"Başka","body":"x","idempotencyKey":"create-1"}`); code != http.StatusConflict {
		t.Fatalf("reused key: %d %s", code, response.Error)
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"x","body":"y","extra":1,"idempotencyKey":"k"}`); code != http.StatusBadRequest || response.Error != "Unsupported announcement field." {
		t.Fatalf("unknown field: %d %s", code, response.Error)
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"x","body":"y","startsAt":10,"endsAt":5,"idempotencyKey":"k"}`); code != http.StatusBadRequest || response.Error != "endsAt must be later than startsAt." {
		t.Fatalf("window: %d %s", code, response.Error)
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"x","body":"y","audience":{"mode":"users","ids":["ghost"]},"idempotencyKey":"k"}`); code != http.StatusBadRequest || !strings.Contains(response.Error, "no longer available") {
		t.Fatalf("stale audience: %d %s", code, response.Error)
	}

	if code, response = request("admin", http.MethodPatch, "/admin/"+draft.ID, `{"version":5,"title":"x","body":"y"}`); code != http.StatusConflict {
		t.Fatalf("stale update: %d %s", code, response.Error)
	}
	code, response = request("admin", http.MethodPatch, "/admin/"+draft.ID, `{"version":1,"title":"Planlı bakım","body":"Gece **bakım** var.","showBanner":true,"bannerColor":"amber","audience":{"mode":"workspaces","ids":["7"]}}`)
	if code != http.StatusOK {
		t.Fatalf("update: %d %s", code, response.Error)
	}
	decodeAnnouncement(t, response.Data, &draft)
	if draft.Version != 2 || draft.Title != "Planlı bakım" {
		t.Fatalf("unexpected update: %+v", draft)
	}

	code, response = request("admin", http.MethodPost, "/admin/"+draft.ID+"/publish", `{"version":2,"idempotencyKey":"publish-1","keepDraft":true}`)
	if code != http.StatusOK {
		t.Fatalf("publish: %d %s", code, response.Error)
	}
	var live announcementPayload
	decodeAnnouncement(t, response.Data, &live)
	if live.Status != "maintenance" || live.ID == draft.ID || live.Version != 3 || live.PublishedByName == nil {
		t.Fatalf("unexpected published copy: %+v", live)
	}

	// Mehmet is outside the workspace audience and sees nothing.
	code, response = request("mehmet", http.MethodGet, "/announcements/current", "")
	var outsider struct {
		Announcement  *announcementPayload  `json:"announcement"`
		Announcements []announcementPayload `json:"announcements"`
		Items         []announcementPayload `json:"items"`
	}
	decodeAnnouncement(t, response.Data, &outsider)
	if code != http.StatusOK || outsider.Announcement != nil || len(outsider.Items) != 0 {
		t.Fatalf("outsider feed: %d %s", code, response.Data)
	}

	code, response = request("ayse", http.MethodGet, "/announcements/current", "")
	var feed struct {
		Announcement *announcementPayload  `json:"announcement"`
		Items        []announcementPayload `json:"items"`
	}
	decodeAnnouncement(t, response.Data, &feed)
	if code != http.StatusOK || feed.Announcement == nil || feed.Announcement.ID != live.ID || len(feed.Items) != 1 || feed.Items[0].Audience != nil {
		t.Fatalf("member feed: %d %s", code, response.Data)
	}

	ref := `[{"announcementId":"` + live.ID + `","phase":"published"}]`
	for _, action := range []string{"seen", "read", "dismiss"} {
		if code, response = request("ayse", http.MethodPatch, "/announcements/notifications", `{"action":"`+action+`","items":`+ref+`}`); code != http.StatusOK {
			t.Fatalf("%s: %d %s", action, code, response.Error)
		}
	}
	var updated struct {
		Items []announcementPayload `json:"items"`
	}
	decodeAnnouncement(t, response.Data, &updated)
	state := updated.Items[0].NotificationState["published"]
	if !state.Read || state.Deleted || state.DismissedVersion == nil || *state.DismissedVersion != 3 {
		t.Fatalf("unexpected state: %+v", state)
	}
	if code, _ = request("mehmet", http.MethodPatch, "/announcements/notifications", `{"action":"read","items":`+ref+`}`); code != http.StatusNotFound {
		t.Fatalf("outsider action: %d", code)
	}
	if code, response = request("ayse", http.MethodPatch, "/announcements/notifications", `{"action":"burn","items":`+ref+`}`); code != http.StatusBadRequest || response.Error != "Invalid notification action." {
		t.Fatalf("invalid action: %d %s", code, response.Error)
	}

	code, response = request("admin", http.MethodGet, "/admin/"+live.ID+"/viewers?phase=published&page=1", "")
	var viewers struct {
		Total int64 `json:"total"`
		Items []struct {
			UserID string `json:"userId"`
			Name   string `json:"name"`
		} `json:"items"`
	}
	decodeAnnouncement(t, response.Data, &viewers)
	if code != http.StatusOK || viewers.Total != 1 || viewers.Items[0].Name != "Ayşe Yılmaz" {
		t.Fatalf("viewers: %d %s", code, response.Data)
	}
	if code, _ = request("admin", http.MethodGet, "/admin/"+live.ID+"/viewers?page=0", ""); code != http.StatusBadRequest {
		t.Fatalf("invalid page: %d", code)
	}

	code, response = request("admin", http.MethodGet, "/admin", "")
	var list struct {
		Items    []announcementPayload `json:"items"`
		Audience announcementCounts    `json:"audience"`
		Types    []announcementTypePayload
	}
	decodeAnnouncement(t, response.Data, &list)
	if code != http.StatusOK || len(list.Items) != 2 || list.Items[0].ID != live.ID || *list.Items[0].ViewerCount != 1 || list.Audience.Users != 3 {
		t.Fatalf("admin list: %d %s", code, response.Data)
	}

	code, response = request("admin", http.MethodPost, "/admin/"+live.ID+"/unpublish", `{"version":3}`)
	decodeAnnouncement(t, response.Data, &live)
	if code != http.StatusOK || live.Status != "withdrawn" || live.Version != 4 {
		t.Fatalf("unpublish: %d %s", code, response.Error)
	}
	if code, _ = request("admin", http.MethodPost, "/admin/"+live.ID+"/unpublish", `{"version":3}`); code != http.StatusOK {
		t.Fatalf("repeated unpublish: %d", code)
	}
	code, response = request("ayse", http.MethodGet, "/announcements/current", "")
	decodeAnnouncement(t, response.Data, &feed)
	if feed.Announcement != nil || len(feed.Items) != 1 || feed.Items[0].Status != "withdrawn" {
		t.Fatalf("withdrawn feed: %s", response.Data)
	}

	if code, response = request("admin", http.MethodDelete, "/admin/"+live.ID, `{"version":4}`); code != http.StatusOK {
		t.Fatalf("delete: %d %s", code, response.Error)
	}
	code, response = request("ayse", http.MethodGet, "/announcements/current", "")
	decodeAnnouncement(t, response.Data, &feed)
	if len(feed.Items) != 0 {
		t.Fatalf("deleted announcement still listed: %s", response.Data)
	}
}

func TestAnnouncementTypesAndOptions(t *testing.T) {
	_, request := newAnnouncementTestRouter(t)
	code, response := request("admin", http.MethodPost, "/admin/types", `{"name":"  Güvenlik   duyurusu "}`)
	var created announcementTypePayload
	decodeAnnouncement(t, response.Data, &created)
	if code != http.StatusCreated || created.Name != "Güvenlik duyurusu" || created.Builtin {
		t.Fatalf("create type: %d %s", code, response.Error)
	}
	for _, name := range []string{"güvenlik duyurusu", "Bakım", "Update"} {
		if code, response = request("admin", http.MethodPost, "/admin/types", `{"name":"`+name+`"}`); code != http.StatusBadRequest || response.Error != "This notification type already exists." {
			t.Fatalf("duplicate %q: %d %s", name, code, response.Error)
		}
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"x","body":"y","typeId":"`+created.ID+`","idempotencyKey":"typed"}`); code != http.StatusCreated {
		t.Fatalf("custom typed draft: %d %s", code, response.Error)
	}
	if code, response = request("admin", http.MethodPost, "/admin", `{"title":"x","body":"y","typeId":"nope","idempotencyKey":"typed-2"}`); code != http.StatusBadRequest || response.Error != "Unknown notification type." {
		t.Fatalf("unknown type: %d %s", code, response.Error)
	}

	code, response = request("admin", http.MethodGet, "/admin/options?mode=users&q=ay%C5%9Fe", "")
	var options []struct {
		ID     string `json:"id"`
		Label  string `json:"label"`
		Detail string `json:"detail"`
	}
	decodeAnnouncement(t, response.Data, &options)
	if code != http.StatusOK || len(options) != 1 || options[0].ID != "ayse" || options[0].Detail != "ayse@example.com" {
		t.Fatalf("user options: %d %s", code, response.Data)
	}
	code, response = request("admin", http.MethodGet, "/admin/options?mode=users&id=mehmet", "")
	decodeAnnouncement(t, response.Data, &options)
	if code != http.StatusOK || len(options) != 1 || options[0].Label != "mehmet" {
		t.Fatalf("user options by id: %d %s", code, response.Data)
	}
	code, response = request("admin", http.MethodGet, "/admin/options?mode=workspaces&q=gen", "")
	decodeAnnouncement(t, response.Data, &options)
	if code != http.StatusOK || len(options) != 1 || options[0].ID != "7" || options[0].Label != "Genel" {
		t.Fatalf("workspace options: %d %s", code, response.Data)
	}
	if code, _ = request("admin", http.MethodGet, "/admin/options?mode=teams", ""); code != http.StatusBadRequest {
		t.Fatalf("invalid mode: %d", code)
	}

	code, response = request("admin", http.MethodPost, "/admin/audience", `{"mode":"roles","ids":["system_admin","viewer"]}`)
	var counts announcementCounts
	decodeAnnouncement(t, response.Data, &counts)
	if code != http.StatusOK || counts.Users != 2 {
		t.Fatalf("role audience: %d %s", code, response.Data)
	}
	if code, response = request("admin", http.MethodPost, "/admin/audience", `{"mode":"roles","ids":[]}`); code != http.StatusBadRequest || response.Error != "Select at least one target for this audience." {
		t.Fatalf("empty audience: %d %s", code, response.Error)
	}
}
