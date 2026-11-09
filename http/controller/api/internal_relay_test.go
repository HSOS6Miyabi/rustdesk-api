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

func TestInternalRelayNodeListingRequiresSharedKey(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:internal_relays_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.RelayNode{});err!=nil{t.Fatal(err)}
	oldDB,oldConfig:=service.DB,global.Config
	defer func(){service.DB,global.Config=oldDB,oldConfig}()
	service.DB=db
	global.Config.App.InternalKey="relay-internal-test-key"
	if err:=db.Create(&model.RelayNode{
		Name:"Primary",Address:"relay-a.example.test:21117",
		Subnets:"192.0.2.0/24,2001:db8::/32",Countries:"JP,US",Priority:5,Enabled:true,
	}).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.RelayNode{
		Name:"Disabled",Address:"relay-b.example.test:21117",Priority:10,Enabled:false,
	}).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.GET("/internal/v1/relay/nodes",(&InternalRelay{}).Nodes)
	check:=func(key string)*httptest.ResponseRecorder{
		t.Helper()
		req:=httptest.NewRequest(http.MethodGet,"/internal/v1/relay/nodes",nil)
		req.Header.Set("X-RustDesk-Internal-Key",key)
		result:=httptest.NewRecorder()
		router.ServeHTTP(result,req)
		return result
	}
	if res:=check("");res.Code!=http.StatusForbidden{t.Fatal("missing key accepted")}
	if res:=check("incorrect");res.Code!=http.StatusForbidden{t.Fatal("incorrect key accepted")}
	result:=check("relay-internal-test-key")
	if result.Code!=http.StatusOK{t.Fatalf("keyed request failed: %d %s",result.Code,result.Body.String())}
	var body struct{
		Nodes []struct{
			Address string `json:"address"`
			Subnets []string `json:"subnets"`
			Countries []string `json:"countries"`
			Priority int `json:"priority"`
		} `json:"nodes"`
	}
	if err:=json.Unmarshal(result.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
	if len(body.Nodes)!=1||body.Nodes[0].Address!="relay-a.example.test:21117"||
		body.Nodes[0].Priority!=5||len(body.Nodes[0].Subnets)!=2||len(body.Nodes[0].Countries)!=2{
		t.Fatalf("unexpected active relay node response: %+v",body)
	}
	if got:=result.Header().Get("Cache-Control");got!="no-store"{
		t.Fatalf("relay settings response is cacheable: %q",got)
	}
}
