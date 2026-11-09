package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAutomationDeviceInventoryAndRouteIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:automation_device_routes_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	connection, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer connection.Close()
	connection.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Peer{}, &model.APIToken{}); err != nil { t.Fatal(err) }
	previousDB, previousServices := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}, PeerService: &service.PeerService{}}
	defer func(){service.DB, service.AllService = previousDB, previousServices}()
	isAdmin := true
	user := &model.User{Username: "automation-owner", IsAdmin: &isAdmin, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	device := &model.Peer{Id: "desk-automation-101", UserId: user.Id, Hostname: "desk-automation"}
	if err := db.Create(device).Error; err != nil { t.Fatal(err) }
	_, readToken, err := service.IssueAPIToken(user.Id, "Device observer", []string{"devices.read"}, 0)
	if err != nil { t.Fatal(err) }
	_, writeToken, err := service.IssueAPIToken(user.Id, "Device editor", []string{"devices.manage"}, 0)
	if err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	AutomationRoutes(engine)
	send := func(method, path, token string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, nil)
		if token != "" { request.Header.Set("Authorization", "Bearer "+token) }
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		return recorder
	}
	collection := send("GET", "/api/automation/v1/devices?page=1&page_size=20", readToken)
	if collection.Code != http.StatusOK || !strings.Contains(collection.Body.String(), "desk-automation-101") {
		t.Fatalf("device list unavailable to read token: %d %s", collection.Code, collection.Body.String())
	}
	detail := send("GET", "/api/automation/v1/devices/1", readToken)
	if detail.Code != http.StatusOK || !strings.Contains(detail.Body.String(), "desk-automation-101") {
		t.Fatalf("device detail unavailable to read token: %d %s", detail.Code, detail.Body.String())
	}
	for _, check := range []struct{method,path,token string; status int}{
		{"POST", "/api/automation/v1/devices/delete", readToken, http.StatusUnauthorized},
		{"GET", "/api/automation/v1/devices", writeToken, http.StatusUnauthorized},
		{"GET", "/api/automation/v1/devices", "", http.StatusUnauthorized},
		{"GET", "/api/automation/v1/deployments", readToken, http.StatusUnauthorized},
		{"GET", "/api/automation/v1/audits/files", readToken, http.StatusUnauthorized},
	} {
		response := send(check.method, check.path, check.token)
		if response.Code != check.status {
			t.Errorf("%s %s returned %d, expected %d", check.method, check.path, response.Code, check.status)
		}
	}
	var count int64
	if err := db.Model(&model.Peer{}).Where("id = ?", device.Id).Count(&count).Error; err != nil { t.Fatal(err) }
	if count != 1 { t.Fatal("read-only token changed the device inventory") }
}
