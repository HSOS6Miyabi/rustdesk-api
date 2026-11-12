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

func TestConnectionAuthenticationEventIsReportedAfterSuccessfulAudit(t *testing.T) {
    db, err:=gorm.Open(sqlite.Open("file:authenticated_connection_route_test?mode=memory&cache=shared"),&gorm.Config{})
    if err!=nil{t.Fatal(err)}
    sqlDB,err:=db.DB();if err!=nil{t.Fatal(err)}
    defer sqlDB.Close()
    sqlDB.SetMaxOpenConns(1)
    if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.PushChannel{},
        &model.PushEventOutbox{},&model.AuditConn{},&model.AuditEventNonce{});err!=nil{t.Fatal(err)}
    originalDB, originalSvc:=service.DB,service.AllService
    service.DB=db
    service.AllService=&service.Service{AuditService:&service.AuditService{}}
    defer func(){service.DB,service.AllService=originalDB,originalSvc}()
    owner:=&model.User{Username:"authenticated-connection-user",Status:model.COMMON_STATUS_ENABLE}
    if err:=db.Create(owner).Error;err!=nil{t.Fatal(err)}
    device:=&model.Peer{Id:"authenticated-target",UserId:owner.Id}
    if err:=db.Create(device).Error;err!=nil{t.Fatal(err)}
    sub,err:=service.SubscribePushEvents(owner.Id,[]string{"connection.authenticated"})
    if err!=nil{t.Fatal(err)}
    defer service.UnsubscribePushEvents(sub)

    gin.SetMode(gin.TestMode)
    engine:=gin.New()
    engine.POST("/api/audit/conn",(&Audit{}).AuditConn)
    send:=func(payload map[string]interface{}){
        t.Helper()
        encoded,err:=json.Marshal(payload);if err!=nil{t.Fatal(err)}
        request:=httptest.NewRequest(http.MethodPost,"/api/audit/conn",bytes.NewReader(encoded))
        request.Header.Set("Content-Type","application/json")
        response:=httptest.NewRecorder()
        engine.ServeHTTP(response,request)
        if response.Code!=http.StatusOK {t.Fatalf("audit HTTP status %d: %s",response.Code,response.Body.String())}
        var result struct{Code int `json:"code"`}
        if err:=json.Unmarshal(response.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
        if result.Code!=0{t.Fatalf("audit rejected: %s",response.Body.String())}
    }
    send(map[string]interface{}{"id":device.Id,"conn_id":51,"action":"new","nonce":"start-auth-51"})
    select {case <-sub.Messages:t.Fatal("opening the transport should not report authentication");default:}
    send(map[string]interface{}{"id":device.Id,"conn_id":51,"peer":[]string{"authenticated-source","Support"}, "primary_auth":2,"nonce":"success-auth-51"})
    processed,err:=service.ProcessPushEventOutbox(10)
    if err!=nil||processed!=1{t.Fatalf("authenticated event outbox: %d %v",processed,err)}
    select {
    case evt:=<-sub.Messages:
        if evt.Type!="connection.authenticated"{t.Fatalf("unexpected event type: %s",evt.Type)}
        var data service.PushEventEnvelope
        if err:=json.Unmarshal(evt.Data,&data);err!=nil{t.Fatal(err)}
        if data.DeviceID!=device.Id||data.FromPeer!="authenticated-source"||data.ConnectionID!=51{
            t.Fatalf("missing authenticated connection details: %+v",data)
        }
    default:t.Fatal("authenticated connection event not delivered")
    }
    send(map[string]interface{}{"id":device.Id,"conn_id":51,"peer":[]string{"authenticated-source","Support"}, "primary_auth":2,"nonce":"repeated-auth-51"})
    processed,err=service.ProcessPushEventOutbox(10)
    if err!=nil||processed!=0{t.Fatalf("duplicate authenticated event: %d %v",processed,err)}
}
