package service

import (
	"testing"
	"strconv"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestConnectionAccessRulesForSmallTeams(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AccessRule{},&model.UserToken{},&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	owner,operator,outsider:=users["anchor"],users["member"],users["other"]
	deskGroup:=&model.DeviceGroup{Name:"Office computers"}
	if err:=DB.Create(deskGroup).Error;err!=nil{t.Fatal(err)}
	device:=&model.Peer{Id:"desk-123456",UserId:owner.Id,GroupId:deskGroup.Id}
	if err:=DB.Create(device).Error;err!=nil{t.Fatal(err)}
	for _,user:=range []*model.User{owner,operator,outsider}{
		if err:=DB.Create(&model.UserToken{
			UserId:user.Id,Token:user.Username+"-session",
			ExpiredAt:time.Now().Add(2*time.Hour).Unix(),
		}).Error;err!=nil{t.Fatal(err)}
	}
	check:=func(token string,expected bool){
		t.Helper()
		result,err:=AuthorizeConnection(token,device.Id)
		if err!=nil||result.Allow!=expected{
			t.Fatalf("authorization %s: %+v %v, expected allow=%v",token,result,err,expected)
		}
	}
	check(owner.Username+"-session",true)
	check(operator.Username+"-session",false)
	check(outsider.Username+"-session",false)
	check("",false)
	check("unknown",false)
	if result,err:=AuthorizeConnection(owner.Username+"-session","missing");err!=nil||result.Allow{
		t.Fatalf("unregistered device is accessible: %+v %v",result,err)
	}
	groupRule:=&model.AccessRule{
		Name:"Share office devices with support",SourceType:"group",SourceID:2,
		TargetType:"device_group",TargetID:itoaGroup(deskGroup.Id),Effect:"allow",Enabled:true,
	}
	if err:=SaveAccessRule(groupRule);err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.GroupMembership{UserId:operator.Id,GroupId:2}).Error;err!=nil{t.Fatal(err)}
	check(operator.Username+"-session",true)
	check(outsider.Username+"-session",false)
	allowEveryone:=&model.AccessRule{
		Name:"Allow team access",SourceType:"all",TargetType:"device",
		TargetID:device.Id,Effect:"allow",Enabled:true,
	}
	if err:=SaveAccessRule(allowEveryone);err!=nil{t.Fatal(err)}
	check(outsider.Username+"-session",true)
	explicitDeny:=&model.AccessRule{
		Name:"Block this member",SourceType:"user",SourceID:operator.Id,
		TargetType:"device",TargetID:device.Id,Effect:"deny",Enabled:true,
	}
	if err:=SaveAccessRule(explicitDeny);err!=nil{t.Fatal(err)}
	check(operator.Username+"-session",false)
	check(owner.Username+"-session",true)
	explicitDeny.Enabled=false
	if err:=SaveAccessRule(explicitDeny);err!=nil{t.Fatal(err)}
	check(operator.Username+"-session",true)
	if err:=DB.Where("user_id = ?",outsider.Id).Delete(&model.UserToken{}).Error;err!=nil{t.Fatal(err)}
	check(outsider.Username+"-session",false)
	if err:=DB.Where("user_id = ?",operator.Id).Delete(&model.GroupMembership{}).Error;err!=nil{t.Fatal(err)}
	if err:=DeleteAccessRule(allowEveryone.Id);err!=nil{t.Fatal(err)}
	check(operator.Username+"-session",false)
	check(owner.Username+"-session",true)
}

func TestConnectionAccessRulesValidateReferences(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AccessRule{},&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	for _,r:=range []*model.AccessRule{
		{Name:"Invalid",SourceType:"all",TargetType:"all",Effect:"grant",Enabled:true},
		{Name:"Unknown user",SourceType:"user",SourceID:9999,TargetType:"all",Effect:"allow",Enabled:true},
		{Name:"Unknown group",SourceType:"group",SourceID:9999,TargetType:"all",Effect:"allow",Enabled:true},
		{Name:"Unknown device",SourceType:"all",TargetType:"device",TargetID:"unknown",Effect:"allow",Enabled:true},
		{Name:"Unknown device group",SourceType:"all",TargetType:"device_group",TargetID:"90001",Effect:"allow",Enabled:true},
		{Name:"Unknown target user",SourceType:"all",TargetType:"user",TargetID:"90001",Effect:"allow",Enabled:true},
		{Name:"Invalid target",SourceType:"all",TargetType:"unknown",Effect:"allow",Enabled:true},
	}{
		if err:=SaveAccessRule(r);err==nil{t.Fatalf("invalid access rule accepted: %+v",r)}
	}
	r:=&model.AccessRule{
		Name:"Device owner's devices",SourceType:"all",TargetType:"user",
		TargetID:itoaGroup(users["anchor"].Id),Effect:"allow",Enabled:true,
	}
	if err:=SaveAccessRule(r);err!=nil{t.Fatal(err)}
	r.Name="Updated sharing permission"
	r.Effect="deny"
	r.Enabled=false
	if err:=SaveAccessRule(r);err!=nil{t.Fatal(err)}
	var loaded model.AccessRule
	if err:=DB.First(&loaded,r.Id).Error;err!=nil{t.Fatal(err)}
	if loaded.Enabled||loaded.Effect!="deny"||loaded.Name!=r.Name{
		t.Fatalf("cannot modify stored rule: %+v",loaded)
	}
	if err:=DeleteAccessRule(r.Id);err!=nil{t.Fatal(err)}
	if err:=DeleteAccessRule(r.Id);err==nil{t.Fatal("deleted rule was deleted again")}
}
func itoaGroup(id uint)string {return strconv.FormatUint(uint64(id),10)}
