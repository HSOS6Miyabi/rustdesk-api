package service

import (
	"strconv"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

// Console summaries aggregate only the records on the requested page.
type ConsoleGroup struct {
	*model.Group
	MemberCount int64 `json:"member_count"`
	DeviceCount int64 `json:"device_count"`
	AnchorCount int64 `json:"anchor_count"`
}

type ConsoleDeviceGroup struct {
	*model.DeviceGroup
	DeviceCount       int64  `json:"device_count"`
	ActiveCount       int64  `json:"active_count"`
	LastActiveAt      int64  `json:"last_active_at"`
	GroupPolicy       string `json:"group_policy"`
	GroupPolicyStatus int    `json:"group_policy_status"`
	ControlRole       string `json:"control_role"`
}

type ConsoleUser struct {
	*model.User
	DeviceCount int64 `json:"device_count"`
}

func SummarizeConsoleGroups(groups []*model.Group) ([]ConsoleGroup, error) {
	result := make([]ConsoleGroup, len(groups))
	ids := make([]uint, 0, len(groups))
	positions := make(map[uint]int, len(groups))
	for i, group := range groups {
		result[i].Group = group
		ids = append(ids, group.Id)
		positions[group.Id] = i
	}
	if len(ids) == 0 {
		return result, nil
	}
	var counts []struct {
		GroupId     uint
		MemberCount int64
		DeviceCount int64
	}
	membership := DB.Raw("SELECT group_id, id AS user_id FROM users WHERE group_id IN ? UNION SELECT gm.group_id, gm.user_id FROM group_memberships gm JOIN users u ON u.id = gm.user_id WHERE gm.group_id IN ?", ids, ids)
	if err := DB.Table("(?) AS members", membership).
		Select("members.group_id, COUNT(DISTINCT members.user_id) AS member_count, COUNT(DISTINCT peers.row_id) AS device_count").
		Joins("LEFT JOIN peers ON peers.user_id = members.user_id").
		Group("members.group_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, count := range counts {
		i := positions[count.GroupId]
		result[i].MemberCount, result[i].DeviceCount = count.MemberCount, count.DeviceCount
	}
	var anchors []struct {
		GroupId uint
		Count   int64
	}
	if err := DB.Model(&model.GroupAnchor{}).Select("group_id, COUNT(*) AS count").
		Where("group_id IN ?", ids).Group("group_id").Scan(&anchors).Error; err != nil {
		return nil, err
	}
	for _, count := range anchors {
		result[positions[count.GroupId]].AnchorCount = count.Count
	}
	return result, nil
}

func SummarizeConsoleDeviceGroups(groups []*model.DeviceGroup) ([]ConsoleDeviceGroup, error) {
	result := make([]ConsoleDeviceGroup, len(groups))
	ids := make([]uint, 0, len(groups))
	stringIDs := make([]string, 0, len(groups))
	positions := make(map[uint]int, len(groups))
	for i, group := range groups {
		result[i].DeviceGroup = group
		ids = append(ids, group.Id)
		stringIDs = append(stringIDs, strconv.FormatUint(uint64(group.Id), 10))
		positions[group.Id] = i
	}
	if len(ids) == 0 {
		return result, nil
	}
	var counts []struct {
		GroupId      uint
		DeviceCount  int64
		ActiveCount  int64
		LastActiveAt int64
	}
	if err := DB.Model(&model.Peer{}).
		Select("group_id, COUNT(*) AS device_count, SUM(CASE WHEN last_online_time >= ? THEN 1 ELSE 0 END) AS active_count, MAX(last_online_time) AS last_active_at", time.Now().Add(-2*time.Minute).Unix()).
		Where("group_id IN ?", ids).Group("group_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, count := range counts {
		row := &result[positions[count.GroupId]]
		row.DeviceCount, row.ActiveCount, row.LastActiveAt = count.DeviceCount, count.ActiveCount, count.LastActiveAt
	}
	var policies []struct {
		GroupId uint
		Name    string
		Status  int
	}
	if err := DB.Table("device_policy_group_assignments AS a").
		Select("a.group_id, p.name, p.status").Joins("JOIN device_policies p ON p.id = a.policy_id").
		Where("a.group_id IN ?", ids).Scan(&policies).Error; err != nil {
		return nil, err
	}
	for _, item := range policies {
		row := &result[positions[item.GroupId]]
		row.GroupPolicy, row.GroupPolicyStatus = item.Name, item.Status
	}
	var roles []struct {
		TargetId string
		Name     string
	}
	if err := DB.Table("control_role_assignments AS a").Select("a.target_id, r.name").
		Joins("JOIN control_roles r ON r.id = a.role_id").
		Where("a.target_type = ? AND a.target_id IN ?", "device_group", stringIDs).Scan(&roles).Error; err != nil {
		return nil, err
	}
	for _, item := range roles {
		id, err := strconv.ParseUint(item.TargetId, 10, 64)
		if err != nil {
			continue
		}
		if i, ok := positions[uint(id)]; ok {
			result[i].ControlRole = item.Name
		}
	}
	return result, nil
}

func SummarizeConsoleUsers(users []*model.User) ([]ConsoleUser, error) {
	result := make([]ConsoleUser, len(users))
	ids := make([]uint, 0, len(users))
	positions := make(map[uint]int, len(users))
	for i, user := range users {
		result[i].User = user
		ids = append(ids, user.Id)
		positions[user.Id] = i
	}
	if len(ids) == 0 {
		return result, nil
	}
	var counts []struct {
		UserId uint
		Count  int64
	}
	if err := DB.Model(&model.Peer{}).Select("user_id, COUNT(*) AS count").
		Where("user_id IN ?", ids).Group("user_id").Scan(&counts).Error; err != nil {
		return nil, err
	}
	for _, item := range counts {
		result[positions[item.UserId]].DeviceCount = item.Count
	}
	return result, nil
}
