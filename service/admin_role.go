package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	AdminScopeGlobal="global"
	AdminScopeGroup="group"
	AdminScopeIndividual="individual"
)

var adminPermissionNames = []string{
	"users.view","users.create","users.delete","users.edit",
	"users.edit_password","users.force_logout","users.manage_2fa",
	"devices.view","devices.create","devices.edit_info","devices.delete","devices.transfer",
	"user_groups.view","user_groups.edit","device_groups.view",
	"device_groups.edit","audits.view","audits.edit",
	"strategies.view","strategies.edit",
	"control_roles.view","control_roles.edit",
	"address_books.view","address_books.edit",
	"api_tokens.view","api_tokens.edit","smtp.view","smtp.edit",
	"notifications.view","notifications.edit",
	"relay_nodes.view","relay_nodes.edit",
	"device_deployment.view","device_deployment.edit",
	"access_rules.view","access_rules.edit",
	"sessions.view","sessions.edit",
}

var knownAdminPermission = func()map[string]bool{
	items:=make(map[string]bool,len(adminPermissionNames))
	for _,permission:=range adminPermissionNames{items[permission]=true}
	return items
}()

func AdminPermissionCatalog()[]string {
	return append([]string{},adminPermissionNames...)
}

func ParseAdminRolePermissions(input string)(string,error){
	parts:=strings.FieldsFunc(input,func(r rune)bool{return r==','||r==';'||r=='\n'})
	used:=make(map[string]bool)
	out:=make([]string,0,len(parts))
	for _,raw:=range parts{
		p:=strings.TrimSpace(raw)
		if p==""{continue}
		if !knownAdminPermission[p]{return "",fmt.Errorf("unknown admin permission %s",p)}
		if !used[p]{used[p]=true;out=append(out,p)}
	}
	if len(out)==0{return "",errors.New("select at least one management permission")}
	sort.Strings(out)
	return strings.Join(out,","),nil
}

func roleSupportsScope(permission,scope string)bool{
	if scope==AdminScopeGlobal{return true}
	switch permission {
	case "users.view", "users.edit", "users.delete",
		"devices.view", "devices.edit_info", "devices.delete", "devices.transfer":
		return true
	default:
		return false
	}
}

func ValidateAdminRole(role *model.AdminRole)error{
	if role==nil{return errors.New("admin role details are required")}
	role.Name=strings.TrimSpace(role.Name)
	if role.Name==""||len([]rune(role.Name))>100{return errors.New("role name must contain 1 to 100 characters")}
	p,err:=ParseAdminRolePermissions(role.Permissions)
	if err!=nil{return err}
	role.Permissions=p
	if role.ScopeType==""{role.ScopeType=AdminScopeGlobal}
	if role.ScopeType!=AdminScopeGlobal&&role.ScopeType!=AdminScopeGroup&&role.ScopeType!=AdminScopeIndividual{
		return errors.New("role scope must be global, group or individual")
	}
	if role.ScopeType==AdminScopeGlobal{
		role.ScopeId=0
	}else {
		if role.ScopeId==0{return errors.New("scoped role must specify a group or user ID")}
		var count int64
		if role.ScopeType==AdminScopeGroup{
			err=DB.Model(&model.Group{}).Where("id = ?",role.ScopeId).Count(&count).Error
		}else{
			err=DB.Model(&model.User{}).Where("id = ?",role.ScopeId).Count(&count).Error
		}
		if err!=nil{return err}
		if count!=1{return errors.New("role scope references a missing group or user")}
		for _,item:=range strings.Split(role.Permissions,","){
			if !roleSupportsScope(item,role.ScopeType){
				return fmt.Errorf("permission %s needs a global role",item)
			}
		}
	}
	return nil
}

func SaveAdminRole(role *model.AdminRole)error{
	if err:=ValidateAdminRole(role);err!=nil{return err}
	if role.Id==0{return DB.Create(role).Error}
	r:=DB.Model(&model.AdminRole{}).Where("id = ?",role.Id).
		Updates(map[string]interface{}{
			"name":role.Name,"permissions":role.Permissions,
			"scope_type":role.ScopeType,"scope_id":role.ScopeId,
			"enabled":role.Enabled,
		})
	if r.Error!=nil{return r.Error}
	if r.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteAdminRole(roleID uint)error{
	if roleID==0{return errors.New("role ID is required")}
	return DB.Transaction(func(tx *gorm.DB)error{
		if err:=tx.Where("role_id = ?",roleID).Delete(&model.AdminRoleAssignment{}).Error;err!=nil{return err}
		result:=tx.Where("id = ?",roleID).Delete(&model.AdminRole{})
		if result.Error!=nil{return result.Error}
		if result.RowsAffected!=1{return gorm.ErrRecordNotFound}
		return nil
	})
}

func SaveAdminRoleAssignment(row *model.AdminRoleAssignment)error{
	if row==nil||row.UserId==0||row.RoleId==0{return errors.New("account and role are required")}
	var user model.User
	if err:=DB.Where("id = ?",row.UserId).First(&user).Error;err!=nil{return err}
	var role model.AdminRole
	if err:=DB.Where("id = ?",row.RoleId).First(&role).Error;err!=nil{return err}
	if row.Id==0{
		return DB.Clauses(clause.OnConflict{DoNothing:true}).Create(row).Error
	}
	result:=DB.Model(&model.AdminRoleAssignment{}).Where("id = ?",row.Id).
		Updates(map[string]interface{}{"user_id":row.UserId,"role_id":row.RoleId})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteAdminRoleAssignment(rowID uint)error {
	if rowID==0{return errors.New("assignment ID is required")}
	r:=DB.Where("id = ?",rowID).Delete(&model.AdminRoleAssignment{})
	if r.Error!=nil{return r.Error}
	if r.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func ListAdminRoles(page,size uint)([]model.AdminRole,int64,error){
	var roles []model.AdminRole
	var count int64
	q:=DB.Model(&model.AdminRole{})
	if err:=q.Count(&count).Error;err!=nil{return nil,0,err}
	err:=q.Order("id DESC").Scopes(Paginate(page,size)).Find(&roles).Error
	return roles,count,err
}

func ListAdminRoleAssignments(page,size uint)([]model.AdminRoleAssignment,int64,error){
	var roles []model.AdminRoleAssignment
	var count int64
	q:=DB.Model(&model.AdminRoleAssignment{})
	if err:=q.Count(&count).Error;err!=nil{return nil,0,err}
	err:=q.Order("id DESC").Scopes(Paginate(page,size)).Find(&roles).Error
	return roles,count,err
}

func adminRolesForUser(userID uint)([]model.AdminRole,error){
	var granted []model.AdminRole
	if userID==0{return granted,nil}
	var ids []uint
	if err:=DB.Model(&model.AdminRoleAssignment{}).
		Where("user_id = ?",userID).Pluck("role_id",&ids).Error;err!=nil{return nil,err}
	if len(ids)==0{return granted,nil}
	err:=DB.Where("enabled = ? AND id IN ?",true,ids).Find(&granted).Error
	return granted,err
}

type AdminRoleAccess struct {
	Global bool
	UserIDs []uint
}

// RoleAccessForUser combines all active roles assigned to an account.
// Global roles grant the permission installation-wide. Scoped roles may also
// grant selected write permissions that enforce target-account restrictions.
func RoleAccessForUser(user *model.User,permission string)(AdminRoleAccess,error){
	result:=AdminRoleAccess{UserIDs:[]uint{}}
	if user==nil||user.Id==0||!knownAdminPermission[permission]{return result,nil}
	if AllService.UserService.IsAdmin(user){result.Global=true;return result,nil}
	roles,err:=adminRolesForUser(user.Id)
	if err!=nil{return result,err}
	visible:=make(map[uint]bool)
	for _,role:=range roles{
		allowed:=false
		for _,p:=range strings.Split(role.Permissions,","){
			if p==permission{allowed=true;break}
		}
		if !allowed||!roleSupportsScope(permission,role.ScopeType){continue}
		switch role.ScopeType{
		case AdminScopeGlobal:
			result.Global=true
			return result,nil
		case AdminScopeIndividual:
			if role.ScopeId>0{visible[role.ScopeId]=true}
		case AdminScopeGroup:
			var primary,secondary []uint
			if err:=DB.Model(&model.User{}).Where("group_id = ?",role.ScopeId).
				Pluck("id",&primary).Error;err!=nil{return result,err}
			if err:=DB.Model(&model.GroupMembership{}).Where("group_id = ?",role.ScopeId).
				Pluck("user_id",&secondary).Error;err!=nil{return result,err}
			for _,id:=range append(primary,secondary...){visible[id]=true}
		}
	}
	for id:=range visible{result.UserIDs=append(result.UserIDs,id)}
	sort.Slice(result.UserIDs,func(i,j int)bool{return result.UserIDs[i]<result.UserIDs[j]})
	return result,nil
}

func AdminRolePermissionNames(user *model.User)([]string,error) {
	if user==nil||user.Id==0{return []string{},nil}
	if AllService.UserService.IsAdmin(user){return []string{"*"},nil}
	roles,err:=adminRolesForUser(user.Id)
	if err!=nil{return nil,err}
	seen:=make(map[string]bool)
	for _,role:=range roles{
		for _,p:=range strings.Split(role.Permissions,","){
			if knownAdminPermission[p]&&roleSupportsScope(p,role.ScopeType){seen[p]=true}
		}
	}
	list:=make([]string,0,len(seen))
	for p:=range seen{list=append(list,p)}
	sort.Strings(list)
	return list,nil
}

func ScopeCanViewUser(scope AdminRoleAccess,userID uint)bool {
	if scope.Global{return true}
	for _,id:=range scope.UserIDs{if id==userID{return true}}
	return false
}
