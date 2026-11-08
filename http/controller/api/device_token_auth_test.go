package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOfficialDeviceCLIWithScopedAPIToken(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:device_cli_token_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.APIToken{});err!=nil{t.Fatal(err)}
	oldDB,oldService:=service.DB,service.AllService
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{}}
	defer func(){service.DB,service.AllService=oldDB,oldService}()
	user:=&model.User{Username:"team",Status:model.COMMON_STATUS_ENABLE}
	if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"team-pc-001",Uuid:"device-uuid-001",UserId:user.Id}
	if err:=db.Create(peer).Error;err!=nil{t.Fatal(err)}
	_,token,err:=service.IssueAPIToken(user.Id,"Device assignments",[]string{"devices.assign"},0)
	if err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/devices/cli",middleware.DeviceCommandAuth("devices.assign"),(&DeviceCommands{}).Assign)
	router.POST("/api/devices/deploy",middleware.DeviceCommandAuth("devices.deploy"),(&DeviceCommands{}).Deploy)
	call:=func(path,access string)*httptest.ResponseRecorder{
		t.Helper()
		raw,_:=json.Marshal(map[string]string{"id":peer.Id,"uuid":peer.Uuid,"note":"Managed from CLI"})
		req:=httptest.NewRequest(http.MethodPost,path,bytes.NewReader(raw))
		req.Header.Set("Authorization","Bearer "+access)
		req.Header.Set("Content-Type","application/json")
		rec:=httptest.NewRecorder()
		router.ServeHTTP(rec,req)
		return rec
	}
	if response:=call("/api/devices/cli",token);response.Code!=http.StatusOK{
		t.Fatalf("scoped assignment rejected: %d %s",response.Code,response.Body.String())
	}
	var updated model.Peer
	if err:=db.First(&updated,peer.RowId).Error;err!=nil{t.Fatal(err)}
	if updated.Note!="Managed from CLI"{t.Fatalf("assignment not saved: %+v",updated)}
	if response:=call("/api/devices/deploy",token);response.Code!=http.StatusUnauthorized{
		t.Fatalf("out-of-scope deploy accepted: %d %s",response.Code,response.Body.String())
	}
	if response:=call("/api/devices/cli","invalid-token");response.Code!=http.StatusUnauthorized{
		t.Fatalf("unknown token accepted: %d %s",response.Code,response.Body.String())
	}
}
