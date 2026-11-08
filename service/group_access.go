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

// VisibleUserIds only controls the account and device directory exposed by API endpoints.
// Connecting directly to a device using its known ID remains unrestricted by group visibility.
func (gs *GroupService) VisibleUserIds(user *model.User) ([]uint, error) {
	if user == nil || user.Id == 0 {
		return nil, errors.New("user does not exist")
	}
	groups, err := gs.GroupIdsForUser(user)
	if err != nil { return nil, err }
	hasAnchorGroup := false
	for _, id := range groups {
		if group := gs.InfoById(id); group.Id != 0 && group.Type == model.GroupTypeAnchor {
			hasAnchorGroup = true
		}
	}
	// Even an administrator who belongs to an anchor group sees only the
	// explicitly selected public users in RustDesk client directory results.
	if !hasAnchorGroup && user.IsAdmin != nil && *user.IsAdmin {
		var all []uint
		err := DB.Model(&model.User{}).Pluck("id", &all).Error
		return all, err
	}
	visible := []uint{user.Id}
	for _, id := range groups {
		group := gs.InfoById(id)
		if group.Id == 0 { continue }
		var members []uint
		switch group.Type {
		case model.GroupTypeShare:
			// Anchor groups take precedence over broad group visibility.
			if hasAnchorGroup { continue }
			members, err = gs.MemberIds(id)
		case model.GroupTypeAnchor:
			members, err = gs.AnchorIds(id)
		default:
			continue
		}
		if err != nil { return nil, err }
		visible = append(visible, members...)
	}
	// Contact exposure is independent of group visibility and explicitly
	// granted by each device owner.
	var contacts []uint
	if err := DB.Model(&model.Contact{}).Where("recipient_user_id = ?", user.Id).
		Pluck("owner_user_id", &contacts).Error; err != nil { return nil, err }
	visible = append(visible, contacts...)
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
