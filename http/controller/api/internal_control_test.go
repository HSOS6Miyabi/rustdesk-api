package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInternalControlPermissionsUseEffectiveRole(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:control_permissions_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.Peer{},&model.ControlRole{},&model.ControlRoleAssignment{});err!=nil{t.Fatal(err)}
	oldDB,oldCfg:=service.DB,global.Config
	defer func(){service.DB,global.Config=oldDB,oldCfg}()
	service.DB=db
	global.Config.App.InternalKey="internal-control-role-test-secret"
	device:=&model.Peer{Id:"support-device-222"}
	if err:=db.Create(device).Error;err!=nil{t.Fatal(err)}
	role:=&model.ControlRole{Name:"Read only",Keyboard:1,File:1,Clipboard:2,USB:1}
	if err:=db.Create(role).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.GET("/internal/v1/control/permissions",(&InternalControl{}).Permissions)
	send:=func(id,key string)*httptest.ResponseRecorder {
		t.Helper()
		req:=httptest.NewRequest(http.MethodGet,"/internal/v1/control/permissions?device_id="+id,nil)
		req.Header.Set("X-RustDesk-Internal-Key",key)
		w:=httptest.NewRecorder()
		router.ServeHTTP(w,req)
		return w
	}
	if result:=send(device.Id,"");result.Code!=http.StatusForbidden{t.Fatal("missing key accepted")}
	if result:=send(device.Id,"wrong");result.Code!=http.StatusForbidden{t.Fatal("wrong key accepted")}
	if result:=send("","internal-control-role-test-secret");result.Code!=http.StatusBadRequest{
		t.Fatal("empty device ID accepted")
	}
	type Result struct {
		Assigned bool `json:"assigned"`
		Permissions uint64 `json:"permissions"`
		Source string `json:"source"`
	}
	get:=func()Result{
		t.Helper()
		out:=send(device.Id,"internal-control-role-test-secret")
		if out.Code!=http.StatusOK{t.Fatalf("unexpected response: %d %s",out.Code,out.Body.String())}
		var data Result
		if err:=json.Unmarshal(out.Body.Bytes(),&data);err!=nil{t.Fatal(err)}
		return data
	}
	if got:=get();got.Assigned||got.Permissions!=0{t.Fatalf("unassigned device: %+v",got)}
	if err:=db.Create(&model.ControlRoleAssignment{
		TargetType:"device",TargetId:device.Id,RoleId:role.Id,
	}).Error;err!=nil{t.Fatal(err)}
	expected,err:=role.PermissionBits()
	if err!=nil{t.Fatal(err)}
	got:=get()
	if !got.Assigned||got.Permissions!=expected||got.Source!="device"{
		t.Fatalf("native client permission mask incorrect: %+v, want %d",got,expected)
	}
}
