package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestBatchRestoreAndPurgeAddressBooksAreAtomic(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AddressBook{}, &model.AddressBookCollection{}); err != nil { t.Fatal(err) }
	users := seedGroups(t)
	owner, other := users["member"], users["other"]
	collection := &model.AddressBookCollection{UserId: owner.Id, Name: "Team books"}
	if err := DB.Create(collection).Error; err != nil { t.Fatal(err) }
	entries := []*model.AddressBook{
		{Id: "device-a", UserId: owner.Id, CollectionId: collection.Id, Tags: []byte("[]")},
		{Id: "device-b", UserId: owner.Id, CollectionId: collection.Id, Tags: []byte("[]")},
		{Id: "other-book", UserId: other.Id, Tags: []byte("[]")},
	}
	for _, entry := range entries {
		if err := DB.Create(entry).Error; err != nil { t.Fatal(err) }
		if err := (&AddressBookService{}).Delete(entry); err != nil { t.Fatal(err) }
	}
	if _, err := BatchRestoreAddressBookEntries(owner.Id, []uint{entries[0].RowId, 99999}); err == nil {
		t.Fatal("partial restore was accepted")
	}
	if _, err := BatchRestoreAddressBookEntries(owner.Id, []uint{entries[0].RowId, entries[2].RowId}); err == nil {
		t.Fatal("another account's archived entry was restored")
	}
	var active int64
	if err := DB.Model(&model.AddressBook{}).Where("user_id = ?", owner.Id).Count(&active).Error; err != nil || active != 0 {
		t.Fatalf("failed batch changed archived entries: %d %v", active, err)
	}
	if count, err := BatchRestoreAddressBookEntries(owner.Id, []uint{
		entries[0].RowId, entries[1].RowId, entries[0].RowId,
	}); err != nil || count != 2 {
		t.Fatalf("batch restore failed: %d %v", count, err)
	}
	if err := DB.Model(&model.AddressBook{}).Where("user_id = ?", owner.Id).Count(&active).Error; err != nil || active != 2 {
		t.Fatalf("restored entries not found: %d %v", active, err)
	}
	for _, entry := range entries[:2] {
		if err := (&AddressBookService{}).Delete(entry); err != nil { t.Fatal(err) }
	}
	conflict := &model.AddressBook{
		Id: entries[0].Id, UserId: owner.Id, CollectionId: collection.Id, Tags: []byte("[]"),
	}
	if err := DB.Create(conflict).Error; err != nil { t.Fatal(err) }
	if _, err := BatchRestoreAddressBookEntries(owner.Id, []uint{entries[0].RowId, entries[1].RowId}); err == nil {
		t.Fatal("conflicting restore was accepted")
	}
	if err := DB.Model(&model.AddressBook{}).Where("user_id = ?", owner.Id).Count(&active).Error; err != nil || active != 1 {
		t.Fatalf("conflicting batch partly restored entries: %d %v", active, err)
	}
	if _, err := BatchPermanentlyDeleteAddressBookEntries(owner.Id, []uint{entries[0].RowId, entries[2].RowId}); err == nil {
		t.Fatal("another account's device was permanently deleted")
	}
	if _, err := BatchPermanentlyDeleteAddressBookEntries(owner.Id, []uint{entries[0].RowId, 99999}); err == nil {
		t.Fatal("partial permanent deletion was accepted")
	}
	if count, err := BatchPermanentlyDeleteAddressBookEntries(owner.Id, []uint{entries[0].RowId, entries[1].RowId}); err != nil || count != 2 {
		t.Fatalf("batch permanent deletion failed: %d %v", count, err)
	}
	var deleted int64
	if err := DB.Unscoped().Model(&model.AddressBook{}).
		Where("row_id IN ?", []uint{entries[0].RowId, entries[1].RowId}).
		Count(&deleted).Error; err != nil || deleted != 0 {
		t.Fatalf("batch permanent deletion left archived entries: %d %v", deleted, err)
	}
	if err := DB.Model(&model.AddressBook{}).Where("row_id = ?", conflict.RowId).
		Count(&active).Error; err != nil || active != 1 {
		t.Fatalf("batch permanent deletion affected active entry: %d %v", active, err)
	}
}
