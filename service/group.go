package service

import (
	"errors"
	"strconv"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type GroupService struct {
}

// InfoById 根据用户id取用户信息
func (us *GroupService) InfoById(id uint) *model.Group {
	u := &model.Group{}
	DB.Where("id = ?", id).First(u)
	return u
}

func (us *GroupService) List(page, pageSize uint, where func(tx *gorm.DB), sort ...string) (res *model.GroupList) {
	res = &model.GroupList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.Group{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page,pageSize),SortScope(sort...))
	tx.Find(&res.Groups)
	return
}

// Create 创建
func (us *GroupService) Create(u *model.Group) error {
	res := DB.Create(u).Error
	return res
}
func (us *GroupService) Delete(u *model.Group) error {
	if u == nil || u.Id <= 1 { return errors.New("cannot delete the default group") }
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("group_id = ?", u.Id).
			Update("group_id", uint(1)).Error; err != nil { return err }
		if err := tx.Where("group_id = ?", u.Id).
			Delete(&model.GroupMembership{}).Error; err != nil { return err }
		if err := tx.Where("group_id = ?", u.Id).
			Delete(&model.GroupAnchor{}).Error; err != nil { return err }
		if tx.Migrator().HasTable(&model.AccessRule{}) {
			if err:=tx.Where("source_type = ? AND source_id = ?",accessSourceGroup,u.Id).
				Delete(&model.AccessRule{}).Error;err!=nil{return err}
		}
		if tx.Migrator().HasTable(&model.AdminRole{}) {
			var scoped []uint
			if err:=tx.Model(&model.AdminRole{}).
				Where("scope_type = ? AND scope_id = ?",AdminScopeGroup,u.Id).
				Pluck("id",&scoped).Error;err!=nil{return err}
			if len(scoped)>0{
				if err:=tx.Where("role_id IN ?",scoped).Delete(&model.AdminRoleAssignment{}).Error;err!=nil{return err}
				if err:=tx.Where("id IN ?",scoped).Delete(&model.AdminRole{}).Error;err!=nil{return err}
			}
		}
		return tx.Delete(u).Error
	})
}

// Update 更新
func (us *GroupService) Update(u *model.Group) error {
	return DB.Model(u).Updates(u).Error
}

// DeviceGroupInfoById 根据用户id取用户信息
func (us *GroupService) DeviceGroupInfoById(id uint) *model.DeviceGroup {
	u := &model.DeviceGroup{}
	DB.Where("id = ?", id).First(u)
	return u
}

func (us *GroupService) DeviceGroupList(page, pageSize uint, where func(tx *gorm.DB), sort ...string) (res *model.DeviceGroupList) {
	res = &model.DeviceGroupList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.DeviceGroup{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page,pageSize),SortScope(sort...))
	tx.Find(&res.DeviceGroups)
	return
}

func (us *GroupService) DeviceGroupCreate(u *model.DeviceGroup) error {
	res := DB.Create(u).Error
	return res
}
func (us *GroupService) DeviceGroupDelete(u *model.DeviceGroup) error {
	if u==nil||u.Id==0{return errors.New("invalid device group")}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err:=tx.Model(&model.Peer{}).Where("group_id = ?",u.Id).
			Update("group_id",uint(0)).Error;err!=nil{return err}
		if err:=tx.Where("group_id = ?",u.Id).
			Delete(&model.DevicePolicyGroupAssignment{}).Error;err!=nil{return err}
		if err:=tx.Where("target_type = ? AND target_id = ?",controlTargetDeviceGroup,
			strconv.FormatUint(uint64(u.Id),10)).
			Delete(&model.ControlRoleAssignment{}).Error;err!=nil{return err}
		if tx.Migrator().HasTable(&model.AccessRule{}) {
			if err:=tx.Where("target_type = ? AND target_id = ?",accessTargetDeviceGroup,
				strconv.FormatUint(uint64(u.Id),10)).
				Delete(&model.AccessRule{}).Error;err!=nil{return err}
		}
		result:=tx.Delete(u)
		if result.Error!=nil{return result.Error}
		if result.RowsAffected==0{return gorm.ErrRecordNotFound}
		return nil
	})
}

func (us *GroupService) DeviceGroupUpdate(u *model.DeviceGroup) error {
	return DB.Model(u).Updates(u).Error
}
