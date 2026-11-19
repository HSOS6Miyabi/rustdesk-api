package service

import (
	"strconv"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

// EnrichGroupDirectory attaches member and owned-device totals for a page of groups.
// Primary and additional group memberships are deduplicated before counting devices.
func EnrichGroupDirectory(groups []*model.Group) error {
	if len(groups) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(groups))
	byID := make(map[uint]*model.Group, len(groups))
	for _, group := range groups {
		ids = append(ids, group.Id)
		byID[group.Id] = group
	}
	var members []struct {
		GroupID     uint  `gorm:"column:group_id"`
		MemberCount int64 `gorm:"column:member_count"`
		DeviceCount int64 `gorm:"column:device_count"`
	}
	err := DB.Raw(`
        SELECT membership.group_id,
               COUNT(DISTINCT membership.user_id) AS member_count,
               COUNT(DISTINCT peers.row_id) AS device_count
        FROM (
            SELECT id AS user_id, group_id FROM users WHERE group_id IN ?
            UNION
            SELECT user_id, group_id FROM group_memberships WHERE group_id IN ?
        ) AS membership
        LEFT JOIN peers ON peers.user_id = membership.user_id
        GROUP BY membership.group_id`, ids, ids).Scan(&members).Error
	if err != nil {
		return err
	}
	for _, entry := range members {
		if group := byID[entry.GroupID]; group != nil {
			group.MemberCount = entry.MemberCount
			group.DeviceCount = entry.DeviceCount
		}
	}
	var anchors []struct {
		GroupID     uint  `gorm:"column:group_id"`
		AnchorCount int64 `gorm:"column:anchor_count"`
	}
	if err := DB.Model(&model.GroupAnchor{}).Select("group_id, COUNT(*) AS anchor_count").
		Where("group_id IN ?", ids).Group("group_id").Scan(&anchors).Error; err != nil {
		return err
	}
	for _, entry := range anchors {
		if group := byID[entry.GroupID]; group != nil {
			group.AnchorCount = entry.AnchorCount
		}
	}
	return nil
}

// EnrichDeviceGroupDirectory supplies totals and readable policy/role names for one page.
func EnrichDeviceGroupDirectory(groups []*model.DeviceGroup) error {
	if len(groups) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(groups))
	stringIDs := make([]string, 0, len(groups))
	byID := make(map[uint]*model.DeviceGroup, len(groups))
	for _, group := range groups {
		ids = append(ids, group.Id)
		stringIDs = append(stringIDs, strconv.FormatUint(uint64(group.Id), 10))
		byID[group.Id] = group
	}
	var counts []struct {
		GroupID     uint  `gorm:"column:group_id"`
		DeviceCount int64 `gorm:"column:device_count"`
	}
	if err := DB.Model(&model.Peer{}).Select("group_id, COUNT(*) AS device_count").
		Where("group_id IN ?", ids).Group("group_id").Scan(&counts).Error; err != nil {
		return err
	}
	for _, entry := range counts {
		if group := byID[entry.GroupID]; group != nil {
			group.DeviceCount = entry.DeviceCount
		}
	}
	var policies []struct {
		GroupID    uint   `gorm:"column:group_id"`
		PolicyName string `gorm:"column:policy_name"`
	}
	if err := DB.Table("device_policy_group_assignments AS assignment").
		Select("assignment.group_id, policy.name AS policy_name").
		Joins("JOIN device_policies AS policy ON policy.id = assignment.policy_id").
		Where("assignment.group_id IN ?", ids).Scan(&policies).Error; err != nil {
		return err
	}
	for _, entry := range policies {
		if group := byID[entry.GroupID]; group != nil {
			group.PolicyName = entry.PolicyName
		}
	}
	var roles []struct {
		TargetID string `gorm:"column:target_id"`
		RoleName string `gorm:"column:role_name"`
	}
	if err := DB.Table("control_role_assignments AS assignment").
		Select("assignment.target_id, role.name AS role_name").
		Joins("JOIN control_roles AS role ON role.id = assignment.role_id").
		Where("assignment.target_type = ? AND assignment.target_id IN ?", "device_group", stringIDs).
		Scan(&roles).Error; err != nil {
		return err
	}
	for _, entry := range roles {
		groupID, err := strconv.ParseUint(entry.TargetID, 10, 32)
		if err == nil {
			if group := byID[uint(groupID)]; group != nil {
				group.ControlRoleName = entry.RoleName
			}
		}
	}
	return nil
}

func EnrichUserDeviceCounts(users []*model.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(users))
	byID := make(map[uint]*model.User, len(users))
	for _, user := range users {
		ids = append(ids, user.Id)
		byID[user.Id] = user
	}
	var counts []struct {
		UserID      uint  `gorm:"column:user_id"`
		DeviceCount int64 `gorm:"column:device_count"`
	}
	if err := DB.Model(&model.Peer{}).Select("user_id, COUNT(*) AS device_count").
		Where("user_id IN ?", ids).Group("user_id").Scan(&counts).Error; err != nil {
		return err
	}
	for _, entry := range counts {
		if user := byID[entry.UserID]; user != nil {
			user.DeviceCount = entry.DeviceCount
		}
	}
	return nil
}
