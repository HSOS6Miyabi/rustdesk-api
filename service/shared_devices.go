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
	if user == nil || user.Id == 0 { return nil, fmt.Errorf("user is required") }
	groups, hasAnchor, err := AllService.GroupService.directoryGroups(user)
	if err != nil { return nil, err }
	groupNames := make(map[uint]string, len(groups))
	shareIds, anchorIds := make([]uint, 0), make([]uint, 0)
	for _, group := range groups {
		groupNames[group.Id] = group.Name
		switch group.Type {
		case model.GroupTypeShare:
			if !hasAnchor { shareIds = append(shareIds, group.Id) }
		case model.GroupTypeAnchor:
			anchorIds = append(anchorIds, group.Id)
		}
	}
	seenGroup := make(map[uint]map[uint]bool)
	addGroup := func(owner, groupId uint) {
		if owner == 0 || owner == user.Id { return }
		if seenGroup[owner] == nil { seenGroup[owner] = make(map[uint]bool) }
		if seenGroup[owner][groupId] { return }
		seenGroup[owner][groupId] = true
		owners[owner] = append(owners[owner], SharedDeviceSource{
			Type:"group", Id:groupId, Name:groupNames[groupId],
		})
	}
	if len(shareIds) > 0 {
		var primary []struct{ Id uint; GroupId uint }
		if err := DB.Model(&model.User{}).Select("id,group_id").
			Where("group_id IN ?", shareIds).Scan(&primary).Error; err != nil { return nil, err }
		for _, row := range primary { addGroup(row.Id, row.GroupId) }
		var extra []model.GroupMembership
		if err := DB.Where("group_id IN ?", shareIds).Find(&extra).Error; err != nil { return nil, err }
		for _, row := range extra { addGroup(row.UserId, row.GroupId) }
	}
	if len(anchorIds) > 0 {
		var anchors []model.GroupAnchor
		if err := DB.Where("group_id IN ?", anchorIds).Find(&anchors).Error; err != nil { return nil, err }
		for _, row := range anchors { addGroup(row.UserId, row.GroupId) }
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

func (ps *PeerService) SharedDeviceList(user *model.User, page, size uint, filter, sourceType string, sourceId uint, sortArgs ...string) (*SharedDevicePage, error) {
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
	var users []model.User
	if err := DB.Select("id, username").Where("id IN ?", ids).Find(&users).Error; err != nil { return nil, err }
	names := make(map[uint]string, len(users))
	for _, account := range users { names[account.Id] = account.Username }

	var sortField, sortDir string
	if len(sortArgs)>=2 { sortField=sortArgs[0];sortDir=strings.ToLower(sortArgs[1]) }
	sortValid:=sortDir=="asc"||sortDir=="desc"
	query := DB.Model(&model.Peer{}).Where("peers.user_id IN ?", ids)
	if sortValid&&sortField=="owner_username" {
		query=query.Joins("LEFT JOIN users AS owners ON owners.id = peers.user_id").Select("peers.*")
	}
	if search := strings.TrimSpace(filter); search != "" {
		search = "%" + search + "%"
		query = query.Where("(peers.id LIKE ? OR peers.hostname LIKE ? OR peers.alias LIKE ?)", search, search, search)
	}
	if err := query.Count(&result.Total).Error; err != nil { return nil, err }
	ordered:=query.Order("peers.last_online_time DESC, peers.row_id DESC")
	if sortValid&&sortField=="owner_username" {
		ApplyComputedSort(ordered,"LOWER(owners.username)",nil,sortDir,"peers.row_id")
	}
	if sortValid&&sortField=="sources" {
		var sql strings.Builder
		sql.WriteString("CASE peers.user_id")
		params:=make([]interface{},0,len(ids)*2)
		for _,id:=range ids {
			sources:=owners[id]
			sourceLabels:=make([]string,0,len(sources))
			for _,source:=range sources {
				name:=source.Name
				if source.Type=="contact" {name=names[id]}
				sourceLabels=append(sourceLabels,source.Type+":"+name)
			}
			sort.Strings(sourceLabels)
			sql.WriteString(" WHEN ? THEN ?")
			params=append(params,id,strings.ToLower(strings.Join(sourceLabels,"、")))
		}
		sql.WriteString(" ELSE '' END")
		ApplyComputedSort(ordered,sql.String(),params,sortDir,"peers.row_id")
	}
	var peers []model.Peer
	if err := ordered.Scopes(SortScope(sortArgs...)).
		Offset(int((page-1)*size)).Limit(int(size)).Find(&peers).Error; err != nil { return nil, err }
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

func (ps *PeerService) SharedDeviceSources(user *model.User) ([]SharedDeviceSource, error) {
	owners, err := sharedOwnerSources(user)
	if err != nil { return nil, err }
	all := make(map[string]SharedDeviceSource)
	var ids []uint
	for id := range owners { ids = append(ids, id) }
	var users []model.User
	if len(ids) > 0 {
		if err := DB.Select("id,username").Where("id IN ?", ids).Find(&users).Error; err != nil { return nil, err }
	}
	nameByID := make(map[uint]string,len(users))
	for _, u := range users { nameByID[u.Id] = u.Username }
	for owner, sources := range owners {
		for _, source := range sources {
			if source.Type == "contact" { source.Name = nameByID[owner] }
			key := fmt.Sprintf("%s:%d", source.Type, source.Id)
			all[key] = source
		}
	}
	result := make([]SharedDeviceSource, 0, len(all))
	for _, source := range all { result = append(result, source) }
	sort.Slice(result,func(i,j int) bool {
		if result[i].Type != result[j].Type { return result[i].Type < result[j].Type }
		return result[i].Name < result[j].Name
	})
	return result,nil
}

// SharedDeviceOwnerIDs returns the account IDs whose device inventories the user may access.
func SharedDeviceOwnerIDs(user *model.User) ([]uint,error) {
 owners,err:=sharedOwnerSources(user)
 if err!=nil{return nil,err}
 ids:=make([]uint,0,len(owners))
 for id:=range owners{ids=append(ids,id)}
 sort.Slice(ids,func(i,j int)bool{return ids[i]<ids[j]})
 return ids,nil
}
