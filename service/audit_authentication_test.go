package service

import (
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

// The RustDesk client reports "new" when the transport opens and sends the
// peer identity in a later audit request after successful authentication.
func TestAuditAuthenticationIsEmittedOncePerConnection(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.AuditConn{}, &model.AuditEventNonce{}); err != nil { t.Fatal(err) }
    client := &model.User{Username:"auth-audit-source", Status:model.COMMON_STATUS_ENABLE}
    target := &model.User{Username:"auth-audit-target", Status:model.COMMON_STATUS_ENABLE}
    for _, user := range []*model.User{client,target} {
        if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
    }
    for _, peer := range []*model.Peer{
        {Id:"auth-audit-client", UserId:client.Id},
        {Id:"auth-audit-target", UserId:target.Id},
    }{
        if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
    }
    audit := &AuditService{}
    start := &model.AuditConn{
        PeerId:"auth-audit-target",ConnId:114,Action:"new",EventNonce:"auth-start",
    }
    if err := audit.SaveAuditConnection(start);err!=nil{t.Fatal(err)}
    if start.AuthenticatedNew || start.Id==0 {t.Fatalf("start marked as authenticated: %+v",start)}
    auth := &model.AuditConn{
        PeerId:"auth-audit-target",ConnId:114,FromPeer:"auth-audit-client",
        PrimaryAuth:2,EventNonce:"auth-success",
    }
    if err := audit.SaveAuditConnection(auth);err!=nil{t.Fatal(err)}
    if !auth.AuthenticatedNew || auth.Id!=start.Id {
        t.Fatalf("authenticated audit did not trigger transition: %+v",auth)
    }
    // Duplicate delivery with the same nonce is ignored, even if reused in memory.
    if err := audit.SaveAuditConnection(auth);err!=nil{t.Fatal(err)}
    if auth.AuthenticatedNew {t.Fatal("same audit nonce emitted authentication again")}
    repeat := &model.AuditConn{
        PeerId:"auth-audit-target",ConnId:114,FromPeer:"auth-audit-client",
        PrimaryAuth:2,EventNonce:"auth-success-different-nonce",
    }
    if err := audit.SaveAuditConnection(repeat);err!=nil{t.Fatal(err)}
    if repeat.AuthenticatedNew {t.Fatal("repeated authenticated detail emitted again")}
    closeEvent := &model.AuditConn{
        PeerId:"auth-audit-target",ConnId:114,Action:"close",EventNonce:"auth-close",
    }
    if err := audit.SaveAuditConnection(closeEvent);err!=nil{t.Fatal(err)}
    if closeEvent.AuthenticatedNew {t.Fatal("connection close emitted authentication")}
    var saved model.AuditConn
    if err := DB.First(&saved,start.Id).Error;err!=nil{t.Fatal(err)}
    if saved.FromPeer!="auth-audit-client" || saved.PrimaryAuth!=2 || saved.CloseTime==0 {
        t.Fatalf("audit state was not merged correctly: %+v",saved)
    }
}

func TestAuditAuthenticationArrivesBeforeConnectionOpenAudit(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.AuditConn{}, &model.AuditEventNonce{}); err != nil { t.Fatal(err) }
    svc := &AuditService{}
    first := &model.AuditConn{
        PeerId:"target-out-of-order", FromPeer:"source-out-of-order",
        ConnId:900,EventNonce:"auth-before-open",
    }
    if err:=svc.SaveAuditConnection(first);err!=nil{t.Fatal(err)}
    if !first.AuthenticatedNew || first.Id==0 {
        t.Fatalf("first authenticated report was not recorded: %+v",first)
    }
    late := &model.AuditConn{
        PeerId:"target-out-of-order",ConnId:900,Action:"new",EventNonce:"late-open",
    }
    if err:=svc.SaveAuditConnection(late);err!=nil{t.Fatal(err)}
    if late.AuthenticatedNew {t.Fatal("late opening audit produced duplicate authentication")}
    var count int64
    if err:=DB.Model(&model.AuditConn{}).Count(&count).Error;err!=nil{t.Fatal(err)}
    if count!=1 {t.Fatalf("unexpected audit rows: %d",count)}
}
