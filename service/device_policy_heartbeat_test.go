package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestHeartbeatAppliesAndUpdatesDevicePolicies(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{});err!=nil{t.Fatal(err)}
	owner:=seedGroups(t)["member"]
	peer:=&model.Peer{Id:"managed-001",Uuid:"registered-device-uuid",UserId:owner.Id}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	policy:=&model.DevicePolicy{Name:"Managed devices",IdWhitelist:"abc123",IpWhitelist:"192.0.2.0/24",TrustDurationDays:30}
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{DeviceId:peer.Id,PolicyId:policy.Id});err!=nil{t.Fatal(err)}
	check:=func(modifiedAt int64)map[string]interface{}{
		t.Helper()
		resp,err:=HeartbeatDeviceStrategy(peer,modifiedAt)
		if err!=nil{t.Fatal(err)}
		return resp
	}
	first:=check(0)
	if first["modified_at"]==nil{t.Fatal("assigned policy has no change marker")}
	revision,ok:=first["modified_at"].(int64)
	if !ok||revision<=0{t.Fatalf("invalid policy revision: %v",first)}
	strategy:=first["strategy"].(map[string]interface{})
	options:=strategy["config_options"].(map[string]string)
	if options["id-whitelist"]!="abc123"||options["whitelist"]!="192.0.2.0/24"||
		options["trusted-device-expiration-days"]!="30"{
		t.Fatalf("client strategy missing expected settings: %v",options)
	}
	if identical:=check(revision);len(identical)>0{
		t.Fatalf("unchanged policy was sent repeatedly: %v",identical)
	}

	policy.IdWhitelist="xyz987"
	if err:=SaveDevicePolicy(policy);err!=nil{t.Fatal(err)}
	changed:=check(revision)
	nextVersion:=changed["modified_at"].(int64)
	if nextVersion==revision||nextVersion<=0{t.Fatal("policy edit did not update revision")}
	if got:=changed["strategy"].(map[string]interface{})["config_options"].(map[string]string)["id-whitelist"];got!="xyz987"{
		t.Fatalf("updated ID whitelist was not returned: %s",got)
	}

	if err:=DeleteDevicePolicy(policy.Id);err!=nil{t.Fatal(err)}
	removed:=check(nextVersion)
	if removed["modified_at"]!=int64(0){t.Fatalf("removing assignment did not reset version: %v",removed)}
	cleared:=removed["strategy"].(map[string]interface{})["config_options"].(map[string]string)
	for _,name:=range []string{"id-whitelist","whitelist","trusted-device-expiration-days"}{
		if cleared[name]!=""{t.Fatalf("previous policy %s was not cleared",name)}
	}
	if fresh:=check(0);len(fresh)>0{t.Fatalf("unassigned device received a policy: %v",fresh)}
}

func TestHeartbeatWithoutAssignmentKeepsLegacyBehavior(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{});err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"unassigned",Uuid:"registered"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	rsp,err:=HeartbeatDeviceStrategy(peer,0)
	if err!=nil{t.Fatal(err)}
	if len(rsp)!=0{t.Fatalf("legacy heartbeat must remain unchanged: %v",rsp)}
	rsp,err=HeartbeatDeviceStrategy(nil,0)
	if err!=nil||len(rsp)!=0{t.Fatalf("missing device should get no strategy: %v %v",rsp,err)}
}
