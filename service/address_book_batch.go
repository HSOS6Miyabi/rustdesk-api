package service

import (
	"errors"
	"fmt"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func addressBookBatchIDs(ids []uint) ([]uint, error) {
	if len(ids) == 0 || len(ids) > 200 {
		return nil, errors.New("select between 1 and 200 address book entries")
	}
	unique := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 { return nil, errors.New("address book entry ID is required") }
		if _, exists := unique[id]; exists { continue }
		unique[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// BatchRestoreAddressBookEntries restores all selected entries in one
// transaction. A missing, conflicting, or unauthorized entry cancels the batch.
func BatchRestoreAddressBookEntries(ownerID uint, ids []uint) (int, error) {
	if ownerID == 0 { return 0, errors.New("account ID is required") }
	selected, err := addressBookBatchIDs(ids)
	if err != nil { return 0, err }
	err = DB.Transaction(func(tx *gorm.DB) error {
		var entries []model.AddressBook
		if err := tx.Unscoped().Where("user_id = ? AND row_id IN ? AND deleted_at IS NOT NULL",
			ownerID, selected).Find(&entries).Error; err != nil {
			return err
		}
		if len(entries) != len(selected) {
			return errors.New("some entries are missing or unavailable for restoration")
		}
		seen := make(map[string]bool, len(entries))
		for _, entry := range entries {
			if entry.CollectionId > 0 {
				var exists int64
				if err := tx.Model(&model.AddressBookCollection{}).
					Where("id = ? AND user_id = ?", entry.CollectionId, ownerID).
					Count(&exists).Error; err != nil { return err }
				if exists != 1 { return errors.New("address book collection is missing") }
			}
			key := fmt.Sprintf("%d:%s", entry.CollectionId, entry.Id)
			if seen[key] { return errors.New("restored entries would duplicate a device") }
			seen[key] = true
			var conflicts int64
			if err := tx.Model(&model.AddressBook{}).
				Where("user_id = ? AND collection_id = ? AND id = ?",
					ownerID, entry.CollectionId, entry.Id).
				Count(&conflicts).Error; err != nil { return err }
			if conflicts > 0 { return errors.New("a current device already uses one of the selected IDs") }
		}
		result := tx.Unscoped().Model(&model.AddressBook{}).
			Where("user_id = ? AND row_id IN ? AND deleted_at IS NOT NULL", ownerID, selected).
			Update("deleted_at", nil)
		if result.Error != nil { return result.Error }
		if result.RowsAffected != int64(len(selected)) {
			return errors.New("the number of restored entries did not match the request")
		}
		return nil
	})
	if err != nil { return 0, err }
	return len(selected), nil
}

// BatchPermanentlyDeleteAddressBookEntries deletes all selected archived
// entries only after every entry has been checked for ownership and status.
func BatchPermanentlyDeleteAddressBookEntries(ownerID uint, ids []uint) (int, error) {
	if ownerID == 0 { return 0, errors.New("account ID is required") }
	selected, err := addressBookBatchIDs(ids)
	if err != nil { return 0, err }
	err = DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Unscoped().Model(&model.AddressBook{}).
			Where("user_id = ? AND row_id IN ? AND deleted_at IS NOT NULL",
				ownerID, selected).Count(&count).Error; err != nil { return err }
		if count != int64(len(selected)) {
			return errors.New("some entries are missing or unavailable for permanent deletion")
		}
		result := tx.Unscoped().Where("user_id = ? AND row_id IN ? AND deleted_at IS NOT NULL",
			ownerID, selected).Delete(&model.AddressBook{})
		if result.Error != nil { return result.Error }
		if result.RowsAffected != int64(len(selected)) {
			return errors.New("the number of permanently deleted entries did not match the request")
		}
		return nil
	})
	if err != nil { return 0, err }
	return len(selected), nil
}
