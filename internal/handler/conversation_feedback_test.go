package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/acrbaran/rag/internal/types"
	"github.com/acrbaran/rag/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type feedbackSessionService struct{ interfaces.SessionService }

func (feedbackSessionService) GetSession(_ context.Context, id string) (*types.Session, error) {
	return &types.Session{ID: id, UserID: "u1"}, nil
}

func TestConversationFeedbackSaveAndTenantList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:feedback-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE TABLE sessions (id TEXT PRIMARY KEY, title TEXT, tenant_id INTEGER, user_id TEXT)",
		"CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT, first_name TEXT, last_name TEXT, email TEXT)",
		"CREATE TABLE tenants (id INTEGER PRIMARY KEY, name TEXT)",
		"CREATE TABLE messages (id TEXT PRIMARY KEY, session_id TEXT, role TEXT, content TEXT, created_at DATETIME, deleted_at DATETIME)",
		"INSERT INTO sessions VALUES ('s1', 'Test sohbeti', 7, 'u1')",
		"INSERT INTO users VALUES ('u1', 'baran', 'Baran', 'Acar', 'baran@example.com')",
		"INSERT INTO tenants VALUES (7, 'Genel')",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&ConversationFeedback{}); err != nil {
		t.Fatal(err)
	}
	h := NewConversationFeedbackHandler(db, feedbackSessionService{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(7))
		ctx = context.WithValue(ctx, types.UserIDContextKey, "u1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.PUT("/sessions/:id/feedback", h.SaveOwn)
	r.GET("/feedback", h.ListTenant)
	r.GET("/feedback/:id", h.DetailTenant)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w
	}
	if got := request(http.MethodPut, "/sessions/s1/feedback", `{"helpful":true}`); got.Code != http.StatusOK {
		t.Fatalf("rating: %d %s", got.Code, got.Body.String())
	}
	if got := request(http.MethodPut, "/sessions/s1/feedback", `{"category":"suggestion","comment":"İyi çalışıyor"}`); got.Code != http.StatusOK {
		t.Fatalf("comment: %d %s", got.Code, got.Body.String())
	}
	if got := request(http.MethodPut, "/sessions/s1/feedback", `{"category":"complaint","comment":""}`); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid comment: %d", got.Code)
	}
	got := request(http.MethodGet, "/feedback", "")
	if got.Code != http.StatusOK {
		t.Fatalf("list: %d %s", got.Code, got.Body.String())
	}
	var response struct {
		Data struct {
			Items []struct {
				Helpful      *bool   `json:"helpful"`
				Category     *string `json:"category"`
				Comment      *string `json:"comment"`
				SessionTitle string  `json:"session_title"`
			} `json:"items"`
			Total int `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(got.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.Total != 1 || len(response.Data.Items) != 1 || response.Data.Items[0].Helpful == nil || !*response.Data.Items[0].Helpful || response.Data.Items[0].Category == nil || *response.Data.Items[0].Category != "suggestion" || response.Data.Items[0].SessionTitle != "Test sohbeti" {
		t.Fatalf("unexpected result: %s", got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?category=complaint", ""); !strings.Contains(got.Body.String(), `"total":0`) {
		t.Fatalf("category filter: %s", got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?q=%C3%A7al%C4%B1%C5%9F", ""); !strings.Contains(got.Body.String(), `"total":1`) {
		t.Fatalf("comment search: %s", got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?q=BARAN@", ""); !strings.Contains(got.Body.String(), `"total":1`) {
		t.Fatalf("email search: %s", got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?q=100%25", ""); !strings.Contains(got.Body.String(), `"total":0`) {
		t.Fatalf("wildcard search: %s", got.Body.String())
	}
	var saved ConversationFeedback
	if err := db.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if got := request(http.MethodGet, "/feedback/"+saved.ID, ""); got.Code != http.StatusOK || !strings.Contains(got.Body.String(), "Test sohbeti") || !strings.Contains(got.Body.String(), `"messages":[]`) {
		t.Fatalf("detail: %d %s", got.Code, got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?from=bad-date", ""); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid date filter: %d %s", got.Code, got.Body.String())
	}
	if got := request(http.MethodGet, "/feedback?from=2000-01-01T00:00:00Z&to=2000-01-02", ""); !strings.Contains(got.Body.String(), `"total":0`) {
		t.Fatalf("date filter: %s", got.Body.String())
	}
}
