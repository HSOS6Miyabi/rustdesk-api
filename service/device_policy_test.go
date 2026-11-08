package service

import (
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestDevicePolicyValidationAndAssignment(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	owner,other,admin:=users["member"],users["other"],users["admin"]
	device:=&model.Peer{Id:"desk-test-001",UserId:owner.Id,Hostname:"Test device"}
	if err:=DB.Create(device).Error;err!=nil{t.Fatal(err)}

	policy:=&model.DevicePolicy{
		Name:" Office computers ",
		IdWhitelist:" 123456, 123456 \n 987* , host-01 ",
		IpWhitelist:" 192.0.2.7,192.0.2.0/24 , 192.0.2.7 ",
		EnableWebRTC:true,
		TrustDurationDays:30,
	}
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if policy.Id==0 {t.Fatal("policy has no ID")}
	if policy.Name!="Office computers" || policy.IdWhitelist!="123456,987*,host-01" ||
		policy.IpWhitelist!="192.0.2.7,192.0.2.0/24"{
		t.Fatalf("policy normalization mismatch: %+v",policy)
	}
	for _,invalid:=range []*model.DevicePolicy{
		{Name:"",EnableWebRTC:true},
		{Name:"invalid",IdWhitelist:"x; DROP TABLE users"},
		{Name:"invalid",IpWhitelist:"127.0.0.1:3000"},
		{Name:"invalid",TrustDurationDays:3650},
	}{
		if err:=SaveDevicePolicy(invalid);err==nil{t.Fatalf("invalid policy was accepted: %+v",invalid)}
	}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:"desk-test-001",PolicyId:policy.Id,
	});err!=nil{t.Fatal(err)}
	options,assigned,err:=EffectiveDevicePolicy(device.Id,owner.Id,false)
	if err!=nil||!assigned{t.Fatalf("owner cannot read policy: %v %v",options,err)}
	if options["id-whitelist"]!="123456,987*,host-01"||
		options["whitelist"]!="192.0.2.7,192.0.2.0/24"||
		options["enable-webrtc"]!="Y"||
		options["trusted-device-expiration-days"]!="30"{
		t.Fatalf("RustDesk policy keys were not preserved: %v",options)
	}
	if _,_,err:=EffectiveDevicePolicy(device.Id,other.Id,false);err==nil{
		t.Fatal("unrelated user could read managed device policy")
	}
	if _,adminAssigned,err:=EffectiveDevicePolicy(device.Id,admin.Id,true);
		err!=nil||!adminAssigned{t.Fatalf("administrator cannot inspect device policy: %v",err)}
	if _,_,err:=EffectiveDevicePolicy("absent-device",admin.Id,true);err==nil{
		t.Fatal("unknown device policy lookup succeeded")
	}

	second:=&model.DevicePolicy{Name:"Another office",EnableWebRTC:false,TrustDurationDays:7}
	if err:=SaveDevicePolicy(second);err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:device.Id,PolicyId:second.Id,
	});err!=nil{t.Fatal(err)}
	var assignmentCount int64
	if err:=DB.Model(&model.DevicePolicyAssignment{}).Count(&assignmentCount).Error;
		err!=nil||assignmentCount!=1{t.Fatalf("policy reassignment duplicated records: %d %v",assignmentCount,err)}
	options,assigned,err=EffectiveDevicePolicy(device.Id,owner.Id,false)
	if err!=nil||!assigned||options["enable-webrtc"]!="N"||
		options["trusted-device-expiration-days"]!="7" {
		t.Fatalf("policy reassignment was not applied: %v %v",options,err)
	}
	second.EnableWebRTC=true
	second.TrustDurationDays=90
	if err:=SaveDevicePolicy(second);err!=nil{t.Fatal(err)}
	options,_,err=EffectiveDevicePolicy(device.Id,owner.Id,false)
	if err!=nil||options["enable-webrtc"]!="Y"||
		options["trusted-device-expiration-days"]!="90"{
		t.Fatalf("device policy update was not applied: %v %v",options,err)
	}
	if err:=DeleteDevicePolicy(second.Id);err!=nil{t.Fatal(err)}
	options,assigned,err=EffectiveDevicePolicy(device.Id,owner.Id,false)
	if err!=nil||assigned||len(options)!=0{
		t.Fatalf("deleted policy is still assigned: %v %v %v",options,assigned,err)
	}
	if err:=DeleteDevicePolicy(0);err==nil{t.Fatal("zero policy ID should be rejected")}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:"unknown-device",PolicyId:policy.Id,
	});err==nil{t.Fatal("assignment to unregistered device was accepted")}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:device.Id,PolicyId:second.Id,
	});err==nil{t.Fatal("assignment to deleted policy was accepted")}
}

func TestDevicePolicyLimitsAndIndependentAssignments(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	policy:=&model.DevicePolicy{
		Name:"Exam sessions",IdWhitelist:strings.Repeat("A",128),
		IpWhitelist:"2001:db8::1/64",TrustDurationDays:1,
	}
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if policy.IpWhitelist!="2001:db8::/64"{
		t.Fatalf("CIDR normalization: %s",policy.IpWhitelist)
	}
	if err:=DB.Create(&model.Peer{Id:"peer-user-1",UserId:users["member"].Id}).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.Peer{Id:"peer-user-2",UserId:users["other"].Id}).Error;err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{DeviceId:"peer-user-1",PolicyId:policy.Id});err!=nil{t.Fatal(err)}
	options,assigned,err:=EffectiveDevicePolicy("peer-user-2",users["other"].Id,false)
	if err!=nil||assigned||len(options)!=0{
		t.Fatalf("unassigned device got another device's policy: %v %v",options,err)
	}
}
