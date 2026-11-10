package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAuditRetentionAndManagementQueries(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(
		&model.AuditOperation{},&model.AuditRetentionPolicy{},
		&model.AuditConn{},&model.AuditFile{},&model.AuditAlarm{},&model.LoginLog{},
	);err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	owner:=users["member"]
	service:=&AuditService{}
	policy,err:=GetAuditRetention()
	if err!=nil{t.Fatal(err)}
	if policy.ConnectionDays!=0 || policy.OperationDays!=0 {t.Fatal("default retention must preserve all logs")}
	invalid:=&model.AuditRetentionPolicy{OperationDays:3651}
	if err:=SaveAuditRetention(invalid);err==nil{t.Fatal("overlarge retention accepted")}
	if err:=SaveAuditRetention(&model.AuditRetentionPolicy{
		ConnectionDays:30,FileDays:0,AlarmDays:7,LoginDays:90,OperationDays:14,
	});err!=nil{t.Fatal(err)}
	updated,err:=GetAuditRetention()
	if err!=nil || updated.ConnectionDays!=30||updated.FileDays!=0||updated.AlarmDays!=7 {
		t.Fatalf("retention setting was not saved: %+v %v",updated,err)
	}

	entries:=[]interface{}{
		&model.AuditConn{PeerId:"PC1",ConnId:7},
		&model.AuditFile{PeerId:"PC1",Path:"/file"},
		&model.AuditAlarm{PeerId:"PC1",Typ:10,Info:"{}",ToUserId:owner.Id},
		&model.LoginLog{UserId:owner.Id,Client:"webadmin"},
		&model.AuditOperation{UserId:owner.Id,Account:owner.Username,Method:"POST",Path:"/admin/old",Success:true},
	}
	for _,entry:=range entries{
		if err:=DB.Create(entry).Error;err!=nil{t.Fatal(err)}
	}
	old:=time.Now().Add(-120*24*time.Hour)
	for _,entry:=range entries{
		if err:=DB.Model(entry).Update("created_at",old).Error;err!=nil{t.Fatal(err)}
	}

	recent:=&model.AuditAlarm{PeerId:"PC2",Typ:0,Info:"{}",ToUserId:owner.Id}
	if err:=DB.Create(recent).Error;err!=nil{t.Fatal(err)}
	recentOperation:=&model.AuditOperation{UserId:owner.Id,Account:owner.Username,Method:"POST",Path:"/admin/new",Success:true}
	if err:=DB.Create(recentOperation).Error;err!=nil{t.Fatal(err)}

	alarms,total,err:=service.AllAlarms(1,20,"PC2",nil)
	if err!=nil||total!=1||len(alarms)!=1||alarms[0].PeerId!="PC2"{
		t.Fatalf("alarm filtering failed: %+v %d %v",alarms,total,err)
	}
	kind:=10
	alarms,total,err=service.AllAlarms(1,20,"PC1",&kind)
	if err!=nil||total!=1||len(alarms)!=1||alarms[0].Typ!=10{
		t.Fatalf("alarm type filtering failed: %+v %d %v",alarms,total,err)
	}
	operations,total,err:=service.AllOperations(1,20,owner.Id,"POST")
	if err!=nil||total!=2||len(operations)!=2{
		t.Fatalf("operation query failed: %+v %d %v",operations,total,err)
	}
	_,total,err=service.AllOperations(1,20,owner.Id,"DELETE")
	if err!=nil||total!=0{t.Fatalf("method filter incorrect: %d %v",total,err)}

	if err:=PruneOldAuditRecords(time.Now());err!=nil{t.Fatal(err)}
	for _,check:=range []struct{m interface{};expected int64}{
		{&model.AuditConn{},0},{&model.AuditFile{},1},
		{&model.AuditAlarm{},1},{&model.LoginLog{},0},{&model.AuditOperation{},1},
	}{
		var count int64
		if err:=DB.Model(check.m).Count(&count).Error;err!=nil{t.Fatal(err)}
		if count!=check.expected{t.Fatalf("retention count for %T: got %d expected %d",check.m,count,check.expected)}
	}
	if err:=SaveAuditRetention(&model.AuditRetentionPolicy{});err!=nil{t.Fatal(err)}
	if err:=PruneOldAuditRecords(time.Now());err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.AuditFile{}).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("unlimited retention removed file history: %d %v",count,err)
	}
}

func TestDeliveryHistoryCleanupPreservesUnfinishedMessages(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AuditRetentionPolicy{}, &model.PushDelivery{}, &model.NotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	initial, err := GetAuditRetention()
	if err != nil || initial.PushSentDays != 30 || initial.NotificationFailedDays != 90 {
		t.Fatalf("default notification retention: %+v %v", initial, err)
	}
	now := time.Now()
	policy := &model.AuditRetentionPolicy{
		PushSentDays:30, PushFailedDays:90, NotificationSentDays:30, NotificationFailedDays:90,
	}
	if err := SaveAuditRetention(policy); err != nil { t.Fatal(err) }
	for _, item := range []struct{ m interface{}; status string; days int }{
		{&model.PushDelivery{ChannelId:1, EventType:"test", EventKey:"expired-push-sent", Status:"sent", SentAt:now.Add(-40*24*time.Hour).Unix()}, "sent", 40},
		{&model.PushDelivery{ChannelId:1, EventType:"test", EventKey:"recent-push-sent", Status:"sent", SentAt:now.Unix()}, "sent", 0},
		{&model.PushDelivery{ChannelId:1, EventType:"test", EventKey:"expired-push-failed", Status:"failed"}, "failed", 120},
		{&model.PushDelivery{ChannelId:1, EventType:"test", EventKey:"pending-push", Status:"pending"}, "pending", 120},
		{&model.NotificationDelivery{EventType:"test", EventKey:"expired-mail", Recipient:"a@example.test", Status:"sent", SentAt:now.Add(-40*24*time.Hour).Unix()}, "sent", 40},
		{&model.NotificationDelivery{EventType:"test", EventKey:"failed-mail", Recipient:"a@example.test", Status:"failed"}, "failed", 120},
		{&model.NotificationDelivery{EventType:"test", EventKey:"paused-mail", Recipient:"a@example.test", Status:"paused"}, "paused", 120},
	} {
		if err := DB.Create(item.m).Error; err != nil { t.Fatal(err) }
		if item.days > 0 {
			if err := DB.Model(item.m).UpdateColumn("updated_at", now.Add(-time.Duration(item.days)*24*time.Hour)).Error; err != nil { t.Fatal(err) }
		}
	}
	if err := PruneOldDeliveryRecords(now); err != nil { t.Fatal(err) }
	var remainingPush []model.PushDelivery
	if err := DB.Order("id").Find(&remainingPush).Error; err != nil { t.Fatal(err) }
	if len(remainingPush) != 2 { t.Fatalf("push retention: %+v", remainingPush) }
	var remainingMail []model.NotificationDelivery
	if err := DB.Order("id").Find(&remainingMail).Error; err != nil { t.Fatal(err) }
	if len(remainingMail) != 1 || remainingMail[0].Status != "paused" {
		t.Fatalf("mail retention: %+v", remainingMail)
	}
	policy.PushSentDays = 0
	policy.PushFailedDays = 0
	policy.NotificationSentDays = 0
	policy.NotificationFailedDays = 0
	if err := SaveAuditRetention(policy); err != nil { t.Fatal(err) }
	saved, err := GetAuditRetention()
	if err != nil || saved.PushSentDays != 0 || saved.NotificationFailedDays != 0 {
		t.Fatalf("explicit unlimited retention was not saved: %+v %v", saved, err)
	}
}
