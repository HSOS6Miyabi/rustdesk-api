package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestManagedDeploymentCreatesOwnedPeerAndProtectsSysInfo(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:managed_sysinfo_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sql.Close()
	sql.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{}, &model.Peer{}, &model.LoginLog{},
		&model.DeviceDeployment{}, &model.UserToken{},
	); err != nil {
		t.Fatal(err)
	}

	oldDB, oldService, oldConfig, oldGlobalConfig := service.DB, service.AllService, service.Config, global.Config
	conf := &config.Config{}
	conf.App.RequireDeviceDeployment = true
	conf.App.InternalKey = "isolated-test-key"
	service.DB = db
	service.Config = conf
	service.AllService = &service.Service{
		UserService: &service.UserService{},
		PeerService: &service.PeerService{},
	}
	global.Config = *conf
	defer func() {
		service.DB, service.AllService, service.Config, global.Config = oldDB, oldService, oldConfig, oldGlobalConfig
	}()

	admin := true
	administrator := &model.User{
		Username: "administrator", IsAdmin: &admin, Status: model.COMMON_STATUS_ENABLE,
	}
	target := &model.User{Username: "alice", Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(administrator).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(target).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/devices/deploy", func(c *gin.Context) {
		c.Set("curUser", administrator)
		c.Next()
	}, (&DeviceCommands{}).Deploy)
	router.POST("/api/devices/cli", func(c *gin.Context) {
		c.Set("curUser", administrator)
		c.Next()
	}, (&DeviceCommands{}).Assign)
	router.POST("/api/sysinfo", (&Peer{}).SysInfo)

	send := func(path string, payload interface{}) *httptest.ResponseRecorder {
		t.Helper()
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	id := "approved-test-device"
	uuid := base64.StdEncoding.EncodeToString([]byte("approved-device-uuid"))
	publicKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	deploy := map[string]string{"id": id, "uuid": uuid, "pk": publicKey}
	sysinfo := map[string]string{
		"id": id, "uuid": uuid, "hostname": "Office Linux",
		"os": "Linux", "version": "1.5.1",
	}

	if response := send("/api/sysinfo", sysinfo); response.Code != http.StatusForbidden {
		t.Fatalf("unregistered inventory HTTP %d", response.Code)
	}
	if response := send("/api/devices/deploy", deploy); response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"OK"`) {
		t.Fatalf("device deployment rejected: %d %s", response.Code, response.Body.String())
	}
	if response := send("/api/devices/deploy", deploy); response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), `"OK"`) {
		t.Fatalf("repeated deployment rejected: %d %s", response.Code, response.Body.String())
	}

	var peer model.Peer
	if err := db.Where("id = ?", id).First(&peer).Error; err != nil {
		t.Fatal(err)
	}
	if peer.Uuid != uuid || peer.UserId != administrator.Id {
		t.Fatalf("deployed inventory owner or UUID differs: %+v", peer)
	}
	var count int64
	if err := db.Model(&model.Peer{}).Where("id = ?", id).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("duplicate deployment created %d device records: %v", count, err)
	}

	previousSession := &model.UserToken{
		UserId: administrator.Id, DeviceId: id, DeviceUuid: uuid,
		Token: "previous-device-session", ExpiredAt: 4102444800,
	}
	otherSession := &model.UserToken{
		UserId: administrator.Id, DeviceId: "unrelated-device", DeviceUuid: "unrelated-uuid",
		Token: "unrelated-device-session", ExpiredAt: 4102444800,
	}
	if err := db.Create(previousSession).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(otherSession).Error; err != nil {
		t.Fatal(err)
	}
	assignment := map[string]string{
		"id": id, "uuid": uuid, "user_name": target.Username, "device_name": "Assigned Desk",
	}
	if response := send("/api/devices/cli", assignment); response.Code != http.StatusOK {
		t.Fatalf("assignment after deployment rejected: %d %s", response.Code, response.Body.String())
	}
	var deployment model.DeviceDeployment
	if err := db.Where("device_id = ?", id).First(&deployment).Error; err != nil {
		t.Fatal(err)
	}
	if deployment.OwnerUserId != target.Id {
		t.Fatal("device deployment owner did not follow administrator assignment")
	}
	var remaining int64
	if err := db.Model(&model.UserToken{}).Where("token = ?", previousSession.Token).
		Count(&remaining).Error; err != nil || remaining != 0 {
		t.Fatalf("previous device session remained active: %d, %v", remaining, err)
	}
	if err := db.Model(&model.UserToken{}).Where("token = ?", otherSession.Token).
		Count(&remaining).Error; err != nil || remaining != 1 {
		t.Fatalf("unrelated session was changed: %d, %v", remaining, err)
	}
	currentSession := &model.UserToken{
		UserId: target.Id, DeviceId: id, DeviceUuid: uuid,
		Token: "new-owner-device-session", ExpiredAt: 4102444800,
	}
	if err := db.Create(currentSession).Error; err != nil {
		t.Fatal(err)
	}
	if response := send("/api/devices/cli", assignment); response.Code != http.StatusOK {
		t.Fatalf("repeating an unchanged assignment was rejected: %d %s",
			response.Code, response.Body.String())
	}
	if err := db.Model(&model.UserToken{}).Where("token = ?", currentSession.Token).
		Count(&remaining).Error; err != nil || remaining != 1 {
		t.Fatalf("unchanged ownership revoked an active session: %d, %v", remaining, err)
	}
	if response := send("/api/sysinfo", map[string]string{
		"id": id, "uuid": base64.RawStdEncoding.EncodeToString([]byte("approved-device-uuid")),
		"hostname": "Updated Linux", "os": "Linux",
	}); response.Code != http.StatusOK {
		t.Fatalf("approved inventory HTTP %d %s", response.Code, response.Body.String())
	}
	if err := db.Where("id = ?", id).First(&peer).Error; err != nil {
		t.Fatal(err)
	}
	if peer.UserId != target.Id || peer.Hostname != "Updated Linux" || peer.Uuid != uuid {
		t.Fatalf("approved inventory changed device identity or ownership: %+v", peer)
	}
	if response := send("/api/sysinfo", map[string]string{
		"id": id, "uuid": base64.StdEncoding.EncodeToString([]byte("wrong-device-uuid")),
		"hostname": "Untrusted Host",
	}); response.Code != http.StatusForbidden {
		t.Fatalf("unapproved device identity HTTP %d", response.Code)
	}
	if err := db.Model(&model.DeviceDeployment{}).Where("device_id = ?", id).
		Update("status", "revoked").Error; err != nil {
		t.Fatal(err)
	}
	if response := send("/api/sysinfo", sysinfo); response.Code != http.StatusForbidden {
		t.Fatalf("revoked inventory HTTP %d", response.Code)
	}
	if err := db.Where("id = ?", id).First(&peer).Error; err != nil {
		t.Fatal(err)
	}
	if peer.Hostname != "Updated Linux" {
		t.Fatal("rejected inventory request changed saved device data")
	}
	global.Config.App.RequireDeviceDeployment = false
	if response := send("/api/sysinfo", map[string]string{
		"id": "legacy-device-001", "uuid": "legacy-uuid-001", "hostname": "Legacy Desk",
	}); response.Code != http.StatusOK {
		t.Fatalf("original unmanaged sysinfo behavior changed: %d", response.Code)
	}
}
