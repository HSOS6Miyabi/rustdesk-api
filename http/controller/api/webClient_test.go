package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWebClientSharedPeerRejectsMalformedTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/shared-peer", (&WebClient{}).SharedPeer)
	for _, testcase := range []struct {
		name string
		body string
	}{
		{"empty JSON", `{}`},
		{"wrong type", `{"share_token":123}`},
		{"null", `{"share_token":null}`},
		{"array", `{"share_token":[]}`},
		{"malformed JSON", `{"share_token":`},
		{"very long token", `{"share_token":"` + strings.Repeat("x", 4100) + `"}`},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/shared-peer", strings.NewReader(testcase.body))
			req.Header.Set("Content-Type", "application/json")
			reply := httptest.NewRecorder()
			router.ServeHTTP(reply, req)
			if reply.Code >= http.StatusInternalServerError {
				t.Fatalf("malformed share token returned HTTP %d", reply.Code)
			}
			if !strings.Contains(reply.Body.String(), "share_token") {
				t.Fatalf("expected clear share-token validation, got: %s", reply.Body.String())
			}
		})
	}
}
