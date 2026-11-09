package service

import (
	"errors"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type TrashedAddressBook struct {
	RowId uint `json:"row_id"`
	Id string `json:"id"`
	Alias string `json:"alias"`
	Hostname string `json:"hostname"`
	CollectionId uint `json:"collection_id"`
	DeletedAt int64 `json:"deleted_at"`
}

func ListTrashedAddressBooks(ownerID uint, page,pageSize uint) ([]TrashedAddressBook,int64,error) {
	if ownerID == 0 { return nil,0,errors.New("account ID is required") }
	query:=DB.Unscoped().Model(&model.AddressBook{}).
		Where("user_id = ? AND deleted_at IS NOT NULL",ownerID)
	var total int64
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	var entries []model.AddressBook
	if err:=query.Order("deleted_at DESC, row_id DESC").
		Scopes(Paginate(page,pageSize)).Find(&entries).Error;err!=nil{return nil,0,err}
	rows:=make([]TrashedAddressBook,0,len(entries))
	for _,entry:=range entries{
		rows=append(rows,TrashedAddressBook{
			RowId:entry.RowId,Id:entry.Id,Alias:entry.Alias,
			Hostname:entry.Hostname,CollectionId:entry.CollectionId,
			DeletedAt:entry.DeletedAt.Time.Unix(),
		})
	}
	return rows,total,nil
}

// RestoreAddressBookEntry restores an archived entry for its original owner.
func RestoreAddressBookEntry(ownerID,rowID uint) error {
	if ownerID==0||rowID==0{return errors.New("account and entry IDs are required")}
	return DB.Transaction(func(tx *gorm.DB) error {
		var entry model.AddressBook
		if err:=tx.Unscoped().Where("row_id = ? AND user_id = ?",rowID,ownerID).
			First(&entry).Error;err!=nil{return err}
		if !entry.DeletedAt.Valid{return errors.New("entry is already active")}
		if entry.CollectionId>0{
			var count int64
			if err:=tx.Model(&model.AddressBookCollection{}).
				Where("id = ? AND user_id = ?",entry.CollectionId,ownerID).
				Count(&count).Error;err!=nil{return err}
			if count!=1{return errors.New("address book collection no longer exists")}
		}
		var count int64
		if err:=tx.Model(&model.AddressBook{}).
			Where("user_id = ? AND collection_id = ? AND id = ?",
				ownerID,entry.CollectionId,entry.Id).
			Count(&count).Error;err!=nil{return err}
		if count!=0{return errors.New("a current entry already uses this device ID")}
		result:=tx.Unscoped().Model(&model.AddressBook{}).
			Where("row_id = ? AND user_id = ? AND deleted_at IS NOT NULL",rowID,ownerID).
			Update("deleted_at",nil)
		if result.Error!=nil{return result.Error}
		if result.RowsAffected==0{return gorm.ErrRecordNotFound}
		return nil
	})
}

func PermanentlyDeleteAddressBookEntry(ownerID,rowID uint) error {
	if ownerID==0||rowID==0{return errors.New("account and entry IDs are required")}
	result:=DB.Unscoped().Where("row_id = ? AND user_id = ? AND deleted_at IS NOT NULL",
		rowID,ownerID).Delete(&model.AddressBook{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}
