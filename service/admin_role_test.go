package service

import (
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAdministratorRolesAndScopedDirectory(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AdminRole{},&model.AdminRoleAssignment{});err!=nil{t.Fatal(err)}
	AllService.UserService=&UserService{}
	users:=seedGroups(t)
	operator:=users["member"]
	if err:=DB.Create(&model.GroupMembership{GroupId:2,UserId:users["other"].Id}).Error;err!=nil{t.Fatal(err)}
	scoped:=&model.AdminRole{
		Name:"Support directory",Permissions:" users.view,users.view ",
		ScopeType:AdminScopeGroup,ScopeId:2,Enabled:true,
	}
	if err:=SaveAdminRole(scoped);err!=nil{t.Fatal(err)}
	if scoped.Permissions!="users.view"{t.Fatalf("repeated permission was not normalized: %q",scoped.Permissions)}
	direct:=&model.AdminRole{
		Name:"Individual account",Permissions:"users.view",
		ScopeType:AdminScopeIndividual,ScopeId:users["contact"].Id,Enabled:true,
	}
	if err:=SaveAdminRole(direct);err!=nil{t.Fatal(err)}
	global:=&model.AdminRole{
		Name:"View all devices",Permissions:"devices.view",
		ScopeType:AdminScopeGlobal,Enabled:true,
	}
	if err:=SaveAdminRole(global);err!=nil{t.Fatal(err)}
	for _,role:=range []*model.AdminRole{scoped,direct,global}{
		if err:=SaveAdminRoleAssignment(&model.AdminRoleAssignment{
			UserId:operator.Id,RoleId:role.Id,
		});err!=nil{t.Fatal(err)}
	}
	if err:=SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId:operator.Id,RoleId:global.Id,
	});err!=nil{t.Fatal(err)}
	roles,total,err:=ListAdminRoleAssignments(1,30)
	if err!=nil||total!=3||len(roles)!=3{
		t.Fatalf("multiple assignment or duplicate protection failed: %d %v %v",total,roles,err)
	}
	access,err:=RoleAccessForUser(operator,"users.view")
	if err!=nil||access.Global||len(access.UserIDs)!=2||
		!ScopeCanViewUser(access,users["contact"].Id)||
		!ScopeCanViewUser(access,users["other"].Id)||
		ScopeCanViewUser(access,users["admin"].Id){
		t.Fatalf("group and individual scopes were combined incorrectly: %+v %v",access,err)
	}
	allDevices,err:=RoleAccessForUser(operator,"devices.view")
	if err!=nil||!allDevices.Global{t.Fatalf("global device permission was not granted: %+v %v",allDevices,err)}
	noWrite,err:=RoleAccessForUser(operator,"users.delete")
	if err!=nil||noWrite.Global||len(noWrite.UserIDs)>0{
		t.Fatalf("unassigned deletion privilege was granted: %+v %v",noWrite,err)
	}
	names:=AllService.UserService.RouteNames(operator)
	if !strings.Contains(strings.Join(names,","),"users.view")||
		!strings.Contains(strings.Join(names,","),"devices.view")||
		strings.Contains(strings.Join(names,","),"*"){
		t.Fatalf("login did not expose assigned management rights: %v",names)
	}
	if !strings.Contains(strings.Join(names,","),"users.view") || strings.Contains(strings.Join(names,","),"users.delete"){
		t.Fatalf("permission presentation failed: %v",names)
	}
	global.Enabled=false
	if err:=SaveAdminRole(global);err!=nil{t.Fatal(err)}
	allDevices,err=RoleAccessForUser(operator,"devices.view")
	if err!=nil||allDevices.Global||len(allDevices.UserIDs)>0{
		t.Fatalf("disabled role remained active: %+v %v",allDevices,err)
	}
	if err:=DeleteAdminRole(scoped.Id);err!=nil{t.Fatal(err)}
	access,err=RoleAccessForUser(operator,"users.view")
	if err!=nil||len(access.UserIDs)!=1||access.UserIDs[0]!=users["contact"].Id{
		t.Fatalf("deleted scope still granted access: %+v %v",access,err)
	}
	if len(AllService.UserService.RouteNames(users["admin"]))==0||
		AllService.UserService.RouteNames(users["admin"])[0]!="*"{
		t.Fatal("existing full administrator privileges changed")
	}
}

func TestAdministratorRolePermissionsAreValidated(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AdminRole{},&model.AdminRoleAssignment{});err!=nil{t.Fatal(err)}
	AllService.UserService=&UserService{}
	users:=seedGroups(t)
	for _,role:=range []*model.AdminRole{
		{Name:"Invalid action",Permissions:"devices.mystery",ScopeType:AdminScopeGlobal,Enabled:true},
		{Name:"Restricted modifications",Permissions:"users.delete",ScopeType:AdminScopeGroup,ScopeId:2,Enabled:true},
		{Name:"Missing group",Permissions:"users.view",ScopeType:AdminScopeGroup,ScopeId:99999,Enabled:true},
		{Name:"No granted permissions",Permissions:"",ScopeType:AdminScopeGlobal,Enabled:true},
		{Name:"Malformed scope",Permissions:"users.view",ScopeType:"planet",Enabled:true},
	}{
		if err:=SaveAdminRole(role);err==nil{
			t.Fatalf("invalid administrator role was accepted: %+v",role)
		}
	}
	role:=&model.AdminRole{
		Name:"Account viewer",Permissions:"users.view",
		ScopeType:AdminScopeIndividual,ScopeId:users["member"].Id,Enabled:true,
	}
	if err:=SaveAdminRole(role);err!=nil{t.Fatal(err)}
	if err:=SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId:999999,RoleId:role.Id,
	});err==nil{t.Fatal("role assigned to an unknown account")}
	if err:=SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId:users["member"].Id,RoleId:999999,
	});err==nil{t.Fatal("unknown role was assigned")}
	if err:=DeleteAdminRole(role.Id);err!=nil{t.Fatal(err)}
	if err:=DeleteAdminRole(role.Id);err==nil{t.Fatal("role deleted twice")}
}

func TestDelegatedManagementRoutesAreExplicit(t *testing.T){
	testCases:=[]struct{method,path,want string}{
		{"GET","/api/admin/user/list","users.view"},
		{"GET","/api/admin/user/detail/:id","users.view"},
		{"POST","/api/admin/user/create","users.create"},
		{"POST","/api/admin/user/changePwd","users.edit_password"},
		{"GET","/api/admin/peer/list","devices.view"},
		{"POST","/api/admin/peer/delete","devices.delete"},
		{"GET","/api/admin/control_role/list","control_roles.view"},
		{"POST","/api/admin/control_assignment/create","control_roles.edit"},
		{"GET","/api/admin/device_policy/list","strategies.view"},
		{"GET","/api/admin/audit/export","audits.view"},
		{"GET","/api/admin/admin_role/list",""},
		{"POST","/api/admin/oauth/create",""},
		{"GET","/api/admin/mail/settings","smtp.view"},
		{"GET","/api/admin/user/current",""},
		{"GET","/api/admin/unknown",""},
		{"DELETE","/api/admin/user/list",""},
	}
	for _,tc:=range testCases{
		got:=AdminPermissionForRoute(tc.method,tc.path)
		if got!=tc.want{t.Errorf("%s %s: expected %q, got %q",tc.method,tc.path,tc.want,got)}
	}
}
