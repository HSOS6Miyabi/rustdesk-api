package service

import (
	"errors"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

var ErrGroupOwnership = errors.New("group does not exist or is not owned by this user")

func validGroupType(kind int) bool {
	return kind == model.GroupTypeDefault || kind == model.GroupTypeShare || kind == model.GroupTypeAnchor
}

func validGroupName(name string) bool {
	n := len([]rune(strings.TrimSpace(name)))
	return n >= 1 && n <= 100
}

func (gs *GroupService) OwnedGroup(ownerId, groupId uint) (*model.Group, error) {
	if ownerId == 0 || groupId == 0 {
		return nil, ErrGroupOwnership
	}
	group := &model.Group{}
	if err := DB.Where("id = ? AND owner_user_id = ?", groupId, ownerId).First(group).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGroupOwnership
		}
		return nil, err
	}
	return group, nil
}

func (gs *GroupService) OwnedGroups(ownerId, page, pageSize uint) *model.GroupList {
	return gs.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("owner_user_id = ?", ownerId).Order("id DESC")
	})
}

func (gs *GroupService) CreateOwnedGroup(ownerId uint, name string, kind int) (*model.Group, error) {
	if ownerId == 0 || !validGroupName(name) || !validGroupType(kind) {
		return nil, errors.New("invalid group configuration")
	}
	user := &model.User{}
	if err := DB.Where("id = ? AND status = ?", ownerId, model.COMMON_STATUS_ENABLE).First(user).Error; err != nil {
		return nil, errors.New("owner account is unavailable")
	}
	group := &model.Group{Name: strings.TrimSpace(name), Type: kind, OwnerUserId: ownerId}
	if err := gs.Create(group); err != nil {
		return nil, err
	}
	// The creator belongs to the new group without losing their original primary group.
	if err := gs.ChangeMembers(group.Id, []uint{ownerId}, "add"); err != nil {
		_ = gs.Delete(group)
		return nil, err
	}
	return group, nil
}

func (gs *GroupService) UpdateOwnedGroup(ownerId, groupId uint, name string, kind int) error {
	if !validGroupName(name) || !validGroupType(kind) {
		return errors.New("invalid group configuration")
	}
	_, err := gs.OwnedGroup(ownerId, groupId)
	if err != nil { return err }
	result := DB.Model(&model.Group{}).Where("id = ? AND owner_user_id = ?", groupId, ownerId).
		Updates(map[string]interface{}{"name": strings.TrimSpace(name), "type": kind})
	if result.Error != nil { return result.Error }
	if result.RowsAffected == 0 { return ErrGroupOwnership }
	return nil
}

func (gs *GroupService) DeleteOwnedGroup(ownerId, groupId uint) error {
	group, err := gs.OwnedGroup(ownerId, groupId)
	if err != nil { return err }
	return gs.Delete(group)
}

func (gs *GroupService) OwnedGroupMembers(ownerId, groupId uint) ([]uint, error) {
	if _, err := gs.OwnedGroup(ownerId, groupId); err != nil { return nil, err }
	return gs.MemberIds(groupId)
}

func (gs *GroupService) OwnedGroupAnchors(ownerId, groupId uint) ([]uint, error) {
	if _, err := gs.OwnedGroup(ownerId, groupId); err != nil { return nil, err }
	return gs.AnchorIds(groupId)
}

func verifyNonAdminMembers(ids []uint) error {
	if len(ids) == 0 { return nil }
	var count int64
	if err := DB.Model(&model.User{}).Where("id IN ? AND is_admin = ?", distinctUserIDs(ids), true).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 { return errors.New("only system administrators can change administrator group memberships") }
	return nil
}

func (gs *GroupService) ChangeOwnedMembers(ownerId, groupId uint, ids []uint, action string) error {
	if _, err := gs.OwnedGroup(ownerId, groupId); err != nil { return err }
	if action != "add" && action != "remove" && action != "replace" {
		return errors.New("invalid member operation")
	}
	if action != "remove" {
		if err := verifyNonAdminMembers(ids); err != nil { return err }
	}
	if action == "replace" {
		// The owner must remain a member of a group that they manage.
		ids = append(distinctUserIDs(ids), ownerId)
	}
	if action == "remove" {
		for _, id := range ids {
			if id == ownerId { return errors.New("the owner cannot remove themselves from their group") }
		}
	}
	return gs.ChangeMembers(groupId, ids, action)
}

func (gs *GroupService) ChangeOwnedAnchors(ownerId, groupId uint, ids []uint) error {
	if _, err := gs.OwnedGroup(ownerId, groupId); err != nil { return err }
	return gs.ReplaceAnchors(groupId, ids)
}
