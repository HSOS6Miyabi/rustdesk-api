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
	statuses, _, err := DevicePresenceDetailsForUser(user, ids, administrator)
	return statuses, err
}

// DevicePresenceDetailsForUser returns status and registered display names only
// for devices that the current user may inspect.
func DevicePresenceDetailsForUser(user *model.User, ids []string, administrator bool) (map[string]bool, map[string]string, error) {
	if user == nil || user.Id == 0 { return nil, nil, errors.New("请先登录") }
	if len(ids) > 256 { return nil, nil, errors.New("每次最多查询 256 台设备") }
	results := make(map[string]bool, len(ids))
	unique := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" || len(id) > 253 { continue }
		if _, ok := results[id]; ok { continue }
		results[id] = false
		unique = append(unique, id)
	}
	if len(unique) == 0 { return results, map[string]string{}, nil }

	type deviceOwner struct {
		RowId  uint
		Id     string
		UserId uint
		LastOnlineTime int64
		Alias string
		Hostname string
	}
	var peers []deviceOwner
	if err := DB.Model(&model.Peer{}).Select("row_id,id,user_id,last_online_time,alias,hostname").
		Where("id IN ?", unique).Find(&peers).Error; err != nil {
		return nil, nil, err
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
			return nil, nil, err
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
			if err != nil { return nil, nil, err }
			for _, peer := range peers {
				if len(owners[peer.UserId]) > 0 { allowed[peer.Id] = true }
			}
			// Delegated management roles can also grant visibility into
			// other owners' registered devices.
			role, err := RoleAccessForUser(user, "devices.view")
			if err != nil { return nil, nil, err }
			if role.Global {
				for _, peer := range peers { allowed[peer.Id] = true }
			} else if len(role.UserIDs) > 0 {
				visible := make(map[uint]bool, len(role.UserIDs))
				for _, id := range role.UserIDs { visible[id] = true }
				for _, peer := range peers {
					if visible[peer.UserId] { allowed[peer.Id] = true }
				}
			}
			// A participant may view the devices named in their own
			// connection audit, even when neither device is shared.
			stillHidden := false
			for _, peer := range peers {
				if !allowed[peer.Id] { stillHidden = true; break }
			}
			if stillHidden {
				// DISTINCT and UNION bound the result to the requested IDs,
				// even when a device has millions of historical audit rows.
				var records []struct { Id string }
				if err := DB.Raw(
					"SELECT DISTINCT from_peer AS id FROM audit_conns "+
						"WHERE (from_user_id = ? OR to_user_id = ?) AND from_peer IN ? "+
						"UNION SELECT DISTINCT peer_id AS id FROM audit_conns "+
						"WHERE (from_user_id = ? OR to_user_id = ?) AND peer_id IN ?",
					user.Id,user.Id,unique,user.Id,user.Id,unique,
				).Scan(&records).Error; err != nil { return nil, nil, err }
				for _, record := range records { allowed[record.Id] = true }
			}
		}
	}
	names := make(map[string]string, len(peers))
	rowIDs := make([]uint, 0, len(peers))
	byRowID := make(map[uint]string, len(peers))
	notifyPresence := hasPushSubscription("device.online") || hasPushSubscription("device.offline")
	cutoff := time.Now().Add(-120*time.Second).Unix()
	for _, peer := range peers {
		if !allowed[peer.Id] { continue }
		name := strings.TrimSpace(peer.Alias)
		if name == "" { name = strings.TrimSpace(peer.Hostname) }
		if name == "" { name = peer.Id }
		names[peer.Id] = name
		if !notifyPresence {
			results[peer.Id] = peer.LastOnlineTime >= cutoff
			continue
		}
		rowIDs = append(rowIDs, peer.RowId)
		byRowID[peer.RowId] = peer.Id
	}
	if !notifyPresence || len(rowIDs) == 0 { return results, names, nil }
	var rows []model.PushPresence
	if err := DB.Select("peer_row_id").Where("peer_row_id IN ? AND online = ? AND last_seen_at >= ?",
		rowIDs, true, time.Now().Add(-120*time.Second).Unix()).
		Find(&rows).Error; err != nil { return nil, nil, err }
	for _, presence := range rows {
		results[byRowID[presence.PeerRowId]] = true
	}
	return results, names, nil
}
