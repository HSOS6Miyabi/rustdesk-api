package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupPerformanceAPI(t *testing.T) (*gorm.DB, *model.User, func()) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:performance_api_regression?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Peer{}, &model.LoginLog{},
		&model.DevicePolicy{}, &model.DevicePolicyAssignment{},
		&model.DevicePolicyUserAssignment{}, &model.DevicePolicyGroupAssignment{},
		&model.DeviceDeployment{}, &model.DeviceActiveConnection{}, &model.UserToken{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices, oldConf, oldGlobal := service.DB, service.AllService, service.Config, global.Config
	conf := &config.Config{}
	service.DB = db
	service.Config = conf
	global.Config = *conf
	service.AllService = &service.Service{UserService: &service.UserService{}, PeerService: &service.PeerService{}}
	admin := true
	user := &model.User{Username: "performance-admin", IsAdmin: &admin, Status: model.COMMON_STATUS_ENABLE}
	if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	return db, user, func() {
		service.DB, service.AllService, service.Config, global.Config = oldDB, oldServices, oldConf, oldGlobal
		conn.Close()
	}
}

func TestDeviceAssignmentImmediatelyUpdatesEffectiveStrategy(t *testing.T) {
	db, administrator, cleanup := setupPerformanceAPI(t)
	defer cleanup()
	peer := &model.Peer{Id: "strategy-perf-device", Uuid: "strategy-device-uuid", UserId: administrator.Id}
	if err := db.Create(peer).Error; err != nil { t.Fatal(err) }
	first := &model.DevicePolicy{Name: "Office restriction", IdWhitelist: "first"}
	second := &model.DevicePolicy{Name: "Home restriction", IdWhitelist: "second"}
	for _, policy := range []*model.DevicePolicy{first, second} {
		if err := service.SaveDevicePolicy(policy); err != nil { t.Fatal(err) }
	}
	check := func(modifiedAt int64) map[string]interface{} {
		t.Helper()
		result, err := service.HeartbeatDeviceStrategy(peer, modifiedAt)
		if err != nil { t.Fatal(err) }
		return result
	}
	if result := check(0); len(result) != 0 { t.Fatal("unassigned policy returned instructions") }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/devices/cli", func(c *gin.Context) { c.Set("curUser", administrator); c.Next() },
		(&DeviceCommands{}).Assign)
	assign := func(policy string, expected int) {
		t.Helper()
		payload, _ := json.Marshal(map[string]interface{}{
			"id": peer.Id, "uuid": peer.Uuid, "strategy_name": policy,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/devices/cli", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != expected { t.Fatalf("assignment response: %d %s", response.Code, response.Body.String()) }
	}
	assign(first.Name, http.StatusOK)
	response := check(0)
	revision, ok := response["modified_at"].(int64)
	if !ok || revision == 0 { t.Fatalf("first policy missing: %+v", response) }
	assign("missing policy", http.StatusBadRequest)
	if unchanged := check(revision); len(unchanged) != 0 {
		t.Fatal("failed assignment changed the active policy")
	}
	assign(second.Name, http.StatusOK)
	response = check(revision)
	next := response["modified_at"].(int64)
	if next == revision || next == 0 { t.Fatalf("policy change was delayed: %+v", response) }
	assign("", http.StatusOK)
	response = check(next)
	if response["modified_at"] != int64(0) { t.Fatalf("policy removal was delayed: %+v", response) }
}

func TestRepeatedSystemInformationDoesNotWriteDatabase(t *testing.T) {
	db, administrator, cleanup := setupPerformanceAPI(t)
	defer cleanup()
	peer := &model.Peer{Id: "sysinfo-perf-device", Uuid: "sysinfo-device-uuid", UserId: administrator.Id,
		Cpu: "cpu-a", Hostname: "host-a", Memory: "16G", Os: "linux", Username: "user-a", Version: "1.5.1"}
	if err := db.Create(peer).Error; err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/sysinfo", (&Peer{}).SysInfo)
	updates := 0
	key := "perf:sysinfo_update"
	if err := db.Callback().Update().Before("gorm:update").Register(key, func(*gorm.DB) { updates++ }); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Update().Remove(key)
	send := func(cpu string) {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{
			"id": peer.Id, "uuid": peer.Uuid, "cpu": cpu, "hostname": peer.Hostname,
			"memory": peer.Memory, "os": peer.Os, "username": peer.Username, "version": peer.Version,
		})
		request := httptest.NewRequest(http.MethodPost, "/api/sysinfo", bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, request)
		if w.Code != http.StatusOK { t.Fatalf("sysinfo failed: %d %s", w.Code, w.Body.String()) }
	}
	send("cpu-a")
	if updates != 0 { t.Fatalf("unchanged system information caused %d updates", updates) }
	send("cpu-b")
	if updates != 1 { t.Fatalf("changed system information caused %d updates", updates) }
	send("cpu-b")
	if updates != 1 { t.Fatalf("repeated change caused %d updates", updates) }
	request := httptest.NewRequest(http.MethodPost, "/api/sysinfo",
		bytes.NewReader([]byte(`{"id":"sysinfo-perf-device","hostname":"host-b"}`)))
	request.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, request)
	if w.Code != http.StatusOK { t.Fatalf("partial update failed: %d", w.Code) }
	var saved model.Peer
	if err := db.Where("id = ?", peer.Id).Take(&saved).Error; err != nil { t.Fatal(err) }
	if saved.Hostname != "host-b" || saved.Uuid != peer.Uuid ||
		saved.Cpu != "cpu-b" || saved.Os != peer.Os {
		t.Fatalf("partial inventory update cleared an existing field: %+v", saved)
	}
}
