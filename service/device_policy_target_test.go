package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestDevicePolicyAssignmentPriorityAndFallback(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{}, &model.DevicePolicyAssignment{},
		&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	group:=&model.DeviceGroup{Name:"Office"}
	if err:=DB.Create(group).Error;err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"office-computer-001", UserId:users["member"].Id, GroupId:group.Id, Uuid:"identity-001"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	newPolicy:=func(name, allowed string)*model.DevicePolicy{
		t.Helper()
		p:=&model.DevicePolicy{Name:name,IdWhitelist:allowed}
		if err:=SaveDevicePolicy(p);err!=nil{t.Fatal(err)}
		return p
	}
	groupPolicy:=newPolicy("Group computers","group-allowed")
	userPolicy:=newPolicy("Member computers","user-allowed")
	devicePolicy:=newPolicy("Special computer","device-allowed")
	if err:=SaveDevicePolicyGroupAssignment(&model.DevicePolicyGroupAssignment{
		GroupId:group.Id,PolicyId:groupPolicy.Id});err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyUserAssignment(&model.DevicePolicyUserAssignment{
		UserId:peer.UserId,PolicyId:userPolicy.Id});err!=nil{t.Fatal(err)}
	verify:=func(source,allowed string, previous int64)int64{
		t.Helper()
		options,assigned,actualSource,_,err:=EffectiveDevicePolicyDetails(peer.Id,peer.UserId,false)
		if err!=nil||!assigned||actualSource!=source||options["id-whitelist"]!=allowed{
			t.Fatalf("expected %s policy %s, got %s %v %v",source,allowed,actualSource,options,err)
		}
		out,err:=HeartbeatDeviceStrategy(peer,previous)
		if err!=nil{t.Fatal(err)}
		strategy,ok:=out["strategy"].(map[string]interface{})
		if !ok{t.Fatalf("no client strategy for %s: %v",source,out)}
		opts,ok:=strategy["config_options"].(map[string]string)
		if !ok||opts["id-whitelist"]!=allowed{
			t.Fatalf("invalid %s strategy: %v",source,strategy)
		}
		revision,ok:=out["modified_at"].(int64)
		if !ok||revision<=0||revision==previous{
			t.Fatalf("invalid policy transition marker for %s: %v",source,out)
		}
		if stable,err:=HeartbeatDeviceStrategy(peer,revision);err!=nil||len(stable)!=0{
			t.Fatalf("identical revision caused redundant update: %v %v",stable,err)
		}
		return revision
	}
	revision:=verify("user","user-allowed",0)
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:peer.Id, PolicyId:devicePolicy.Id});err!=nil{t.Fatal(err)}
	revision=verify("device","device-allowed",revision)
	devicePolicy.Status=2
	if err:=SaveDevicePolicy(devicePolicy);err!=nil{t.Fatal(err)}
	revision=verify("user","user-allowed",revision)
	var userAssignment model.DevicePolicyUserAssignment
	if err:=DB.Where("user_id = ?",peer.UserId).First(&userAssignment).Error;err!=nil{t.Fatal(err)}
	if err:=DeleteDevicePolicyUserAssignment(userAssignment.Id);err!=nil{t.Fatal(err)}
	revision=verify("device_group","group-allowed",revision)
	groupPolicy.Status=2
	if err:=SaveDevicePolicy(groupPolicy);err!=nil{t.Fatal(err)}
	cleared,err:=HeartbeatDeviceStrategy(peer,revision)
	if err!=nil{t.Fatal(err)}
	if cleared["modified_at"]!=int64(0){
		t.Fatalf("disabled final policy must clear managed settings: %v",cleared)
	}
	options,assigned,_,_,err:=EffectiveDevicePolicyDetails(peer.Id,peer.UserId,false)
	if err!=nil||assigned||len(options)!=0{
		t.Fatalf("disabled policies must not apply: %v %v %v",options,assigned,err)
	}
	groupPolicy.Status=1
	if err:=SaveDevicePolicy(groupPolicy);err!=nil{t.Fatal(err)}
	verify("device_group","group-allowed",0)
	if _,_,_,_,err:=EffectiveDevicePolicyDetails(peer.Id,users["other"].Id,false);err==nil{
		t.Fatal("unrelated user could inspect another device's strategy")
	}
}

func TestDevicePolicyTargetReassignmentAndCleanup(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{},
		&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	group:=&model.DeviceGroup{Name:"Warehouse"}
	if err:=DB.Create(group).Error;err!=nil{t.Fatal(err)}
	first:=&model.DevicePolicy{Name:"First"}
	second:=&model.DevicePolicy{Name:"Second"}
	for _,p:=range []*model.DevicePolicy{first,second}{
		if err:=SaveDevicePolicy(p);err!=nil{t.Fatal(err)}
	}
	for _,p:=range []*model.DevicePolicyUserAssignment{
		{UserId:users["member"].Id,PolicyId:first.Id},
		{UserId:users["member"].Id,PolicyId:second.Id},
	}{
		if err:=SaveDevicePolicyUserAssignment(p);err!=nil{t.Fatal(err)}
	}
	for _,p:=range []*model.DevicePolicyGroupAssignment{
		{GroupId:group.Id,PolicyId:first.Id},
		{GroupId:group.Id,PolicyId:second.Id},
	}{
		if err:=SaveDevicePolicyGroupAssignment(p);err!=nil{t.Fatal(err)}
	}
	for _,table:=range []interface{}{&model.DevicePolicyUserAssignment{},&model.DevicePolicyGroupAssignment{}}{
		var total int64
		if err:=DB.Model(table).Count(&total).Error;err!=nil||total!=1{
			t.Fatalf("reassignment should update one row: %d %v",total,err)
		}
	}
	if err:=DeleteDevicePolicy(second.Id);err!=nil{t.Fatal(err)}
	for _,table:=range []interface{}{&model.DevicePolicyUserAssignment{},&model.DevicePolicyGroupAssignment{}}{
		var total int64
		if err:=DB.Model(table).Count(&total).Error;err!=nil||total!=0{
			t.Fatalf("deleting policy must remove its target assignments: %d %v",total,err)
		}
	}
	if err:=SaveDevicePolicyUserAssignment(&model.DevicePolicyUserAssignment{
		UserId:999999,PolicyId:first.Id});err==nil{t.Fatal("unknown account accepted")}
	if err:=SaveDevicePolicyGroupAssignment(&model.DevicePolicyGroupAssignment{
		GroupId:999999,PolicyId:first.Id});err==nil{t.Fatal("unknown group accepted")}
	if err:=SaveDevicePolicyGroupAssignment(&model.DevicePolicyGroupAssignment{
		GroupId:group.Id,PolicyId:first.Id});err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.Peer{Id:"group-member-pc",GroupId:group.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=AllService.GroupService.DeviceGroupDelete(group);err!=nil{t.Fatal(err)}
	var peer model.Peer
	if err:=DB.Where("id = ?","group-member-pc").First(&peer).Error;err!=nil||peer.GroupId!=0{
		t.Fatalf("removed group is still assigned to device: %v %v",peer,err)
	}
	var count int64
	if err:=DB.Model(&model.DevicePolicyGroupAssignment{}).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted group still has policy assignment: %d %v",count,err)
	}
}
