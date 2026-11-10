package service

import (
	"errors"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

// DevicePresenceForUser returns a bounded snapshot of authenticated heartbeat
// status. Device IDs are checked against ownership, address books and shared
// devices before their presence records are read.
func DevicePresenceForUser(user *model.User, ids []string, administrator bool) (map[string]bool, error) {
	if user == nil || user.Id == 0 { return nil, errors.New("请先登录") }
	if len(ids) > 256 { return nil, errors.New("每次最多查询 256 台设备") }
	results := make(map[string]bool, len(ids))
	unique := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || len(id) > 253 { continue }
		if _, ok := results[id]; ok { continue }
		results[id] = false
		unique = append(unique, id)
	}
	if len(unique) == 0 { return results, nil }

	type deviceOwner struct {
		RowId  uint
		Id     string
		UserId uint
	}
	var peers []deviceOwner
	if err := DB.Model(&model.Peer{}).Select("row_id,id,user_id").
		Where("id IN ?", unique).Find(&peers).Error; err != nil {
		return nil, err
	}

	allowed := make(map[string]bool, len(unique))
	if administrator {
		for _, id := range unique { allowed[id] = true }
	} else {
		// An address-book entry represents a device the user has explicitly
		// saved. It may refer to a device outside their own account.
		var bookIDs []string
		if err := DB.Model(&model.AddressBook{}).
			Where("user_id = ? AND id IN ?", user.Id, unique).
			Distinct().Pluck("id", &bookIDs).Error; err != nil {
			return nil, err
		}
		for _, id := range bookIDs { allowed[id] = true }
		needsSharing := false
		for _, peer := range peers {
			if peer.UserId == user.Id {
				allowed[peer.Id] = true
			} else if !allowed[peer.Id] {
				needsSharing = true
			}
		}
		if needsSharing {
			owners, err := sharedOwnerSources(user)
			if err != nil { return nil, err }
			for _, peer := range peers {
				if len(owners[peer.UserId]) > 0 { allowed[peer.Id] = true }
			}
		}
	}
	rowIDs := make([]uint, 0, len(peers))
	byRowID := make(map[uint]string, len(peers))
	for _, peer := range peers {
		if allowed[peer.Id] {
			rowIDs = append(rowIDs, peer.RowId)
			byRowID[peer.RowId] = peer.Id
		}
	}
	if len(rowIDs) == 0 { return results, nil }
	var rows []model.PushPresence
	if err := DB.Select("peer_row_id").Where("peer_row_id IN ? AND online = ? AND last_seen_at >= ?",
		rowIDs, true, time.Now().Add(-120*time.Second).Unix()).
		Find(&rows).Error; err != nil { return nil, err }
	for _, presence := range rows {
		results[byRowID[presence.PeerRowId]] = true
	}
	return results, nil
}
