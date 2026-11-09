package service

import (
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestClientSecurityPolicyOptionsApplyAndClear(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DevicePolicy{},&model.DevicePolicyAssignment{});err!=nil{t.Fatal(err)}
	owner:=seedGroups(t)["member"]
	peer:=&model.Peer{Id:"security-policy-pc",UserId:owner.Id}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	for _,invalid:=range []*model.DevicePolicy{
		{Name:"Invalid access",AccessMode:"superuser"},
		{Name:"Invalid keyboard",EnableKeyboard:"true"},
		{Name:"Invalid clipboard",EnableClipboard:"on"},
		{Name:"Invalid file permissions",EnableFileTransfer:"YES"},
		{Name:"Invalid approval",ApprovalMode:"without-password"},
	} {
		if err:=SaveDevicePolicy(invalid);err==nil {t.Fatalf("invalid security setting accepted: %+v",invalid)}
	}
	plain:=&model.DevicePolicy{Name:"Legacy baseline"}
	if err:=SaveDevicePolicy(plain);err!=nil{t.Fatal(err)}
	if plain.ManagedClientKeys!=""{t.Fatal("existing policy suddenly managed additional settings")}
	security:=&model.DevicePolicy{
		Name:"Restricted support",
		AccessMode:"view",EnableKeyboard:"N",EnableClipboard:"N",
		EnableFileTransfer:"N",ApprovalMode:"password-click",
	}
	if err:=SaveDevicePolicy(security);err!=nil{t.Fatal(err)}
	if !strings.Contains(security.ManagedClientKeys,"access-mode") {
		t.Fatalf("policy did not track centrally-managed keys: %s",security.ManagedClientKeys)
	}
	if err:=SaveDevicePolicyAssignment(&model.DevicePolicyAssignment{
		DeviceId:peer.Id,PolicyId:security.Id,
	});err!=nil{t.Fatal(err)}
	check:=func(version int64)(map[string]string,int64){
		t.Helper()
		rsp,err:=HeartbeatDeviceStrategy(peer,version)
		if err!=nil{t.Fatal(err)}
		options:=rsp["strategy"].(map[string]interface{})["config_options"].(map[string]string)
		return options,rsp["modified_at"].(int64)
	}
	first,version:=check(0)
	for key,want:=range map[string]string{
		"access-mode":"view","enable-keyboard":"N","enable-clipboard":"N",
		"enable-file-transfer":"N","approve-mode":"password-click",
	}{
		if first[key]!=want{t.Fatalf("%s should be %s; received %v",key,want,first)}
	}
	security.EnableKeyboard=""
	security.AccessMode=""
	if err:=SaveDevicePolicy(security);err!=nil{t.Fatal(err)}
	changed,revision:=check(version)
	if revision==version{t.Fatal("policy revision did not change")}
	if value,exists:=changed["access-mode"];!exists||value!=""{
		t.Fatalf("clearing managed access-mode was not transmitted: %+v",changed)
	}
	if value,exists:=changed["enable-keyboard"];!exists||value!=""{
		t.Fatalf("clearing managed keyboard was not transmitted: %+v",changed)
	}
	if changed["enable-file-transfer"]!="N"{t.Fatalf("remaining settings were lost: %+v",changed)}
	if err:=DeleteDevicePolicy(security.Id);err!=nil{t.Fatal(err)}
	removed,_:=check(revision)
	if _,exists:=removed["approve-mode"];!exists||removed["approve-mode"]!="" {
		t.Fatalf("removal did not clear the strategy: %+v",removed)
	}
}
