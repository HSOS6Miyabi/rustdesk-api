package service

import (
	"fmt"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func TestUnusedDevicePoliciesSkipSteadyHeartbeatQueries(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{},
		&model.DevicePolicyUserAssignment{},&model.DevicePolicyGroupAssignment{});err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"idle-policy-device"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	if _,err:=HeartbeatDeviceStrategy(peer,0);err!=nil{t.Fatal(err)}
	queries:=0
	if err:=DB.Callback().Query().Before("gorm:query").
		Register("perf:policy_read",func(*gorm.DB){queries++});err!=nil{t.Fatal(err)}
	for i:=0;i<25;i++ {
		answer,err:=HeartbeatDeviceStrategy(peer,0)
		if err!=nil||len(answer)!=0 {t.Fatalf("unexpected inactive policy: %v %v",answer,err)}
	}
	if err:=DB.Callback().Query().Remove("perf:policy_read");err!=nil{t.Fatal(err)}
	if queries!=0 {t.Fatalf("inactive policies produced %d queries",queries)}
	policy:=&model.DevicePolicy{Name:"Active policy",IdWhitelist:"allowed"}
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{DeviceId:peer.Id,PolicyId:policy.Id});err!=nil{t.Fatal(err)}
	answer,err:=HeartbeatDeviceStrategy(peer,0)
	if err!=nil||answer["strategy"]==nil {t.Fatalf("saved policy was not applied: %v %v",answer,err)}
	revision:=answer["modified_at"].(int64)
	if err:=DeleteDevicePolicy(policy.Id);err!=nil{t.Fatal(err)}
	answer,err=HeartbeatDeviceStrategy(peer,revision)
	if err!=nil||answer["modified_at"]!=int64(0) {t.Fatalf("removed policy was not cleared: %v %v",answer,err)}
}

func TestIdleDeviceSessionsAvoidDatabaseOperations(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DeviceActiveConnection{},&model.DeviceDisconnectRequest{});err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"idle-session-device"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	if _,err:=SynchronizeDeviceSessions(peer,nil);err!=nil{t.Fatal(err)}
	reads,deletes:=0,0
	if err:=DB.Callback().Query().Before("gorm:query").Register("perf:session_read",func(*gorm.DB){reads++});err!=nil{t.Fatal(err)}
	if err:=DB.Callback().Delete().Before("gorm:delete").Register("perf:session_delete",func(*gorm.DB){deletes++});err!=nil{t.Fatal(err)}
	for i:=0;i<25;i++ {
		commands,err:=SynchronizeDeviceSessions(peer,nil)
		if err!=nil||len(commands)!=0 {t.Fatalf("inactive session response: %v %v",commands,err)}
	}
	DB.Callback().Query().Remove("perf:session_read")
	DB.Callback().Delete().Remove("perf:session_delete")
	if reads!=0||deletes!=0 {t.Fatalf("idle session caused SQL queries/deletes: %d/%d",reads,deletes)}
	if _,err:=SynchronizeDeviceSessions(peer,[]int32{123});err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.DeviceActiveConnection{}).Where("device_id = ?",peer.Id).Count(&count).Error;err!=nil||count!=1 {
		t.Fatalf("active connection missing: %d %v",count,err)
	}
	if _,err:=SynchronizeDeviceSessions(peer,nil);err!=nil{t.Fatal(err)}
	if err:=DB.Model(&model.DeviceActiveConnection{}).Where("device_id = ?",peer.Id).Count(&count).Error;err!=nil||count!=0 {
		t.Fatalf("closed connection remains: %d %v",count,err)
	}
}

func TestDisabledPushSubscriptionsAvoidDeviceQueries(t *testing.T) {
	cleanup:=setupPushTestDB(t)
	defer cleanup()
	if hasPushSubscription("device.registered") {t.Fatal("inactive push marked subscribed")}
	queries:=0
	if err:=DB.Callback().Query().Before("gorm:query").Register("perf:push_read",func(*gorm.DB){queries++});err!=nil{t.Fatal(err)}
	for i:=0;i<25;i++{
		if err:=processPushEvent(PushEvent{Code:"device.registered",DeviceID:"unknown"});err!=nil{t.Fatal(err)}
	}
	DB.Callback().Query().Remove("perf:push_read")
	if queries!=0 {t.Fatalf("inactive push produced %d queries",queries)}
}

// Active policies are resolved once per device until an assignment or policy
// changes. Changing policy options invalidates the snapshot immediately.
func TestActivePolicyHeartbeatResolutionUsesCache(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{},
		&model.DevicePolicyUserAssignment{},&model.DevicePolicyGroupAssignment{});err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"cached-policy-device"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	policy:=&model.DevicePolicy{Name:"Cached",IdWhitelist:"first"}
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{DeviceId:peer.Id,PolicyId:policy.Id});err!=nil{t.Fatal(err)}
	answer,err:=HeartbeatDeviceStrategy(peer,0)
	if err!=nil||answer["strategy"]==nil{t.Fatalf("policy not initially applied: %v %v",answer,err)}
	revision:=answer["modified_at"].(int64)
	queries:=0
	if err:=DB.Callback().Query().Before("gorm:query").
		Register("perf:cached_policy_read",func(*gorm.DB){queries++});err!=nil{t.Fatal(err)}
	for i:=0;i<20;i++{
		value,err:=HeartbeatDeviceStrategy(peer,revision)
		if err!=nil||len(value)!=0{t.Fatalf("identical policy was returned: %v %v",value,err)}
	}
	if err:=DB.Callback().Query().Remove("perf:cached_policy_read");err!=nil{t.Fatal(err)}
	if queries!=0{t.Fatalf("unchanged active policy issued %d queries",queries)}
	policy.IdWhitelist="second"
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	answer,err=HeartbeatDeviceStrategy(peer,revision)
	if err!=nil||answer["modified_at"]==nil||answer["modified_at"]==revision{
		t.Fatalf("updated policy was not returned: %v %v",answer,err)
	}
}

func TestGloballyUnusedSessionsAvoidAllDeviceWrites(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DeviceActiveConnection{},&model.DeviceDisconnectRequest{});err!=nil{t.Fatal(err)}
	peers:=make([]*model.Peer,30)
	for i:=range peers{
		peers[i]=&model.Peer{Id:fmt.Sprintf("idle-device-%d",i)}
		if err:=DB.Create(peers[i]).Error;err!=nil{t.Fatal(err)}
	}
	if empty,err:=persistedSessionsEmpty();err!=nil||!empty{t.Fatalf("unexpected session state: %v %v",empty,err)}
	reads,updates,deletes:=0,0,0
	if err:=DB.Callback().Query().Before("gorm:query").Register("perf:global_session_read",func(*gorm.DB){reads++});err!=nil{t.Fatal(err)}
	if err:=DB.Callback().Update().Before("gorm:update").Register("perf:global_session_update",func(*gorm.DB){updates++});err!=nil{t.Fatal(err)}
	if err:=DB.Callback().Delete().Before("gorm:delete").Register("perf:global_session_delete",func(*gorm.DB){deletes++});err!=nil{t.Fatal(err)}
	for _,peer:=range peers {
		for i:=0;i<10;i++ {
			commands,err:=SynchronizeDeviceSessions(peer,nil)
			if err!=nil||len(commands)!=0{t.Fatalf("idle session error: %v %v",commands,err)}
		}
	}
	DB.Callback().Query().Remove("perf:global_session_read")
	DB.Callback().Update().Remove("perf:global_session_update")
	DB.Callback().Delete().Remove("perf:global_session_delete")
	if reads!=0||updates!=0||deletes!=0{
		t.Fatalf("idle sessions read/updated/deleted: %d/%d/%d",reads,updates,deletes)
	}
	if _,err:=SynchronizeDeviceSessions(peers[0],[]int32{99});err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.DeviceActiveConnection{}).Where("device_id = ?",peers[0].Id).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("active connection missing: %d %v",count,err)
	}
	if _,err:=SynchronizeDeviceSessions(peers[0],nil);err!=nil{t.Fatal(err)}
	if err:=DB.Model(&model.DeviceActiveConnection{}).Where("device_id = ?",peers[0].Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("closed connection remains: %d %v",count,err)
	}
}
