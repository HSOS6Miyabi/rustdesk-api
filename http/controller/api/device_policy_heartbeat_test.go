package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviceHeartbeatReturnsNativeRustDeskStrategyOnlyForMatchingUUID(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:heartbeat_strategy_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	sql,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer sql.Close()
	sql.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.Peer{},&model.DevicePolicy{},&model.DevicePolicyAssignment{},&model.DeviceActiveConnection{},&model.DeviceDisconnectRequest{});err!=nil{t.Fatal(err)}
	oldDB,oldService,oldLogger,oldConfig:=service.DB,service.AllService,service.Logger,service.Config
	service.New(&config.Config{},db,log.New(),nil,nil)
	service.AllService.PeerService=&service.PeerService{}
	defer func(){
		service.DB,service.AllService,service.Logger,service.Config=oldDB,oldService,oldLogger,oldConfig
	}()
	peer:=&model.Peer{Id:"heartbeat-managed",Uuid:"uuid-registered"}
	if err:=db.Create(peer).Error;err!=nil{t.Fatal(err)}
	policy:=&model.DevicePolicy{Name:"Desk strategy",IdWhitelist:"allowed-123"}
	if err:=service.SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if err:=service.SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{DeviceId:peer.Id,PolicyId:policy.Id});err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/heartbeat",(&Index{}).Heartbeat)
	check:=func(uuid string,modifiedAt int64)map[string]json.RawMessage{
		t.Helper()
		in,_:=json.Marshal(map[string]interface{}{"id":peer.Id,"uuid":uuid,"ver":10501,"modified_at":modifiedAt})
		w:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodPost,"/api/heartbeat",bytes.NewReader(in))
		router.ServeHTTP(w,req)
		if w.Code!=http.StatusOK{t.Fatalf("heartbeat HTTP %d: %s",w.Code,w.Body.String())}
		var result map[string]json.RawMessage
		if err:=json.Unmarshal(w.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
		return result
	}
	if unverified:=check("spoofed-device-uuid",0);len(unverified)!=0{
		t.Fatalf("wrong UUID disclosed security policy: %v",unverified)
	}
	first:=check("uuid-registered",0)
	if first["strategy"]==nil || first["modified_at"]==nil{
		t.Fatalf("RustDesk strategy response missing fields: %v",first)
	}
	var version int64
	if err:=json.Unmarshal(first["modified_at"],&version);err!=nil||version<=0{
		t.Fatalf("invalid change marker %s %v",first["modified_at"],err)
	}
	var strategy struct {ConfigOptions map[string]string `json:"config_options"`}
	if err:=json.Unmarshal(first["strategy"],&strategy);err!=nil{t.Fatal(err)}
	if strategy.ConfigOptions["id-whitelist"]!="allowed-123"{
		t.Fatalf("RustDesk config_options missing: %+v",strategy)
	}
	if unchanged:=check("uuid-registered",version);len(unchanged)!=0{
		t.Fatalf("same policy must not be sent again: %v",unchanged)
	}
}


func TestManagedHeartbeatChecksApprovedDeployment(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:managed_heartbeat_approval?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{}, &model.Peer{}, &model.DeviceDeployment{},
		&model.DevicePolicy{}, &model.DevicePolicyAssignment{},
		&model.DeviceActiveConnection{}, &model.DeviceDisconnectRequest{},
	); err != nil { t.Fatal(err) }

	oldDB, oldService, oldLogger, oldConfig, oldGlobalConfig := service.DB, service.AllService, service.Logger, service.Config, global.Config
	service.New(&config.Config{}, db, log.New(), nil, nil)
	service.AllService.PeerService = &service.PeerService{}
	global.Config.App.RequireDeviceDeployment = true
	defer func() {
		service.DB, service.AllService, service.Logger, service.Config, global.Config =
			oldDB, oldService, oldLogger, oldConfig, oldGlobalConfig
	}()

	adminFlag := true
	owner := &model.User{
		Username: "managed-owner", Status: model.COMMON_STATUS_ENABLE,
		IsAdmin: &adminFlag,
	}
	if err := db.Create(owner).Error; err != nil { t.Fatal(err) }
	uuidData := []byte("managed-heartbeat-device-uuid")
	uuid := base64.StdEncoding.EncodeToString(uuidData)
	peer := &model.Peer{Id: "managed-heartbeat-device", Uuid: uuid, UserId: owner.Id}
	if err := db.Create(peer).Error; err != nil { t.Fatal(err) }
	policy := &model.DevicePolicy{Name: "Managed heartbeat policy", IdWhitelist: "managed-allow"}
	if err := service.SaveDevicePolicy(policy); err != nil { t.Fatal(err) }
	if err := service.SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId: peer.Id, PolicyId: policy.Id,
	}); err != nil { t.Fatal(err) }

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/heartbeat", (&Index{}).Heartbeat)
	check := func(requestUUID string) map[string]json.RawMessage {
		t.Helper()
		body, err := json.Marshal(map[string]interface{}{
			"id": peer.Id, "uuid": requestUUID, "ver": 10501, "modified_at": 0,
		})
		if err != nil { t.Fatal(err) }
		req := httptest.NewRequest(http.MethodPost, "/api/heartbeat", bytes.NewReader(body))
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != http.StatusOK { t.Fatalf("heartbeat returned HTTP %d", out.Code) }
		var result map[string]json.RawMessage
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil { t.Fatal(err) }
		return result
	}
	if response := check(uuid); len(response) != 0 {
		t.Fatalf("undeployed device received policy: %v", response)
	}
	deployment := &model.DeviceDeployment{
		DeviceId: peer.Id, Uuid: uuid, PublicKey: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
		OwnerUserId: owner.Id, Status: "approved",
	}
	if err := db.Create(deployment).Error; err != nil { t.Fatal(err) }
	if response := check(uuid); response["strategy"] == nil {
		t.Fatalf("approved device did not receive policy: %v", response)
	}
	if response := check(base64.RawStdEncoding.EncodeToString(uuidData)); response["strategy"] == nil {
		t.Fatalf("equivalent UUID encoding was rejected: %v", response)
	}
	if response := check(base64.StdEncoding.EncodeToString([]byte("different-managed-device-uuid"))); len(response) != 0 {
		t.Fatalf("unregistered identity received policy: %v", response)
	}
	if err := db.Model(&model.DeviceDeployment{}).Where("device_id = ?", peer.Id).
		Update("owner_user_id", owner.Id + 100).Error; err != nil { t.Fatal(err) }
	if response := check(uuid); len(response) != 0 {
		t.Fatalf("device with conflicting owner received policy: %v", response)
	}
	if err := db.Model(&model.DeviceDeployment{}).Where("device_id = ?", peer.Id).
		Update("owner_user_id", owner.Id).Error; err != nil { t.Fatal(err) }
	if response := check(uuid); response["strategy"] == nil {
		t.Fatalf("restored ownership did not receive policy: %v", response)
	}
	if err := db.Model(&model.DeviceDeployment{}).Where("device_id = ?", peer.Id).
		Update("status", "revoked").Error; err != nil { t.Fatal(err) }
	if response := check(uuid); len(response) != 0 {
		t.Fatalf("revoked device received policy or session commands: %v", response)
	}
}
