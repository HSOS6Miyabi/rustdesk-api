package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrDeviceGroupChanged = errors.New("设备分组已发生变化，请刷新设备列表后重新选择")

type DeviceGroupDevice struct {
	RowId          uint   `json:"row_id"`
	Id             string `json:"id"`
	Alias          string `json:"alias"`
	Hostname       string `json:"hostname"`
	Os             string `json:"os"`
	UserId         uint   `json:"user_id"`
	OwnerUsername  string `json:"owner_username"`
	GroupId        uint   `json:"group_id"`
	GroupName      string `json:"group_name"`
	LastOnlineTime int64  `json:"last_online_time"`
}

type DeviceGroupMemberChange struct {
	RowId           uint  `json:"row_id"`
	ExpectedGroupId *uint `json:"expected_group_id"`
}

// ListDeviceGroupDevices returns a server-paginated member or candidate list.
func ListDeviceGroupDevices(groupID uint, candidates bool, keyword string, page, size uint) ([]DeviceGroupDevice, int64, error) {
	var group model.DeviceGroup
	if groupID == 0 {
		return nil, 0, errors.New("请选择设备分组")
	}
	if err := DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return nil, 0, err
	}
	if page == 0 {
		page = 1
	}
	if size == 0 || size > 100 {
		size = 20
	}
	query := DB.Table("peers AS p").
		Joins("LEFT JOIN users u ON u.id = p.user_id").
		Joins("LEFT JOIN device_groups g ON g.id = p.group_id")
	if candidates {
		query = query.Where("p.group_id <> ?", groupID)
	} else {
		query = query.Where("p.group_id = ?", groupID)
	}
	keyword = strings.TrimSpace(keyword)
	if len([]rune(keyword)) > 200 {
		return nil, 0, errors.New("搜索内容不能超过 200 个字符")
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("p.id LIKE ? OR p.alias LIKE ? OR p.hostname LIKE ? OR u.username LIKE ? OR g.name LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]DeviceGroupDevice, 0)
	err := query.Select("p.row_id, p.id, p.alias, p.hostname, p.os, p.user_id, COALESCE(u.username, '') AS owner_username, p.group_id, COALESCE(g.name, '') AS group_name, p.last_online_time").
		Order("p.last_online_time DESC, p.row_id ASC").Offset(int(page-1) * int(size)).Limit(int(size)).Scan(&rows).Error
	return rows, total, err
}

// ChangeDeviceGroupMembers validates every selection before changing membership.
// Expected memberships preserve changes made by another administrator.
func ChangeDeviceGroupMembers(groupID uint, changes []DeviceGroupMemberChange, action string) (int64, error) {
	if groupID == 0 || (action != "add" && action != "remove") {
		return 0, errors.New("请选择设备分组和有效的操作")
	}
	if len(changes) == 0 || len(changes) > 200 {
		return 0, errors.New("每次请选择 1 至 200 台设备")
	}
	expected := make(map[uint]uint, len(changes))
	ids := make([]uint, 0, len(changes))
	for _, item := range changes {
		if item.RowId == 0 || item.ExpectedGroupId == nil {
			return 0, errors.New("请选择设备并提供当前分组编号")
		}
		if _, exists := expected[item.RowId]; exists {
			return 0, errors.New("所选设备不能重复")
		}
		expected[item.RowId] = *item.ExpectedGroupId
		ids = append(ids, item.RowId)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var affected int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var group model.DeviceGroup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", groupID).First(&group).Error; err != nil {
			return err
		}
		var peers []model.Peer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("row_id IN ?", ids).Order("row_id ASC").Find(&peers).Error; err != nil {
			return err
		}
		if len(peers) != len(ids) {
			return errors.New("部分设备已不存在，请刷新设备列表")
		}
		for _, peer := range peers {
			if peer.GroupId != expected[peer.RowId] || (action == "remove" && peer.GroupId != groupID) {
				return ErrDeviceGroupChanged
			}
		}
		target := groupID
		if action == "remove" {
			target = 0
		}
		touched := map[uint]bool{groupID: true}
		for _, peer := range peers {
			if peer.GroupId == target {
				continue
			}
			result := tx.Model(&model.Peer{}).Where("row_id = ? AND group_id = ?", peer.RowId, peer.GroupId).Update("group_id", target)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrDeviceGroupChanged
			}
			affected += result.RowsAffected
			if peer.GroupId > 0 {
				touched[peer.GroupId] = true
			}
		}
		if affected > 0 {
			touchedIDs := make([]uint, 0, len(touched))
			for id := range touched {
				touchedIDs = append(touchedIDs, id)
			}
			if err := tx.Model(&model.DeviceGroup{}).Where("id IN ?", touchedIDs).Update("updated_at", time.Now()).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if affected > 0 {
		invalidatePolicyActivity()
	}
	return affected, nil
}
