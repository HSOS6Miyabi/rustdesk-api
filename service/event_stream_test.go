package service

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestStreamSubscribersAreScopedAndCleanedUp(t *testing.T) {
    one, err := SubscribePushEvents(840101, []string{"device.online", "device.online"})
    if err != nil { t.Fatal(err) }
    other, err := SubscribePushEvents(840102, []string{"connection.ended"})
    if err != nil { t.Fatal(err) }
    defer UnsubscribePushEvents(one)
    defer UnsubscribePushEvents(other)

    if !HasLivePushSubscribers("device.online") { t.Fatal("subscriber not registered") }
    if _, err := SubscribePushEvents(840101, []string{"unknown.event"}); err == nil {
        t.Fatal("unknown event was accepted")
    }
    PublishLivePushForUsers([]uint{840101, 840101, 840102}, "device.online", []byte("{\"ok\":true}"))
    select {
    case event := <-one.Messages:
        if event.Type != "device.online" { t.Fatalf("unexpected event: %s", event.Type) }
    default: t.Fatal("owner did not receive an event")
    }
    select {
    case <-other.Messages: t.Fatal("another user received the event")
    default:
    }
    for i:=0; i<streamBufferSize+1; i++ {
        PublishLivePushForUsers([]uint{840101}, "device.online", []byte("{}"))
    }
    select {
    case <-one.Done:
    default: t.Fatal("slow subscriber was not disconnected")
    }
    UnsubscribePushEvents(one)
    UnsubscribePushEvents(other)
    if HasLivePushSubscribers("device.online") || HasLivePushSubscribers("connection.ended") {
        t.Fatal("subscriber registry was not cleaned up")
    }
}

func TestStreamKeysAreOwnedByTheirCreator(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.APIToken{}); err != nil { t.Fatal(err) }
    first := &model.User{Username:"stream-key-owner", Status:model.COMMON_STATUS_ENABLE}
    second := &model.User{Username:"stream-key-other", Status:model.COMMON_STATUS_ENABLE}
    for _, user := range []*model.User{first,second} {
        if err := DB.Create(user).Error;err!=nil{t.Fatal(err)}
    }
    key, raw, err := IssueAPIToken(first.Id, "Personal stream", []string{"events.read"}, 1)
    if err != nil { t.Fatal(err) }
    if _, err := AuthenticateAPIToken(raw, "events.read"); err != nil { t.Fatal(err) }
    if _, err := AuthenticateAPIToken(raw, "devices.read"); err == nil {
        t.Fatal("SSE key could read management data")
    }
    keys, err := ListPushStreamTokens(second.Id)
    if err != nil || len(keys) != 0 { t.Fatalf("keys leaked to another user: %v %v", keys, err) }
    if err := RevokePushStreamToken(second.Id, key.Id); err == nil { t.Fatal("another user revoked the token") }
    keys, err = ListPushStreamTokens(first.Id)
    if err != nil || len(keys) != 1 || keys[0].TokenHash != key.TokenHash {
        t.Fatalf("owner's stream key not listed: %v %v", keys, err)
    }
    if err := RevokePushStreamToken(first.Id, key.Id); err != nil { t.Fatal(err) }
    if _, err := AuthenticateAPIToken(raw, "events.read"); err == nil { t.Fatal("revoked key is still valid") }
}

func TestSSEReceivesAnEventWithoutAnOutgoingChannel(t *testing.T) {
    cleanup := setupPushTestDB(t)
    defer cleanup()
    owner := &model.User{Username:"stream-only-owner", Status:model.COMMON_STATUS_ENABLE}
    if err := DB.Create(owner).Error; err != nil { t.Fatal(err) }
    peer := &model.Peer{Id:"stream-only-desk", UserId:owner.Id, Hostname:"Stream computer"}
    if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
    sub, err := SubscribePushEvents(owner.Id, []string{"connection.started"})
    if err != nil { t.Fatal(err) }
    defer UnsubscribePushEvents(sub)

    PublishPushEvent(PushEvent{
        Code:"connection.started", DeviceID:peer.Id, FromPeer:"stream-other",
        ConnID:41, Reference:"stream-start-41", When:time.Now(),
    })
    processed, err := ProcessPushEventOutbox(10)
    if err != nil || processed != 1 { t.Fatalf("event outbox: %d %v",processed,err) }
    select {
    case msg := <-sub.Messages:
        var payload PushEventEnvelope
        if err:=json.Unmarshal(msg.Data, &payload);err!=nil{t.Fatal(err)}
        if payload.Type!="connection.started" || payload.DeviceID!=peer.Id || payload.ConnectionID!=41 {
            t.Fatalf("unexpected JSON event: %+v",payload)
        }
    default: t.Fatal("SSE did not receive the event")
    }
    var deliveries int64
    if err:=DB.Model(&model.PushDelivery{}).Count(&deliveries).Error;err!=nil||deliveries!=0 {
        t.Fatalf("unexpected outgoing deliveries: %d %v", deliveries,err)
    }
}
