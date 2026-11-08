package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

type SharedDeviceSource struct {
	Type string `json:"type"`
	Id uint `json:"id"`
	Name string `json:"name"`
}

type SharedDevice struct {
	model.Peer
	OwnerUsername string `json:"owner_username"`
	Sources []SharedDeviceSource `json:"sources"`
}

type SharedDevicePage struct {
	List []SharedDevice `json:"list"`
	Total int64 `json:"total"`
	Page uint `json:"page"`
	PageSize uint `json:"page_size"`
}

func sharedOwnerSources(user *model.User) (map[uint][]SharedDeviceSource, error) {
	owners := make(map[uint][]SharedDeviceSource)
	groups, err := AllService.GroupService.GroupIdsForUser(user)
	if err != nil { return nil, err }
	for _, id := range groups {
		group := AllService.GroupService.InfoById(id)
		if group.Id == 0 { continue }
		var ids []uint
		switch group.Type {
		case model.GroupTypeShare:
			ids, err = AllService.GroupService.MemberIds(id)
		case model.GroupTypeAnchor:
			ids, err = AllService.GroupService.AnchorIds(id)
		default:
			continue
		}
		if err != nil { return nil, err }
		for _, owner := range ids {
			if owner == user.Id { continue }
			owners[owner] = append(owners[owner], SharedDeviceSource{
				Type: "group", Id: id, Name: group.Name,
			})
		}
	}
	var contacts []model.Contact
	if err := DB.Where("recipient_user_id = ?", user.Id).Find(&contacts).Error; err != nil {
		return nil, err
	}
	for _, grant := range contacts {
		if grant.OwnerUserId == user.Id { continue }
		owners[grant.OwnerUserId] = append(owners[grant.OwnerUserId], SharedDeviceSource{
			Type: "contact", Id: grant.OwnerUserId, Name: "联系人",
		})
	}
	return owners, nil
}

func (ps *PeerService) SharedDeviceList(user *model.User, page, size uint, filter, sourceType string, sourceId uint) (*SharedDevicePage, error) {
	if user == nil || user.Id == 0 {
		return nil, fmt.Errorf("user is required")
	}
	if page == 0 { page = 1 }
	if size == 0 { size = 20 }
	if size > 100 { size = 100 }
	if sourceType != "" && sourceType != "all" && sourceType != "contact" && sourceType != "group" {
		return nil, fmt.Errorf("unsupported source type")
	}
	owners, err := sharedOwnerSources(user)
	if err != nil { return nil, err }
	for id, sources := range owners {
		filtered := make([]SharedDeviceSource, 0, len(sources))
		for _, source := range sources {
			if sourceType != "" && sourceType != "all" && source.Type != sourceType { continue }
			if sourceId != 0 && source.Id != sourceId { continue }
			filtered = append(filtered, source)
		}
		if len(filtered) == 0 {
			delete(owners, id)
		} else { owners[id] = filtered }
	}
	result := &SharedDevicePage{List: []SharedDevice{}, Page: page, PageSize: size}
	if len(owners) == 0 { return result, nil }
	ids := make([]uint, 0, len(owners))
	for id := range owners { ids = append(ids, id) }
	sort.Slice(ids, func(i,j int) bool { return ids[i] < ids[j] })
	query := DB.Model(&model.Peer{}).Where("user_id IN ?", ids)
	if search := strings.TrimSpace(filter); search != "" {
		search = "%" + search + "%"
		query = query.Where("(id LIKE ? OR hostname LIKE ? OR alias LIKE ?)", search, search, search)
	}
	if err := query.Count(&result.Total).Error; err != nil { return nil, err }
	var peers []model.Peer
	if err := query.Order("last_online_time DESC, row_id DESC").
		Offset(int((page-1)*size)).Limit(int(size)).Find(&peers).Error; err != nil { return nil, err }
	var users []model.User
	if err := DB.Select("id, username").Where("id IN ?", ids).Find(&users).Error; err != nil { return nil, err }
	names := make(map[uint]string, len(users))
	for _, account := range users { names[account.Id] = account.Username }
	for _, peer := range peers {
		sources := owners[peer.UserId]
		for i := range sources {
			if sources[i].Type == "contact" { sources[i].Name = names[peer.UserId] }
		}
		result.List = append(result.List, SharedDevice{
			Peer: peer,
			OwnerUsername: names[peer.UserId],
			Sources: sources,
		})
	}
	return result, nil
}
