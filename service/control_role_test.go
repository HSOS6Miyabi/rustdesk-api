package service

import (
	"strconv"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestControlPermissionsAndAssignmentPriority(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ControlRole{},&model.ControlRoleAssignment{},
		&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	user:=users["member"]
	group:=&model.DeviceGroup{Name:"Support stations"}
	if err:=DB.Create(group).Error;err!=nil{t.Fatal(err)}
	device:=&model.Peer{Id:"control-test-desk",UserId:user.Id,GroupId:group.Id}
	if err:=DB.Create(device).Error;err!=nil{t.Fatal(err)}
	groupRole:=&model.ControlRole{Name:"Team defaults",Keyboard:2,File:1,USB:2}
	userRole:=&model.ControlRole{Name:"Member defaults",Keyboard:1,Clipboard:2}
	deviceRole:=&model.ControlRole{Name:"Restricted desk",Keyboard:1,Terminal:1,PrivacyMode:2}
	for _,role:=range []*model.ControlRole{groupRole,userRole,deviceRole} {
		if err:=SaveControlRole(role);err!=nil{t.Fatal(err)}
	}
	groupBits,err:=groupRole.PermissionBits()
	if err!=nil{t.Fatal(err)}
	if groupBits!=(uint64(2)<<(2*0))|(uint64(1)<<(2*3))|(uint64(2)<<(2*13)){
		t.Fatalf("control permissions bits do not follow the official protocol: %d",groupBits)
	}
	assignment:=func(targetType,targetId string,roleId uint)*model.ControlRoleAssignment{
		t.Helper()
		item:=&model.ControlRoleAssignment{
			TargetType:targetType,TargetId:targetId,RoleId:roleId,
		}
		if err:=SaveControlRoleAssignment(item);err!=nil{t.Fatal(err)}
		return item
	}
	groupAssignment:=assignment("device_group",strconv.FormatUint(uint64(group.Id),10),groupRole.Id)
	check:=func(wantRole *model.ControlRole,wantSource string){
		t.Helper()
		bits,assigned,source,err:=EffectiveControlPermissions(device.Id)
		expected,err2:=wantRole.PermissionBits()
		if err!=nil||err2!=nil||!assigned||source!=wantSource||bits!=expected{
			t.Fatalf("want %s (%d) from %s, got %d %s %v",wantRole.Name,expected,wantSource,bits,source,err)
		}
	}
	check(groupRole,"device_group")
	userAssignment:=assignment("user",strconv.FormatUint(uint64(user.Id),10),userRole.Id)
	check(userRole,"user")
	directAssignment:=assignment("device",device.Id,deviceRole.Id)
	check(deviceRole,"device")
	if err:=DeleteControlRoleAssignment(directAssignment.Id);err!=nil{t.Fatal(err)}
	check(userRole,"user")
	if err:=DeleteControlRoleAssignment(userAssignment.Id);err!=nil{t.Fatal(err)}
	check(groupRole,"device_group")
	groupRole.File=0
	groupRole.USB=0
	if err:=SaveControlRole(groupRole);err!=nil{t.Fatal(err)}
	check(groupRole,"device_group")
	if err:=DeleteControlRole(groupRole.Id);err!=nil{t.Fatal(err)}
	bits,assigned,source,err:=EffectiveControlPermissions(device.Id)
	if err!=nil||assigned||bits!=0||source!=""{
		t.Fatalf("deleted role still controls device: %d %v %s %v",bits,assigned,source,err)
	}
	var count int64
	if err:=DB.Model(&model.ControlRoleAssignment{}).
		Where("id = ?",groupAssignment.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted role retained assignment: %d %v",count,err)
	}
	if err:=SaveControlRole(&model.ControlRole{Name:"Invalid",USB:3});err==nil{
		t.Fatal("invalid permission state accepted")
	}
	for _,item:=range []*model.ControlRoleAssignment{
		{TargetType:"device",TargetId:"not registered",RoleId:deviceRole.Id},
		{TargetType:"user",TargetId:"999999",RoleId:deviceRole.Id},
		{TargetType:"device_group",TargetId:"999999",RoleId:deviceRole.Id},
		{TargetType:"unknown",TargetId:device.Id,RoleId:deviceRole.Id},
	}{
		if err:=SaveControlRoleAssignment(item);err==nil{
			t.Fatalf("invalid role assignment accepted: %+v",item)
		}
	}
}

func TestControlRoleAssignmentUpdatesAndGroupDeletion(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ControlRole{},&model.ControlRoleAssignment{},
		&model.DeviceGroup{});err!=nil{t.Fatal(err)}
	seedGroups(t)
	role1:=&model.ControlRole{Name:"First role",File:1}
	role2:=&model.ControlRole{Name:"Second role",File:2}
	if err:=SaveControlRole(role1);err!=nil{t.Fatal(err)}
	if err:=SaveControlRole(role2);err!=nil{t.Fatal(err)}
	group:=&model.DeviceGroup{Name:"Operations"}
	if err:=DB.Create(group).Error;err!=nil{t.Fatal(err)}
	for _,role:=range []*model.ControlRole{role1,role2}{
		if err:=SaveControlRoleAssignment(&model.ControlRoleAssignment{
			TargetType:"device_group",TargetId:strconv.FormatUint(uint64(group.Id),10),
			RoleId:role.Id,
		});err!=nil{t.Fatal(err)}
	}
	list,total,err:=ListControlRoleAssignments(1,20)
	if err!=nil||total!=1||len(list)!=1||list[0].RoleId!=role2.Id{
		t.Fatalf("reassignment did not replace previous role: %d %+v %v",total,list,err)
	}
	if err:=AllService.GroupService.DeviceGroupDelete(group);err!=nil{t.Fatal(err)}
	list,total,err=ListControlRoleAssignments(1,20)
	if err!=nil||total!=0{
		t.Fatalf("deleted group retained control role assignment: %v %v %v",list,total,err)
	}
}
