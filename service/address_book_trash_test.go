package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAddressBookTrashRestoreAndPermanentDeletion(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBook{},&model.AddressBookCollection{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	owner:=users["member"]
	other:=users["other"]
	book:=&model.AddressBookCollection{UserId:owner.Id,Name:"Home"}
	if err:=DB.Create(book).Error;err!=nil{t.Fatal(err)}
	entry:=&model.AddressBook{
		Id:"desk-445566",Alias:"Support",UserId:owner.Id,CollectionId:book.Id,
		Password:"connection-password",Tags:[]byte("[]"),
	}
	if err:=DB.Create(entry).Error;err!=nil{t.Fatal(err)}
	if err:=(&AddressBookService{}).Delete(entry);err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.AddressBook{}).Where("row_id = ?",entry.RowId).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted device visible in normal address book: %d %v",count,err)
	}
	trash,total,err:=ListTrashedAddressBooks(owner.Id,1,20)
	if err!=nil||total!=1||len(trash)!=1||trash[0].Id!=entry.Id||trash[0].DeletedAt<=0{
		t.Fatalf("address book trash did not contain deleted device: %d %v %v",total,trash,err)
	}
	if otherTrash,_,err:=ListTrashedAddressBooks(other.Id,1,20);err!=nil||len(otherTrash)!=0{
		t.Fatalf("other account can see deleted device: %v %v",otherTrash,err)
	}
	if err:=RestoreAddressBookEntry(other.Id,entry.RowId);err==nil{t.Fatal("another account restored the device")}
	if err:=RestoreAddressBookEntry(owner.Id,entry.RowId);err!=nil{t.Fatal(err)}
	var restored model.AddressBook
	if err:=DB.Where("row_id = ?",entry.RowId).First(&restored).Error;err!=nil||restored.Password!="connection-password"{
		t.Fatalf("restored device lost its settings: %+v %v",restored,err)
	}
	if err:=RestoreAddressBookEntry(owner.Id,entry.RowId);err==nil{t.Fatal("already active device restored again")}
	if err:=(&AddressBookService{}).Delete(&restored);err!=nil{t.Fatal(err)}
	duplicate:=&model.AddressBook{Id:entry.Id,UserId:owner.Id,CollectionId:book.Id,Tags:[]byte("[]")}
	if err:=DB.Create(duplicate).Error;err!=nil{t.Fatal(err)}
	if err:=RestoreAddressBookEntry(owner.Id,entry.RowId);err==nil{t.Fatal("restore overwrote an active device")}
	if err:=PermanentlyDeleteAddressBookEntry(other.Id,entry.RowId);err==nil{t.Fatal("another account purged the device")}
	if err:=PermanentlyDeleteAddressBookEntry(owner.Id,entry.RowId);err!=nil{t.Fatal(err)}
	var archived int64
	if err:=DB.Unscoped().Model(&model.AddressBook{}).Where("row_id = ?",entry.RowId).
		Count(&archived).Error;err!=nil||archived!=0{
		t.Fatalf("archived entry still exists after permanent deletion: %d %v",archived,err)
	}
	if err:=PermanentlyDeleteAddressBookEntry(owner.Id,entry.RowId);err==nil{t.Fatal("same entry purged twice")}
}

func TestDeletingCollectionRemovesItsArchivedDevices(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBook{},&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{});err!=nil{t.Fatal(err)}
	owner:=seedGroups(t)["member"]
	book:=&model.AddressBookCollection{UserId:owner.Id,Name:"Temporary"}
	if err:=DB.Create(book).Error;err!=nil{t.Fatal(err)}
	for _,name:=range []string{"one","two"}{
		entry:=&model.AddressBook{Id:name,UserId:owner.Id,CollectionId:book.Id,Tags:[]byte("[]")}
		if err:=DB.Create(entry).Error;err!=nil{t.Fatal(err)}
		if name=="one"{
			if err:=(&AddressBookService{}).Delete(entry);err!=nil{t.Fatal(err)}
		}
	}
	if err:=(&AddressBookService{}).DeleteCollection(book);err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Unscoped().Model(&model.AddressBook{}).Where("collection_id = ?",book.Id).
		Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted collection retained orphaned records: %d %v",count,err)
	}
}
