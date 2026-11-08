package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func distinctUserIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id != 0 {
			if _, found := seen[id]; !found {
				seen[id] = struct{}{}
				result = append(result, id)
			}
		}
	}
	return result
}

func (gs *GroupService) GroupIdsForUser(user *model.User) ([]uint, error) {
	if user == nil || user.Id == 0 {
		return nil, errors.New("user does not exist")
	}
	var additional []uint
	err := DB.Model(&model.GroupMembership{}).Where("user_id = ?", user.Id).Pluck("group_id", &additional).Error
	if err != nil {
		return nil, err
	}
	return distinctUserIDs(append([]uint{user.GroupId}, additional...)), nil
}

func (gs *GroupService) directoryGroups(user *model.User) ([]model.Group, bool, error) {
	ids, err := gs.GroupIdsForUser(user)
	if err != nil { return nil, false, err }
	if len(ids) == 0 { return nil, false, nil }
	var groups []model.Group
	if err := DB.Where("id IN ?", ids).Find(&groups).Error; err != nil {
		return nil, false, err
	}
	hasAnchor := false
	for _, group := range groups {
		if group.Type == model.GroupTypeAnchor { hasAnchor = true }
	}
	return groups, hasAnchor, nil
}

// VisibleUserIds filters only directory responses and never denies direct device-ID connections.
func (gs *GroupService) VisibleUserIds(user *model.User) ([]uint, error) {
	if user == nil || user.Id == 0 {
		return nil, errors.New("user does not exist")
	}
	groups, hasAnchor, err := gs.directoryGroups(user)
	if err != nil { return nil, err }
	// Group administrators follow public-anchor visibility in the RustDesk client.
	if !hasAnchor && user.IsAdmin != nil && *user.IsAdmin {
		var all []uint
		err := DB.Model(&model.User{}).Pluck("id", &all).Error
		return all, err
	}
	visible := []uint{user.Id}
	sharedIds, anchorIds := make([]uint, 0), make([]uint, 0)
	for _, group := range groups {
		switch group.Type {
		case model.GroupTypeShare:
			if !hasAnchor { sharedIds = append(sharedIds, group.Id) }
		case model.GroupTypeAnchor:
			anchorIds = append(anchorIds, group.Id)
		}
	}
	if len(sharedIds) > 0 {
		var primary, secondary []uint
		if err := DB.Model(&model.User{}).Where("group_id IN ?", sharedIds).
			Pluck("id", &primary).Error; err != nil { return nil, err }
		if err := DB.Model(&model.GroupMembership{}).Where("group_id IN ?", sharedIds).
			Pluck("user_id", &secondary).Error; err != nil { return nil, err }
		visible = append(visible, primary...)
		visible = append(visible, secondary...)
	}
	if len(anchorIds) > 0 {
		var anchors []uint
		if err := DB.Model(&model.GroupAnchor{}).Where("group_id IN ?", anchorIds).
			Pluck("user_id", &anchors).Error; err != nil { return nil, err }
		visible = append(visible, anchors...)
	}
	// An explicitly created contact grant is independent of the group configuration.
	var contactOwners []uint
	if err := DB.Model(&model.Contact{}).Where("recipient_user_id = ?", user.Id).
		Pluck("owner_user_id", &contactOwners).Error; err != nil { return nil, err }
	visible = append(visible, contactOwners...)
	return distinctUserIDs(visible), nil
}

func (gs *GroupService) MemberIds(groupId uint) ([]uint, error) {
	if gs.InfoById(groupId).Id == 0 {
		return nil, errors.New("group does not exist")
	}
	var primary, additional []uint
	if err := DB.Model(&model.User{}).Where("group_id = ?", groupId).Pluck("id", &primary).Error; err != nil {
		return nil, err
	}
	if err := DB.Model(&model.GroupMembership{}).Where("group_id = ?", groupId).
		Pluck("user_id", &additional).Error; err != nil {
		return nil, err
	}
	return distinctUserIDs(append(primary, additional...)), nil
}

func (gs *GroupService) AnchorIds(groupId uint) ([]uint, error) {
	group := gs.InfoById(groupId)
	if group.Id == 0 || group.Type != model.GroupTypeAnchor {
		return nil, errors.New("group is not an anchor group")
	}
	var ids []uint
	err := DB.Model(&model.GroupAnchor{}).Where("group_id = ?", groupId).Pluck("user_id", &ids).Error
	return ids, err
}

func confirmUsers(tx *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&model.User{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errors.New("one or more users do not exist")
	}
	return nil
}

func (gs *GroupService) ReplaceAnchors(groupId uint, ids []uint) error {
	group := gs.InfoById(groupId)
	if group.Id == 0 || group.Type != model.GroupTypeAnchor {
		return errors.New("group is not an anchor group")
	}
	ids = distinctUserIDs(ids)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := confirmUsers(tx, ids); err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", groupId).Delete(&model.GroupAnchor{}).Error; err != nil {
			return err
		}
		for _, userId := range ids {
			if err := tx.Create(&model.GroupAnchor{GroupId: groupId, UserId: userId}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func removeMemberships(tx *gorm.DB, groupId uint, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("group_id = ? AND user_id IN ?", groupId, ids).
		Delete(&model.GroupMembership{}).Error; err != nil {
		return err
	}
	// Every account retains its primary group. When a primary membership is
	// removed, assign the existing default group.
	if groupId == 1 {
		return nil
	}
	return tx.Model(&model.User{}).Where("group_id = ? AND id IN ?", groupId, ids).
		Update("group_id", 1).Error
}

func (gs *GroupService) ChangeMembers(groupId uint, ids []uint, action string) error {
	if gs.InfoById(groupId).Id == 0 {
		return errors.New("group does not exist")
	}
	if action != "add" && action != "remove" && action != "replace" {
		return errors.New("invalid member operation")
	}
	ids = distinctUserIDs(ids)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := confirmUsers(tx, ids); err != nil {
			return err
		}
		if action == "replace" {
			var currentPrimary, currentAdditional []uint
			if err := tx.Model(&model.User{}).Where("group_id = ?", groupId).
				Pluck("id", &currentPrimary).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.GroupMembership{}).Where("group_id = ?", groupId).
				Pluck("user_id", &currentAdditional).Error; err != nil {
				return err
			}
			keep := make(map[uint]bool, len(ids))
			for _, id := range ids { keep[id] = true }
			var obsolete []uint
			for _, id := range distinctUserIDs(append(currentPrimary, currentAdditional...)) {
				if !keep[id] { obsolete = append(obsolete, id) }
			}
			if err := removeMemberships(tx, groupId, obsolete); err != nil {
				return err
			}
		} else if action == "remove" {
			return removeMemberships(tx, groupId, ids)
		}
		for _, id := range ids {
			row := &model.GroupMembership{GroupId: groupId, UserId: id}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ParseGroupIDs(value string) ([]uint, error) {
	if strings.TrimSpace(value) == "" {
		return []uint{}, nil
	}
	ids := make([]uint, 0)
	for _, part := range strings.Split(value, ",") {
		id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32)
		if err != nil || id == 0 {
			return nil, fmt.Errorf("invalid group ID %q", part)
		}
		ids = append(ids, uint(id))
	}
	return distinctUserIDs(ids), nil
}

func (gs *GroupService) ValidateAutoGroupIDs(value string) ([]uint, error) {
	ids, err := ParseGroupIDs(value)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if gs.InfoById(id).Id == 0 {
			return nil, fmt.Errorf("group %d does not exist", id)
		}
	}
	return ids, nil
}

// AutoJoinOAuthGroups runs after a successful, verified OAuth/OIDC login.
func (gs *GroupService) AutoJoinOAuthGroups(userId uint, provider string) error {
	oauth := AllService.OauthService.InfoByOp(provider)
	if oauth == nil || oauth.Id == 0 || strings.TrimSpace(oauth.AutoGroupIds) == "" {
		return nil
	}
	ids, err := gs.ValidateAutoGroupIDs(oauth.AutoGroupIds)
	if err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := confirmUsers(tx, []uint{userId}); err != nil {
			return err
		}
		for _, id := range ids {
			row := &model.GroupMembership{GroupId: id, UserId: userId}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
