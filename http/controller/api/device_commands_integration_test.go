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

func TestDeviceCommandsWithSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_commands?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Peer{}, &model.User{}, &model.DeviceGroup{}, &model.DevicePolicy{},
		&model.DevicePolicyAssignment{}, &model.AddressBook{}, &model.AddressBookCollection{},
		&model.DeviceDeployment{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldService, oldConfig, oldGlobalConfig := service.DB, service.AllService, service.Config, global.Config
	conf := &config.Config{}
	conf.App.RequireDeviceDeployment = true
	conf.App.InternalKey = "example-private-internal-key"
	service.DB = db
	service.Config = conf
	service.AllService = &service.Service{UserService: &service.UserService{}}
	global.Config = conf
	defer func() {
		service.DB, service.AllService, service.Config, global.Config = oldDB, oldService, oldConfig, oldGlobalConfig
	}()
	adminFlag := true
	administrator := &model.User{Username: "administrator", Status: model.COMMON_STATUS_ENABLE, IsAdmin: &adminFlag}
	user := &model.User{Username: "alice", Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(administrator).Error; err != nil { t.Fatal(err) }
	if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	device := &model.Peer{Id: "managed-10001", Uuid: "device-uuid-10001", UserId: administrator.Id}
	if err := db.Create(device).Error; err != nil { t.Fatal(err) }
	group := &model.DeviceGroup{Name: "Office"}
	policy := &model.DevicePolicy{Name: "Work"}
	if err := db.Create(group).Error; err != nil { t.Fatal(err) }
	if err := db.Create(policy).Error; err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/devices/cli", func(c *gin.Context) { c.Set("curUser", administrator); c.Next() }, (&DeviceCommands{}).Assign)
	router.POST("/api/devices/deploy", func(c *gin.Context) { c.Set("curUser", administrator); c.Next() }, (&DeviceCommands{}).Deploy)
	router.POST("/internal/v1/device/authorize", (&DeviceCommands{}).AuthorizeRegistration)
	send := func(path string, body interface{}, internalKey string) *httptest.ResponseRecorder {
		t.Helper()
		encoded, err := json.Marshal(body)
		if err != nil { t.Fatal(err) }
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
		if internalKey != "" { req.Header.Set("X-RustDesk-Internal-Key", internalKey) }
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	assign := send("/api/devices/cli", map[string]string{
		"id": device.Id, "uuid": device.Uuid, "user_name": "alice", "device_group_name": "Office",
		"strategy_name": "Work", "address_book_name": "Team", "address_book_tag": "helpdesk,work",
		"address_book_alias": "Front desk", "note": "Remote workstation", "device_name": "office-pc",
	}, "")
	if assign.Code != http.StatusOK { t.Fatalf("assign failed: %d %s", assign.Code, assign.Body.String()) }
	if err := db.First(device, device.RowId).Error; err != nil { t.Fatal(err) }
	if device.UserId != user.Id || device.GroupId != group.Id || device.Note != "Remote workstation" {
		t.Fatalf("device assignment not saved: %+v", device)
	}
	var assigned model.DevicePolicyAssignment
	if err := db.Where("device_id = ?", device.Id).First(&assigned).Error; err != nil { t.Fatal(err) }
	if assigned.PolicyId != policy.Id { t.Fatal("strategy assignment did not match") }
	var collection model.AddressBookCollection
	if err := db.Where("name = ? AND user_id = ?", "Team", user.Id).First(&collection).Error; err != nil { t.Fatal(err) }
	var entry model.AddressBook
	if err := db.Where("collection_id = ? AND id = ?", collection.Id, device.Id).First(&entry).Error; err != nil { t.Fatal(err) }
	if entry.Alias != "Front desk" || !strings.Contains(entry.Tags.String(), "helpdesk") {
		t.Fatalf("address book entry is incomplete: %+v", entry)
	}
	invalid := send("/api/devices/cli", map[string]string{
		"id": device.Id, "uuid": "wrong-device-uuid", "note": "must not change",
	}, "")
	if invalid.Code == http.StatusOK { t.Fatal("mismatched UUID was accepted") }
	if err := db.First(device, device.RowId).Error; err != nil { t.Fatal(err) }
	if device.Note != "Remote workstation" { t.Fatal("wrong UUID modified device") }
	uuid := base64.StdEncoding.EncodeToString([]byte("device-uuid-binary"))
	pk := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("p", 32)))
	deploy := map[string]string{"id": "deployed-10002", "uuid": uuid, "pk": pk}
	if result := send("/api/devices/deploy", deploy, ""); !strings.Contains(result.Body.String(), `"OK"`) {
		t.Fatalf("deployment rejected: %s", result.Body.String())
	}
	if result := send("/internal/v1/device/authorize", deploy, ""); result.Code != http.StatusForbidden {
		t.Fatalf("internal API did not require credentials: %d", result.Code)
	}
	if result := send("/internal/v1/device/authorize", deploy, conf.App.InternalKey); !strings.Contains(result.Body.String(), `"OK"`) {
		t.Fatalf("registered device was not accepted: %s", result.Body.String())
	}
	taken := map[string]string{"id": "deployed-10002", "uuid": base64.StdEncoding.EncodeToString([]byte("another-device-uuid")), "pk": pk}
	if result := send("/api/devices/deploy", taken, ""); !strings.Contains(result.Body.String(), "ID_TAKEN") {
		t.Fatalf("conflicting registration should be rejected: %s", result.Body.String())
	}
	if result := send("/internal/v1/device/authorize", taken, conf.App.InternalKey); !strings.Contains(result.Body.String(), "ID_TAKEN") {
		t.Fatalf("conflicting device should be rejected: %s", result.Body.String())
	}
	unknown := map[string]string{"id": "unknown-12345", "uuid": uuid, "pk": pk}
	if result := send("/internal/v1/device/authorize", unknown, conf.App.InternalKey); !strings.Contains(result.Body.String(), "NOT_DEPLOYED") {
		t.Fatalf("unregistered device should be rejected: %s", result.Body.String())
	}
}
