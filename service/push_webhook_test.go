package service

import (
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestWebhookPostsDirectJSONAndRetries(t *testing.T) {
    cleanup:=setupPushTestDB(t)
    defer cleanup()
    t.Setenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS","1")
    type receivedEvent struct {
        Path string
        Method string
        ContentType string
        Authorization string
        Payload PushEventEnvelope
    }
    received := make(chan receivedEvent, 2)
    failures := 0
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        data, _ := io.ReadAll(io.LimitReader(r.Body, 16384))
        var payload PushEventEnvelope
        if err := json.Unmarshal(data, &payload); err != nil {
            w.WriteHeader(http.StatusBadRequest);return
        }
        received<-receivedEvent{
            Path:r.URL.Path,Method:r.Method,ContentType:r.Header.Get("Content-Type"),
            Authorization:r.Header.Get("Authorization"),Payload:payload,
        }
        if failures==0 {
            failures++
            w.WriteHeader(http.StatusServiceUnavailable)
            return
        }
        w.WriteHeader(http.StatusNoContent)
    }))
    defer server.Close()
    owner:=&model.User{Username:"webhook-owner",Status:model.COMMON_STATUS_ENABLE}
    if err:=DB.Create(owner).Error;err!=nil{t.Fatal(err)}
    peer:=&model.Peer{Id:"webhook-desk",UserId:owner.Id,Hostname:"Desktop"}
    if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
    channel,err:=SavePushChannel(owner.Id,PushChannelInput{
        Kind:"webhook",Name:"Events JSON",Enabled:true,
        BaseURL:server.URL+"/events/inbox",Events:[]string{"connection.started"},
    })
    if err!=nil {t.Fatal(err)}
    event:=PushEvent{
        Code:"connection.started",Reference:"webhook-connection-1",
        DeviceID:peer.Id,ConnID:123,When:time.Now(),
    }
    if err:=QueuePushForUsers(event,[]uint{owner.Id});err!=nil{t.Fatal(err)}
    sent,err:=ProcessPushQueue(1)
    if err!=nil||sent!=0{t.Fatalf("expected retry: %d %v",sent,err)}
    first:=<-received
    if first.Method!="POST" || first.Path!="/events/inbox" ||
        first.ContentType!="application/json" || first.Authorization!="" ||
        first.Payload.Type!="connection.started" || first.Payload.DeviceID!=peer.Id ||
        first.Payload.ConnectionID!=123 || first.Payload.ID=="" {
        t.Fatalf("unexpected webhook request: %+v",first)
    }
    if err:=DB.Model(&model.PushDelivery{}).Where("channel_id = ?",channel.Id).
        Update("next_attempt_at",time.Now().Unix()-1).Error;err!=nil{t.Fatal(err)}
    sent,err=ProcessPushQueue(1)
    if err!=nil||sent!=1{t.Fatalf("webhook retry: %d %v",sent,err)}
    second:=<-received
    if first.Payload.ID!=second.Payload.ID {t.Fatal("retry changed the event ID")}
}
