package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutomationTokenRequiresScopeAndAdministrator(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:automation_auth_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	connection, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer connection.Close()
	connection.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.APIToken{}); err != nil { t.Fatal(err) }
	originalDB, originalServices := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}}
	defer func() { service.DB, service.AllService = originalDB, originalServices }()
	administratorFlag, memberFlag := true, false
	admin := &model.User{Username: "automation-admin", Status: model.COMMON_STATUS_ENABLE, IsAdmin: &administratorFlag}
	member := &model.User{Username: "automation-member", Status: model.COMMON_STATUS_ENABLE, IsAdmin: &memberFlag}
	for _, user := range []*model.User{admin, member} {
		if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	}
	_, readToken, err := service.IssueAPIToken(admin.Id, "Read inventory", []string{"devices.read"}, 0)
	if err != nil { t.Fatal(err) }
	_, writeToken, err := service.IssueAPIToken(admin.Id, "Manage inventory", []string{"devices.manage"}, 0)
	if err != nil { t.Fatal(err) }
	_, memberToken, err := service.IssueAPIToken(member.Id, "Member inventory", []string{"devices.read"}, 0)
	if err != nil { t.Fatal(err) }

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/read", AutomationTokenAuth("devices.read"), func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)
		if u == nil { t.Error("authenticated user missing") }
		c.Status(http.StatusNoContent)
	})
	engine.POST("/write", AutomationTokenAuth("devices.manage"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	send := func(method, path, credential string) int {
		t.Helper()
		request := httptest.NewRequest(method, path, nil)
		if credential != "" { request.Header.Set("Authorization", credential) }
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		return response.Code
	}
	cases := []struct {
		name, method, path, token string
		expected int
	}{
		{"authorized read", "GET", "/read", "Bearer " + readToken, http.StatusNoContent},
		{"read scope cannot write", "POST", "/write", "Bearer " + readToken, http.StatusUnauthorized},
		{"write scope cannot read", "GET", "/read", "Bearer " + writeToken, http.StatusUnauthorized},
		{"authorized write", "POST", "/write", "Bearer " + writeToken, http.StatusNoContent},
		{"ordinary user rejected", "GET", "/read", "Bearer " + memberToken, http.StatusForbidden},
		{"missing token rejected", "GET", "/read", "", http.StatusUnauthorized},
		{"browser session rejected", "GET", "/read", "Bearer browser-session", http.StatusUnauthorized},
		{"unknown token rejected", "GET", "/read", "Bearer rdapi_invalid", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if status := send(tc.method, tc.path, tc.token); status != tc.expected {
				t.Fatalf("unexpected HTTP status: got %d, want %d", status, tc.expected)
			}
		})
	}
	if err := db.Model(admin).Update("is_admin", false).Error; err != nil { t.Fatal(err) }
	if status := send("GET", "/read", "Bearer "+readToken); status != http.StatusForbidden {
		t.Fatalf("removed administrator still accessed automation: %d", status)
	}
	if err := db.Model(admin).Update("is_admin", true).Error; err != nil { t.Fatal(err) }
	var record model.APIToken
	if err := db.Where("user_id = ? AND permissions = ?", admin.Id, "devices.read").First(&record).Error; err != nil { t.Fatal(err) }
	if err := service.RevokeAPIToken(record.Id); err != nil { t.Fatal(err) }
	if status := send("GET", "/read", "Bearer "+readToken); status != http.StatusUnauthorized {
		t.Fatalf("revoked token still worked: %d", status)
	}
}
