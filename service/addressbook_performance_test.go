package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func TestAddressBookSyncDoesNotRewriteIdenticalEntries(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBook{});err!=nil{t.Fatal(err)}
	const userID uint=42
	original:=&model.AddressBook{Id:"unchanged-001",UserId:userID,Hostname:"Desk",Username:"owner",Platform:"Linux",Alias:"First"}
	if err:=DB.Create(original).Error;err!=nil{t.Fatal(err)}
	var stored model.AddressBook
	if err:=DB.Where("row_id = ?",original.RowId).First(&stored).Error;err!=nil{t.Fatal(err)}
	updates:=0
	if err:=DB.Callback().Update().Before("gorm:update").
		Register("perf:addressbook_update",func(*gorm.DB){updates++});err!=nil{t.Fatal(err)}
	copyEntry:=stored
	if err:=(&AddressBookService{}).UpdateAddressBook([]*model.AddressBook{&copyEntry},userID);err!=nil{t.Fatal(err)}
	if updates!=0 {t.Fatalf("unchanged sync executed %d SQL updates",updates)}
	copyEntry.Alias="Changed"
	if err:=(&AddressBookService{}).UpdateAddressBook([]*model.AddressBook{&copyEntry},userID);err!=nil{t.Fatal(err)}
	if updates!=1 {t.Fatalf("one changed entry executed %d updates",updates)}
	if err:=DB.Callback().Update().Remove("perf:addressbook_update");err!=nil{t.Fatal(err)}
	var restored model.AddressBook
	if err:=DB.First(&restored,original.RowId).Error;err!=nil{t.Fatal(err)}
	if restored.Alias!="Changed" {t.Fatalf("updated entry not saved: %+v",restored)}
}
