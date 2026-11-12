package service

import (
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestEnrichActiveConnectionsReportsParticipantsAndOwnership(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.AuditConn{}, &model.DeviceActiveConnection{}); err != nil { t.Fatal(err) }
    ordinary := false
    alice := model.User{Username:"alice-session",IsAdmin:&ordinary}
    bob := model.User{Username:"bob-session",IsAdmin:&ordinary}
    for _, u := range []*model.User{&alice,&bob} {
        if err := DB.Create(u).Error; err != nil { t.Fatal(err) }
    }
    for _, p := range []model.Peer{
        {Id:"desk-target-session",Hostname:"Workstation",Alias:"Office",UserId:bob.Id},
        {Id:"desk-source-session",Hostname:"Support PC",UserId:alice.Id},
    } {
        if err := DB.Create(&p).Error; err != nil { t.Fatal(err) }
    }
    audit := model.AuditConn{PeerId:"desk-target-session",ConnId:42,FromPeer:"desk-source-session",FromName:"operator",Ip:"203.0.113.4"}
    if err := DB.Create(&audit).Error; err != nil { t.Fatal(err) }
    rows := []model.DeviceActiveConnection{
        {DeviceId:"desk-target-session",ConnId:42,LastSeenAt:time.Now().Unix()},
        {DeviceId:"desk-target-session",ConnId:41,LastSeenAt:time.Now().Unix()},
    }
    enriched, err := EnrichActiveConnections(rows)
    if err != nil { t.Fatal(err) }
    if len(enriched) != 2 { t.Fatalf("unexpected number of connections: %d",len(enriched)) }
    if enriched[0].FromPeer != "desk-source-session" || enriched[0].FromAccount != alice.Username ||
        enriched[0].ToAccount != bob.Username || enriched[0].ToDeviceName != "Office" ||
        enriched[0].FromName != "operator" || !enriched[0].ParticipantRecorded {
        t.Fatalf("participants not enriched: %+v",enriched[0])
    }
    if enriched[1].FromPeer != "" || enriched[1].ParticipantRecorded || enriched[1].ToAccount != bob.Username {
        t.Fatalf("unrecorded source should remain absent: %+v",enriched[1])
    }
}
