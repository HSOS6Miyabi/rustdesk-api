package service

import (
    "fmt"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/gorm"
)

func TestDelegatedRolesUseBoundedDatabaseQueries(t *testing.T) {
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.AdminRole{},&model.AdminRoleAssignment{});err!=nil{t.Fatal(err)}
    AllService.UserService=&UserService{}
    users:=seedGroups(t)
    extraGroup:=&model.Group{Name:"additional",Type:model.GroupTypeShare}
    if err:=DB.Create(extraGroup).Error;err!=nil{t.Fatal(err)}
    for _,group:=range []uint{2,extraGroup.Id}{
        role:=&model.AdminRole{
            Name:fmt.Sprintf("Delegated group %d",group),Permissions:"users.view",
            ScopeType:AdminScopeGroup,ScopeId:group,Enabled:true,
        }
        if err:=DB.Create(role).Error;err!=nil{t.Fatal(err)}
        if err:=DB.Create(&model.AdminRoleAssignment{UserId:users["member"].Id,RoleId:role.Id}).Error;err!=nil{t.Fatal(err)}
    }
    if err:=DB.Create(&model.GroupMembership{GroupId:extraGroup.Id,UserId:users["other"].Id}).Error;err!=nil{t.Fatal(err)}
    count:=0
    const key="perf:delegated_roles_query_count"
    if err:=DB.Callback().Query().Before("gorm:query").Register(key,func(tx *gorm.DB){count++});err!=nil{t.Fatal(err)}
    access,err:=RoleAccessForUser(users["member"],"users.view")
    DB.Callback().Query().Remove(key)
    if err!=nil{t.Fatal(err)}
    if access.Global||!ScopeCanViewUser(access,users["other"].Id){t.Fatalf("delegated role visibility incorrect: %+v",access)}
    if count>3{t.Fatalf("role queries increased with group count: %d",count)}
}

func TestPushDeliveryReadsEachChannelOnlyOncePerBatch(t *testing.T) {
    cleanup:=setupPushTestDB(t)
    defer cleanup()
    channel:=&model.PushChannel{UserId:1,Kind:"messagepusher",Name:"paused channel",Enabled:false}
    if err:=DB.Create(channel).Error;err!=nil{t.Fatal(err)}
    now:=time.Now().Unix()
    deliveries:=make([]model.PushDelivery,20)
    for i:=range deliveries{
        deliveries[i]=model.PushDelivery{
            ChannelId:channel.Id,UserId:1,EventType:"device.registered",
            EventKey:fmt.Sprintf("batch-event-%02d",i),Status:"pending",
            NextAttemptAt:now,
        }
    }
    if err:=DB.CreateInBatches(deliveries,50).Error;err!=nil{t.Fatal(err)}
    count:=0
    const key="perf:delivery_channel_query_count"
    if err:=DB.Callback().Query().Before("gorm:query").Register(key,func(tx *gorm.DB){
        if tx.Statement.Table=="push_channels" {count++}
    });err!=nil{t.Fatal(err)}
    _,err:=ProcessPushQueue(20)
    DB.Callback().Query().Remove(key)
    if err!=nil{t.Fatal(err)}
    if count!=1{t.Fatalf("channel was reloaded %d times for 20 entries",count)}
    var paused int64
    if err:=DB.Model(&model.PushDelivery{}).Where("status = ?","paused").Count(&paused).Error;err!=nil{t.Fatal(err)}
    if paused!=20{t.Fatalf("disabled channel accepted %d entries: %d",20-paused,paused)}
}
