package service

import (
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestLogCleanupPreviewAndTerminalStatusProtection(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(
        &model.AuditRetentionPolicy{}, &model.AuditConn{}, &model.AuditFile{},
        &model.AuditAlarm{}, &model.LoginLog{}, &model.AuditOperation{},
        &model.DeviceDisconnectRequest{}, &model.SystemSettingHistory{},
        &model.PushDelivery{}, &model.NotificationDelivery{},
        &model.PushEventOutbox{}, &model.NotificationEventOutbox{},
    ); err != nil { t.Fatal(err) }

    now := time.Now()
    past := now.Add(-75 * 24 * time.Hour)
    insert := func(record interface{}, column string) {
        t.Helper()
        if err := DB.Create(record).Error; err != nil { t.Fatal(err) }
        if column != "" {
            if err := DB.Model(record).UpdateColumn(column, past).Error; err != nil { t.Fatal(err) }
        }
    }
    insert(&model.AuditConn{PeerId: "desk", ConnId: 1}, "created_at")
    insert(&model.AuditFile{PeerId: "desk", Path: "/old"}, "created_at")
    insert(&model.AuditAlarm{PeerId: "desk", Typ: 5, Info: "{}"}, "created_at")
    insert(&model.LoginLog{Client: "app"}, "created_at")
    insert(&model.AuditOperation{Method: "POST", Path: "/old"}, "created_at")
    insert(&model.AuditOperation{Method: "POST", Path: "/recent"}, "")
    insert(&model.SystemSettingHistory{SettingKey: "app.register", ChangedAt: past}, "")
    insert(&model.SystemSettingHistory{SettingKey: "app.web-client", ChangedAt: now}, "")

    for _, status := range []string{"completed", "expired", "pending", "delivered"} {
        insert(&model.DeviceDisconnectRequest{
            DeviceId: "desk", ConnId: 15, RequestedBy: 1, Status: status,
            ExpiresAt: past.Unix(),
        }, "updated_at")
    }
    for _, status := range []string{"sent", "failed", "pending", "sending", "retry", "paused"} {
        push := &model.PushDelivery{
            ChannelId: 1, EventType: "test", EventKey: status, Status: status,
            SentAt: past.Unix(),
        }
        insert(push, "updated_at")
        notification := &model.NotificationDelivery{
            EventType: "test", EventKey: status, Recipient: status+"@example.test",
            Status: status, SentAt: past.Unix(),
        }
        insert(notification, "updated_at")
    }
    for _, status := range []string{"failed", "pending", "retry"} {
        insert(&model.PushEventOutbox{Payload: "{}", Status: status}, "updated_at")
        insert(&model.NotificationEventOutbox{Event: "test", Reference: status, Status: status}, "updated_at")
    }

    request := LogCleanupRequest{Kind: "all", Mode: "older", Days: 30}
    preview, err := PreviewLogCleanup(request, now)
    if err != nil { t.Fatal(err) }
    want := map[string]int64{
        "connection": 1, "file": 1, "alarm": 1, "login": 1,
        "operation": 1, "disconnect": 2, "settings_history": 1,
        "push_sent": 1, "push_failed": 1,
        "notification_sent": 1, "notification_failed": 1,
        "push_outbox_failed": 1, "notification_outbox_failed": 1,
    }
    if len(preview) != len(want)+1 { t.Fatalf("expected %d categories; got %d",len(want)+1,len(preview)) }
    var total int64
    for _, row := range preview {
        if row.Kind=="server_log" { continue }
        if row.Eligible != want[row.Kind] {
            t.Errorf("%s preview: got %d, want %d", row.Kind, row.Eligible, want[row.Kind])
        }
        total += row.Eligible
    }
    if total != 14 { t.Fatalf("expected 14 eligible records, got %d", total) }
    if _, err := RunLogCleanup(request, now); err == nil { t.Fatal("deleting without confirmation was allowed") }
    if _, err := PreviewLogCleanup(LogCleanupRequest{Kind:"unknown",Mode:"all"},now); err == nil { t.Fatal("unsupported category accepted") }
    if _, err := PreviewLogCleanup(LogCleanupRequest{Kind:"all",Mode:"older",Days:3651},now); err == nil { t.Fatal("invalid days accepted") }

    request.Confirmation = "DELETE_LOGS"
    result, err := RunLogCleanup(request, now)
    if err != nil { t.Fatal(err) }
    var deleted int64
    for _, item := range result { deleted += item.Deleted }
    if deleted != total { t.Fatalf("deleted %d instead of %d", deleted, total) }

    for _, check := range []struct{ table interface{}; total int64 }{
        {&model.AuditConn{}, 0}, {&model.AuditFile{}, 0},
        {&model.AuditAlarm{}, 0}, {&model.LoginLog{}, 0},
        {&model.AuditOperation{}, 1}, {&model.SystemSettingHistory{}, 1},
        {&model.DeviceDisconnectRequest{}, 2},
        {&model.PushDelivery{}, 4}, {&model.NotificationDelivery{}, 4},
        {&model.PushEventOutbox{}, 2}, {&model.NotificationEventOutbox{}, 2},
    } {
        var count int64
        if err := DB.Model(check.table).Count(&count).Error; err != nil { t.Fatal(err) }
        if count != check.total { t.Fatalf("%T: expected %d remaining, got %d",check.table,check.total,count) }
    }

    // The unrestricted mode still preserves active work, including expired
    // requests that have not yet transitioned into a terminal state.
    all := LogCleanupRequest{Kind:"all",Mode:"all",Confirmation:"DELETE_LOGS"}
    if _, err := RunLogCleanup(all, now); err != nil { t.Fatal(err) }
    var count int64
    if err := DB.Model(&model.DeviceDisconnectRequest{}).Count(&count).Error;err!=nil||count!=2 {
        t.Fatalf("active disconnect requests were deleted: %d %v", count,err)
    }
    if err := DB.Model(&model.PushDelivery{}).Count(&count).Error;err!=nil||count!=4 {
        t.Fatalf("unfinished push deliveries were deleted: %d %v",count,err)
    }
    if err := DB.Model(&model.NotificationEventOutbox{}).Count(&count).Error;err!=nil||count!=2 {
        t.Fatalf("unfinished administrative notifications were deleted: %d %v",count,err)
    }
}

func TestAdditionalLogRetentionIsSavedAndApplied(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.AuditRetentionPolicy{}, &model.DeviceDisconnectRequest{},
        &model.SystemSettingHistory{}, &model.PushEventOutbox{},
        &model.NotificationEventOutbox{}); err != nil { t.Fatal(err) }
    now := time.Now()
    past := now.Add(-60*24*time.Hour)
    entries := []interface{}{
        &model.DeviceDisconnectRequest{DeviceId:"desk",ConnId:1,Status:"completed"},
        &model.SystemSettingHistory{SettingKey:"runtime.test",ChangedAt:past},
        &model.PushEventOutbox{Status:"failed",Payload:"{}"},
        &model.NotificationEventOutbox{Status:"failed",Event:"test",Reference:"old"},
    }
    for _, entry := range entries {
        if err := DB.Create(entry).Error;err!=nil{t.Fatal(err)}
        switch entry.(type) {
        case *model.SystemSettingHistory:
        default:
            if err:=DB.Model(entry).UpdateColumn("updated_at",past).Error;err!=nil{t.Fatal(err)}
        }
    }
    // Zero is an explicit unlimited-retention setting.
    if err := PruneAdditionalLogRecords(now);err!=nil{t.Fatal(err)}
    for _, entry := range entries {
        var total int64
        if err:=DB.Model(entry).Count(&total).Error;err!=nil||total!=1 {
            t.Fatalf("default retention removed %T: %d %v",entry,total,err)
        }
    }
    policy := &model.AuditRetentionPolicy{
        DisconnectDays:30,SettingsHistoryDays:30,
        PushOutboxFailedDays:30,NotificationOutboxFailedDays:30,
    }
    if err := SaveAuditRetention(policy);err!=nil{t.Fatal(err)}
    saved,err := GetAuditRetention()
    if err!=nil || saved.DisconnectDays!=30 || saved.SettingsHistoryDays!=30 ||
        saved.PushOutboxFailedDays!=30 || saved.NotificationOutboxFailedDays!=30 {
        t.Fatalf("additional retention was not saved: %+v %v",saved,err)
    }
    if err := PruneAdditionalLogRecords(now);err!=nil{t.Fatal(err)}
    for _, entry := range entries {
        var total int64
        if err:=DB.Model(entry).Count(&total).Error;err!=nil||total!=0 {
            t.Fatalf("scheduled cleanup retained %T: %d %v",entry,total,err)
        }
    }
    policy.DisconnectDays=3651
    if err := SaveAuditRetention(policy);err==nil { t.Fatal("invalid extended retention was accepted") }
}

func TestManualCleanupProcessesLargeLogTablesInBatches(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.AuditRetentionPolicy{},&model.LoginLog{});err!=nil{t.Fatal(err)}
    now:=time.Now()
    old:=now.Add(-60*24*time.Hour)
    rows:=make([]model.LoginLog,1105)
    for i:=range rows{rows[i].Client="webadmin"}
    if err:=DB.CreateInBatches(rows,100).Error;err!=nil{t.Fatal(err)}
    if err:=DB.Model(&model.LoginLog{}).Where("id > ?",0).UpdateColumn("created_at",old).Error;err!=nil{t.Fatal(err)}
    cleaned,err:=RunLogCleanup(LogCleanupRequest{Kind:"login",Mode:"older",Days:30,Confirmation:"DELETE_LOGS"},now)
    if err!=nil{t.Fatal(err)}
    if len(cleaned)!=1||cleaned[0].Deleted!=int64(len(rows)){
        t.Fatalf("batch cleanup removed wrong number of records: %+v",cleaned)
    }
    var total int64
    if err:=DB.Model(&model.LoginLog{}).Count(&total).Error;err!=nil||total!=0 {
        t.Fatalf("rows remained after cleanup: %d %v",total,err)
    }
}
