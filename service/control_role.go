package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	controlTargetDevice="device"
	controlTargetUser="user"
	controlTargetDeviceGroup="device_group"
)

func SaveControlRole(role *model.ControlRole)error {
	if role==nil{return errors.New("control role is required")}
	role.Name=strings.TrimSpace(role.Name)
	if role.Name==""||len([]rune(role.Name))>100{
		return errors.New("control role name must contain 1 to 100 characters")
	}
	if _,err:=role.PermissionBits();err!=nil{return err}
	if role.Id==0{return DB.Create(role).Error}
	result:=DB.Model(&model.ControlRole{}).Where("id = ?",role.Id).
		Updates(map[string]interface{}{
			"name":role.Name,"keyboard":role.Keyboard,"remote_printer":role.RemotePrinter,
			"clipboard":role.Clipboard,"file":role.File,"audio":role.Audio,
			"camera":role.Camera,"terminal":role.Terminal,"tunnel":role.Tunnel,
			"restart":role.Restart,"recording":role.Recording,
			"block_input":role.BlockInput,"remote_modify":role.RemoteModify,
			"privacy_mode":role.PrivacyMode,"usb":role.USB,
		})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteControlRole(id uint)error{
	if id==0{return errors.New("control role ID is required")}
	return DB.Transaction(func(tx *gorm.DB)error{
		if err:=tx.Where("role_id = ?",id).
			Delete(&model.ControlRoleAssignment{}).Error;err!=nil{return err}
		result:=tx.Where("id = ?",id).Delete(&model.ControlRole{})
		if result.Error!=nil{return result.Error}
		if result.RowsAffected==0{return gorm.ErrRecordNotFound}
		return nil
	})
}

func validateControlTarget(targetType,targetID string)error{
	if targetType!=controlTargetDevice&&targetType!=controlTargetUser&&
		targetType!=controlTargetDeviceGroup{
		return errors.New("control target type must be device, user, or device_group")
	}
	if targetID==""||len(targetID)>253{return errors.New("control target ID is required")}
	var count int64
	switch targetType{
	case controlTargetDevice:
		if err:=DB.Model(&model.Peer{}).Where("id = ?",targetID).Count(&count).Error;err!=nil{return err}
	case controlTargetUser,controlTargetDeviceGroup:
		number,err:=strconv.ParseUint(targetID,10,32)
		if err!=nil||number==0{return errors.New("control target must be a numeric account or group ID")}
		if targetType==controlTargetUser{
			if err:=DB.Model(&model.User{}).Where("id = ?",number).Count(&count).Error;err!=nil{return err}
		}else{
			if err:=DB.Model(&model.DeviceGroup{}).Where("id = ?",number).Count(&count).Error;err!=nil{return err}
		}
	}
	if count!=1{return fmt.Errorf("%s %s does not exist",targetType,targetID)}
	return nil
}

func SaveControlRoleAssignment(item *model.ControlRoleAssignment)error{
	if item==nil||item.RoleId==0{return errors.New("control role assignment is required")}
	item.TargetType=strings.TrimSpace(item.TargetType)
	item.TargetId=strings.TrimSpace(item.TargetId)
	if err:=validateControlTarget(item.TargetType,item.TargetId);err!=nil{return err}
	if item.TargetType==controlTargetUser||item.TargetType==controlTargetDeviceGroup{
		number,_:=strconv.ParseUint(item.TargetId,10,32)
		item.TargetId=strconv.FormatUint(number,10)
	}
	var role model.ControlRole
	if err:=DB.Where("id = ?",item.RoleId).First(&role).Error;err!=nil{
		return errors.New("control role does not exist")
	}
	if item.Id==0{
		return DB.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"target_type"},{Name:"target_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"role_id","updated_at"}),
		}).Create(item).Error
	}
	result:=DB.Model(&model.ControlRoleAssignment{}).Where("id = ?",item.Id).
		Updates(map[string]interface{}{
			"target_type":item.TargetType,"target_id":item.TargetId,"role_id":item.RoleId,
		})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteControlRoleAssignment(id uint)error{
	if id==0{return errors.New("assignment ID is required")}
	result:=DB.Where("id = ?",id).Delete(&model.ControlRoleAssignment{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func ListControlRoles(page,pageSize uint,sort ...string)([]model.ControlRole,int64,error){
	var count int64
	var roles []model.ControlRole
	tx:=DB.Model(&model.ControlRole{})
	if err:=tx.Count(&count).Error;err!=nil{return nil,0,err}
	err:=tx.Order("id DESC").Scopes(Paginate(page,pageSize),SortScope(sort...)).Find(&roles).Error
	return roles,count,err
}

func ListControlRoleAssignments(page,pageSize uint,sort ...string)([]model.ControlRoleAssignment,int64,error){
	var count int64
	var rows []model.ControlRoleAssignment
	tx:=DB.Model(&model.ControlRoleAssignment{})
	if err:=tx.Count(&count).Error;err!=nil{return nil,0,err}
	err:=tx.Order("id DESC").Scopes(Paginate(page,pageSize),SortScope(sort...)).Find(&rows).Error
	return rows,count,err
}

// EffectiveControlPermissions follows the same assignment order as device
// strategies: device, owning account, device group.
func EffectiveControlPermissions(deviceID string)(uint64,bool,string,error){
	if deviceID==""||len(deviceID)>253{return 0,false,"",errors.New("valid device ID is required")}
	var peer model.Peer
	if err:=DB.Where("id = ?",deviceID).First(&peer).Error;err!=nil{
		if errors.Is(err,gorm.ErrRecordNotFound){return 0,false,"",nil}
		return 0,false,"",err
	}
	targets:=[]struct{kind,id string}{
		{controlTargetDevice,peer.Id},
	}
	if peer.UserId>0{
		targets=append(targets,struct{kind,id string}{controlTargetUser,strconv.FormatUint(uint64(peer.UserId),10)})
	}
	if peer.GroupId>0{
		targets=append(targets,struct{kind,id string}{controlTargetDeviceGroup,strconv.FormatUint(uint64(peer.GroupId),10)})
	}
	for _,target:=range targets{
		var record model.ControlRoleAssignment
		err:=DB.Where("target_type = ? AND target_id = ?",target.kind,target.id).
			First(&record).Error
		if errors.Is(err,gorm.ErrRecordNotFound){continue}
		if err!=nil{return 0,false,"",err}
		var role model.ControlRole
		err=DB.Where("id = ?",record.RoleId).First(&role).Error
		if errors.Is(err,gorm.ErrRecordNotFound){continue}
		if err!=nil{return 0,false,"",err}
		bits,err:=role.PermissionBits()
		if err!=nil{return 0,false,"",err}
		return bits,true,target.kind,nil
	}
	return 0,false,"",nil
}
