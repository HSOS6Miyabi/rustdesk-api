package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInternalConnectionAuthorizationChecksRuleAndToken(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:strict_connection_access_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	sqlDB,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.UserToken{},&model.Group{},
		&model.GroupMembership{},&model.AccessRule{});err!=nil{t.Fatal(err)}
	oldDB,oldConfig:=service.DB,global.Config
	defer func(){service.DB,global.Config=oldDB,oldConfig}()
	service.DB=db
	global.Config.App.InternalKey="team-access-internal-secret"
	u:=&model.User{Username:"team-member",Status:1}
	other:=&model.User{Username:"device-owner",Status:1}
	for _,account:=range []*model.User{u,other}{
		if err:=db.Create(account).Error;err!=nil{t.Fatal(err)}
	}
	if err:=db.Create(&model.Peer{Id:"team-desk",UserId:other.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.UserToken{
		UserId:u.Id,Token:"member-session",ExpiredAt:time.Now().Add(time.Hour).Unix(),
	}).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/internal/v1/connection/authorize",(&InternalConnectionAccess{}).Authorize)
	send:=func(token,key string)*httptest.ResponseRecorder{
		t.Helper()
		body,_:=json.Marshal(map[string]string{"device_id":"team-desk","token":token})
		request:=httptest.NewRequest(http.MethodPost,"/internal/v1/connection/authorize",bytes.NewReader(body))
		request.Header.Set("X-RustDesk-Internal-Key",key)
		request.Header.Set("Content-Type","application/json")
		result:=httptest.NewRecorder()
		router.ServeHTTP(result,request)
		return result
	}
	if result:=send("member-session","");result.Code!=http.StatusForbidden{
		t.Fatal("internal API accepted missing key")
	}
	if result:=send("member-session","wrong");result.Code!=http.StatusForbidden{
		t.Fatal("internal API accepted incorrect key")
	}
	result:=send("member-session","team-access-internal-secret")
	if result.Code!=http.StatusOK||!strings.Contains(result.Body.String(),`"allow":false`){
		t.Fatalf("unmatched access unexpectedly permitted: %s",result.Body.String())
	}
	if err:=db.Create(&model.AccessRule{
		Name:"Team support",SourceType:"all",TargetType:"device",TargetID:"team-desk",
		Effect:"allow",Enabled:true,
	}).Error;err!=nil{t.Fatal(err)}
	result=send("member-session","team-access-internal-secret")
	if result.Code!=http.StatusOK||!strings.Contains(result.Body.String(),`"allow":true`){
		t.Fatalf("shared device access not granted: %s",result.Body.String())
	}
	result=send("invalid-session","team-access-internal-secret")
	if result.Code!=http.StatusOK||!strings.Contains(result.Body.String(),`"allow":false`){
		t.Fatalf("invalid token accepted: %s",result.Body.String())
	}
	if result.Header().Get("Cache-Control")!="no-store"{t.Fatal("authorization result may be cached")}
}
