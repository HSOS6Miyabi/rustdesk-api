package service

import (
    "errors"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestPoisonPushEventDoesNotPreventLaterNotifications(t *testing.T) {
    cleanup := setupPushTestDB(t)
    defer cleanup()
    user:=&model.User{Username:"outbox-continue",Status:model.COMMON_STATUS_ENABLE}
    if err:=DB.Create(user).Error;err!=nil{t.Fatal(err)}
    peer:=&model.Peer{Id:"outbox-continue-peer",UserId:user.Id}
    if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
    if _,err:=SavePushChannel(user.Id,PushChannelInput{
        Kind:"messagepusher",Name:"test",Enabled:true,
        APIKey:"mpk_test",BaseURL:"https://push.example.test",
        Events:[]string{"device.registered"},
    });err!=nil{t.Fatal(err)}
    poison:=&model.PushEventOutbox{Payload:"malformed JSON"}
    if err:=DB.Create(poison).Error;err!=nil{t.Fatal(err)}
    if err:=persistPushEvent(PushEvent{
        Code:"device.registered",Reference:"valid-after-invalid",When:time.Now(),
        PeerRowID:peer.RowId,DeviceID:peer.Id,UserID:user.Id,
    });err!=nil{t.Fatal(err)}
    processed,err:=ProcessPushEventOutbox(20)
    if err==nil||processed!=1{t.Fatalf("processed %d entries, error %v",processed,err)}
    if err:=DB.First(poison,poison.Id).Error;err!=nil{t.Fatal(err)}
    if poison.Status!="failed"||poison.Attempts!=1{t.Fatalf("unexpected failed event: %+v",poison)}
    var count int64
    if err:=DB.Model(&model.PushDelivery{}).Count(&count).Error;err!=nil||count!=1{
        t.Fatalf("valid event was not queued: %d %v",count,err)
    }
    if err:=DB.Model(&model.PushEventOutbox{}).Count(&count).Error;err!=nil||count!=1{
        t.Fatalf("unexpected pending outbox count: %d %v",count,err)
    }
}

func TestSharedOutboxLeaseRejectsStaleWorkerUpdates(t *testing.T) {
    cleanup:=setupPushTestDB(t)
    defer cleanup()
    t.Setenv("RUSTDESK_API_SHARED_EVENT_WORKERS","1")
    row:=&model.PushEventOutbox{Payload:"{}"}
    if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
    ownerA,accepted,err:=claimOutboxRow(&model.PushEventOutbox{},row.Id)
    if err!=nil||!accepted||ownerA==""{t.Fatalf("first claim: %s %v %v",ownerA,accepted,err)}
    if err:=DB.Model(row).Update("lease_until",time.Now().Unix()-1).Error;err!=nil{t.Fatal(err)}
    ownerB,accepted,err:=claimOutboxRow(&model.PushEventOutbox{},row.Id)
    if err!=nil||!accepted||ownerB==ownerA{t.Fatalf("second claim: %s %v %v",ownerB,accepted,err)}
    if err:=recordOutboxFailure(&model.PushEventOutbox{},row.Id,0,errors.New("stale result"),ownerA,true);err!=nil{t.Fatal(err)}
    if err:=DB.First(row,row.Id).Error;err!=nil{t.Fatal(err)}
    if row.LeaseOwner!=ownerB||row.Status=="failed"{
        t.Fatalf("stale worker changed the newer claim: %+v",row)
    }
    if err:=deleteProcessedOutboxRow(&model.PushEventOutbox{},row.Id,ownerA);err!=nil{t.Fatal(err)}
    var remaining int64
    if err:=DB.Model(&model.PushEventOutbox{}).Where("id = ?",row.Id).Count(&remaining).Error;err!=nil||remaining!=1{
        t.Fatalf("stale worker deleted an active claim: %d %v",remaining,err)
    }
    if err:=recordOutboxFailure(&model.PushEventOutbox{},row.Id,0,errors.New("invalid event"),ownerB,true);err!=nil{t.Fatal(err)}
    if err:=DB.First(row,row.Id).Error;err!=nil{t.Fatal(err)}
    if row.Status!="failed"||row.Attempts!=1||row.LeaseOwner!=""{
        t.Fatalf("active worker was unable to report failure: %+v",row)
    }
}

func TestApplicationSettingsSaveAndVersionConflict(t *testing.T) {
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.SystemSettingOverride{},&model.SystemSettingRevision{},
        &model.SystemSettingHistory{});err!=nil{t.Fatal(err)}
    original:=&config.Config{App:config.App{Register:true}}
    if err:=LoadConfiguredSettings(original);err!=nil{t.Fatal(err)}
    old:=true
    falseValue:="false"
    change:=SettingsRevisionInput{
        DatabaseVersion:1,
        Changes:[]SettingsChange{{Key:"app.register",Enabled:true,Value:&falseValue}},
    }
    if err:=SaveSettingChanges(1,change);err!=nil{t.Fatal(err)}
    if err:=SaveSettingChanges(1,change);!errors.Is(err,ErrSettingsConflict){
        t.Fatalf("expected concurrent edit protection, got %v",err)
    }
    restarted:=&config.Config{App:config.App{Register:old}}
    if err:=LoadConfiguredSettings(restarted);err!=nil{t.Fatal(err)}
    if restarted.App.Register {t.Fatal("database override did not take precedence")}
    reset:=SettingsRevisionInput{
        DatabaseVersion:2,
        Changes:[]SettingsChange{{Key:"app.register",Enabled:false}},
    }
    if err:=ResetSettingChanges(1,reset);err!=nil{t.Fatal(err)}
    afterReset:=&config.Config{App:config.App{Register:true}}
    if err:=LoadConfiguredSettings(afterReset);err!=nil{t.Fatal(err)}
    if !afterReset.App.Register{t.Fatal("disabled override still changed the running value")}
}
