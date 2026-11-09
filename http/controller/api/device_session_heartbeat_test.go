package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestActiveSessionsAndHeartbeatDisconnect(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:heartbeat_sessions_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	sql,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer sql.Close()
	sql.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.Peer{},&model.User{},
		&model.DevicePolicy{},&model.DevicePolicyAssignment{},
		&model.DeviceActiveConnection{},&model.DeviceDisconnectRequest{});err!=nil{t.Fatal(err)}
	previousDB,previousServices,previousLogger:=service.DB,service.AllService,service.Logger
	service.DB=db
	service.AllService=&service.Service{PeerService:&service.PeerService{},UserService:&service.UserService{}}
	service.Logger=nil
	defer func(){service.DB,service.AllService,service.Logger=previousDB,previousServices,previousLogger}()
	peer:=&model.Peer{Id:"active-test-device",Uuid:"registered-uuid"}
	if err:=db.Create(peer).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/heartbeat",(&Index{}).Heartbeat)
	heartbeat:=func(uuid string,conns []int32)map[string]json.RawMessage{
		t.Helper()
		in,_:=json.Marshal(map[string]interface{}{"id":peer.Id,"uuid":uuid,"conns":conns})
		w:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodPost,"/api/heartbeat",bytes.NewReader(in))
		router.ServeHTTP(w,req)
		if w.Code!=http.StatusOK{t.Fatalf("heartbeat: %d %s",w.Code,w.Body.String())}
		var result map[string]json.RawMessage
		if err:=json.Unmarshal(w.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
		return result
	}
	if returned:=heartbeat(peer.Uuid,[]int32{41,42,41});len(returned)>0{
		t.Fatalf("unassigned device got unexpected instructions: %v",returned)
	}
	rows,total,err:=service.ListActiveDeviceSessions(1,20)
	if err!=nil||total!=2||len(rows)!=2{t.Fatalf("active connections: %d %v %v",total,rows,err)}
	if err:=service.RequestDeviceDisconnect(peer.Id,41,1);err!=nil{t.Fatal(err)}
	if err:=service.RequestDeviceDisconnect(peer.Id,41,1);err!=nil{t.Fatal(err)}
	requests,count,err:=service.ListDisconnectRequests(1,20)
	if err!=nil||count!=1||len(requests)!=1{t.Fatalf("duplicate disconnect request: %d %v %v",count,requests,err)}
	if unexpected:=heartbeat("unregistered-uuid",[]int32{41,42});len(unexpected)>0{
		t.Fatalf("unverified device was served: %v",unexpected)
	}
	result:=heartbeat(peer.Uuid,[]int32{41,42})
	var requested []int32
	if err:=json.Unmarshal(result["disconnect"],&requested);err!=nil||len(requested)!=1||requested[0]!=41{
		t.Fatalf("missing disconnect request: %v %v",requested,err)
	}
	if second:=heartbeat(peer.Uuid,[]int32{41,42});second["disconnect"]==nil{
		t.Fatalf("disconnection was not retried: %v",second)
	}
	if result:=heartbeat(peer.Uuid,[]int32{42});len(result)>0{
		t.Fatalf("closed connection received instructions: %v",result)
	}
	requests,count,err=service.ListDisconnectRequests(1,20)
	if err!=nil||count!=1||requests[0].Status!="completed"||requests[0].CompletedAt==0{
		t.Fatalf("completion not saved: %v %v",requests,err)
	}
	if err:=service.RequestDeviceDisconnect(peer.Id,41,1);err==nil{
		t.Fatal("inactive connection should not accept a new request")
	}
}
