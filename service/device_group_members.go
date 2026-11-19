package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// DeviceGroupDevices lists registered devices belonging to a single group.
func (gs *GroupService) DeviceGroupDevices(groupID, page, size uint, keyword string) (*model.PeerList, error) {
	if groupID == 0 {
		return nil, errors.New("请选择有效的设备分组")
	}
	var exists int64
	if err := DB.Model(&model.DeviceGroup{}).Where("id = ?", groupID).Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	keyword = strings.TrimSpace(keyword)
	if len([]rune(keyword)) > 128 {
		return nil, errors.New("搜索内容过长")
	}
	result := &model.PeerList{Pagination: model.Pagination{Page: int64(page), PageSize: int64(size)}}
	tx := DirectoryPeerKeyword(DB.Model(&model.Peer{}).Where("group_id = ?", groupID), keyword)
	if err := tx.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := tx.Order("row_id DESC").Scopes(Paginate(page, size)).Find(&result.Peers).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// ChangeDeviceGroupMembers changes the membership of existing registered devices.
// Every requested row must exist; removal also requires current membership.
func (gs *GroupService) ChangeDeviceGroupMembers(groupID uint, rowIDs []uint, action string) error {
	if groupID == 0 {
		return errors.New("请选择有效的设备分组")
	}
	if action != "add" && action != "remove" {
		return errors.New("无效的设备分组操作")
	}
	if len(rowIDs) == 0 || len(rowIDs) > 100 {
		return errors.New("每次请选择 1 至 100 台设备")
	}
	unique := make([]uint, 0, len(rowIDs))
	seen := make(map[uint]struct{}, len(rowIDs))
	for _, id := range rowIDs {
		if id == 0 {
			return errors.New("设备记录编号必须为正整数")
		}
		if _, present := seen[id]; present {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var groupCount int64
		if err := tx.Model(&model.DeviceGroup{}).Where("id = ?", groupID).Count(&groupCount).Error; err != nil {
			return err
		}
		if groupCount != 1 {
			return gorm.ErrRecordNotFound
		}
		var devices []model.Peer
		if err := tx.Select("row_id", "group_id").Where("row_id IN ?", unique).Find(&devices).Error; err != nil {
			return err
		}
		if len(devices) != len(unique) {
			return errors.New("部分设备记录已不存在，请刷新后重试")
		}
		if action == "remove" {
			for _, device := range devices {
				if device.GroupId != groupID {
					return fmt.Errorf("设备记录 %d 已不在当前分组，请刷新后重试", device.RowId)
				}
			}
		}
		target := uint(0)
		if action == "add" {
			target = groupID
		}
		query := tx.Model(&model.Peer{}).Where("row_id IN ?", unique)
		if action == "remove" {
			query = query.Where("group_id = ?", groupID)
		}
		result := query.Update("group_id", target)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(unique)) {
			return errors.New("设备分组已发生变化，请刷新后重试")
		}
		return nil
	})
}
