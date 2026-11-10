package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResolvedPolicy struct {
	Policy model.DevicePolicy
	Source string
	Revision int64
}

func devicePolicyRevision(policy *model.DevicePolicy, assignmentRevision int64, assignmentUpdatedAt time.Time, sourceIndex int64) int64 {
	revision := policy.Revision
	if assignmentRevision > revision { revision = assignmentRevision }
	if updated := time.Time(policy.UpdatedAt).UnixNano(); updated > revision { revision = updated }
	if updated := assignmentUpdatedAt.UnixNano(); updated > revision { revision = updated }
	if revision <= 0 { revision = 1 }
	// Keep the source in the version so switching to another assignment
	// changes the client's cached revision even when timestamps are equal.
	return (revision/10)*10 + sourceIndex
}

func findActivePolicy(id uint, source string, sourceIndex int64, revision int64, updatedAt time.Time) (*ResolvedPolicy,error) {
	var policy model.DevicePolicy
	err := DB.Where("id = ?", id).First(&policy).Error
	if errors.Is(err,gorm.ErrRecordNotFound) { return nil,nil }
	if err != nil { return nil,err }
	if policy.Status == 2 { return nil,nil }
	return &ResolvedPolicy{
		Policy:policy, Source:source,
		Revision:devicePolicyRevision(&policy,revision,updatedAt,sourceIndex),
	},nil
}

// ResolveDevicePolicy applies assignments in order: device, user, device group.
// A disabled or removed policy does not override a lower-priority active one.
func ResolveDevicePolicy(peer *model.Peer) (*ResolvedPolicy,error) {
	if peer==nil || peer.RowId==0 { return nil,nil }
	var direct model.DevicePolicyAssignment
	err:=DB.Where("device_id = ?",peer.Id).First(&direct).Error
	if err==nil {
		p,err:=findActivePolicy(direct.PolicyId,"device",1,direct.Revision,time.Time(direct.UpdatedAt))
		if err!=nil || p!=nil{return p,err}
	} else if !errors.Is(err,gorm.ErrRecordNotFound){return nil,err}
	if peer.UserId!=0 {
		var user model.DevicePolicyUserAssignment
		err:=DB.Where("user_id = ?",peer.UserId).First(&user).Error
		if err==nil {
			p,err:=findActivePolicy(user.PolicyId,"user",2,user.Revision,time.Time(user.UpdatedAt))
			if err!=nil || p!=nil{return p,err}
		} else if !errors.Is(err,gorm.ErrRecordNotFound){return nil,err}
	}
	if peer.GroupId!=0 {
		var group model.DevicePolicyGroupAssignment
		err:=DB.Where("group_id = ?",peer.GroupId).First(&group).Error
		if err==nil{
			p,err:=findActivePolicy(group.PolicyId,"device_group",3,group.Revision,time.Time(group.UpdatedAt))
			if err!=nil || p!=nil{return p,err}
		}else if !errors.Is(err,gorm.ErrRecordNotFound){return nil,err}
	}
	return nil,nil
}

func policyAssignmentTargetExists(table interface{}, id uint) error {
	if id==0 {return errors.New("target ID must be positive")}
	var count int64
	if err:=DB.Model(table).Where("id = ?",id).Count(&count).Error;err!=nil{return err}
	if count==0{return fmt.Errorf("target %d was not found",id)}
	return nil
}

func policyExists(id uint) error {
	if id==0{return errors.New("device policy is required")}
	return policyAssignmentTargetExists(&model.DevicePolicy{},id)
}

func SaveDevicePolicyUserAssignment(item *model.DevicePolicyUserAssignment) error {
	if item==nil{return errors.New("user assignment is required")}
	if err:=policyExists(item.PolicyId);err!=nil{return err}
	if err:=policyAssignmentTargetExists(&model.User{},item.UserId);err!=nil{return err}
	item.Revision=time.Now().UnixNano()
	if item.Id==0 {
		err:=DB.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"user_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"policy_id","revision","updated_at"}),
		}).Create(item).Error
		if err==nil {invalidatePolicyActivity()}
		return err
	}
	result:=DB.Model(&model.DevicePolicyUserAssignment{}).Where("id = ?",item.Id).
		Updates(map[string]interface{}{"user_id":item.UserId,"policy_id":item.PolicyId,"revision":item.Revision})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	invalidatePolicyActivity()
	return nil
}

func DeleteDevicePolicyUserAssignment(id uint) error {
	if id==0{return errors.New("assignment ID is required")}
	result:=DB.Where("id = ?",id).Delete(&model.DevicePolicyUserAssignment{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	invalidatePolicyActivity()
	return nil
}

func SaveDevicePolicyGroupAssignment(item *model.DevicePolicyGroupAssignment) error {
	if item==nil{return errors.New("device group assignment is required")}
	if err:=policyExists(item.PolicyId);err!=nil{return err}
	if err:=policyAssignmentTargetExists(&model.DeviceGroup{},item.GroupId);err!=nil{return err}
	item.Revision=time.Now().UnixNano()
	if item.Id==0 {
		err:=DB.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"group_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"policy_id","revision","updated_at"}),
		}).Create(item).Error
		if err==nil {invalidatePolicyActivity()}
		return err
	}
	result:=DB.Model(&model.DevicePolicyGroupAssignment{}).Where("id = ?",item.Id).
		Updates(map[string]interface{}{"group_id":item.GroupId,"policy_id":item.PolicyId,"revision":item.Revision})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	invalidatePolicyActivity()
	return nil
}

func DeleteDevicePolicyGroupAssignment(id uint) error {
	if id==0{return errors.New("assignment ID is required")}
	result:=DB.Where("id = ?",id).Delete(&model.DevicePolicyGroupAssignment{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	invalidatePolicyActivity()
	return nil
}
