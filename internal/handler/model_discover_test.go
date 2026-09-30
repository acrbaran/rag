package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	secutils "github.com/acrbaran/rag/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDiscoverModels(t *testing.T) {
	secutils.SetSSRFWhitelistFromRaw("127.0.0.1")
	t.Cleanup(func() { secutils.SetSSRFWhitelistFromRaw("") })
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"z"},{"id":"a"},{"id":"a"},{"id":""}]}`)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/discover", (&ModelHandler{}).DiscoverModels)
	req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(
		fmt.Sprintf(`{"base_url":%q,"api_key":"test-key"}`, upstream.URL+"/v1")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"success":true,"data":["a","z"]}`, w.Body.String())
}
