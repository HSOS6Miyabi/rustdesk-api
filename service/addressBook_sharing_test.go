package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAddressBookSharingAdditionalGroupsAndEveryone(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBookCollection{},&model.AddressBookCollectionRule{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	owner,member,other:=users["anchor"],users["member"],users["other"]
	book:=&model.AddressBookCollection{Name:"Team knowledge",UserId:owner.Id}
	if err:=DB.Create(book).Error;err!=nil{t.Fatal(err)}
	svc:=&AddressBookService{}
	check:=func(user *model.User,want int) {
		t.Helper()
		actual:=svc.UserMaxRule(user,owner.Id,book.Id)
		if actual!=want{t.Fatalf("%s: expected %d, got %d",user.Username,want,actual)}
	}
	check(member,0)
	if err:=DB.Create(&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleReadWrite,
		Type:model.ShareAddressBookRuleTypeAll,ToId:0,
	}).Error;err!=nil{t.Fatal(err)}
	check(other,model.ShareAddressBookRuleRuleReadWrite)
	check(member,model.ShareAddressBookRuleRuleReadWrite)
	if err:=AllService.GroupService.ChangeMembers(2,[]uint{member.Id},"add");err!=nil{t.Fatal(err)}
	groupRule:=&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleFullControl,
		Type:model.ShareAddressBookRuleTypeGroup,ToId:2,
	}
	if err:=DB.Create(groupRule).Error;err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleFullControl)
	check(other,model.ShareAddressBookRuleRuleReadWrite)
	newGroup:=&model.Group{Name:"secondary",Type:model.GroupTypeShare}
	if err:=DB.Create(newGroup).Error;err!=nil{t.Fatal(err)}
	if err:=AllService.GroupService.ChangeMembers(newGroup.Id,[]uint{member.Id},"add");err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleRead,
		Type:model.ShareAddressBookRuleTypeGroup,ToId:newGroup.Id,
	}).Error;err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleFullControl)
	personal:=&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleRead,
		Type:model.ShareAddressBookRuleTypePersonal,ToId:member.Id,
	}
	if err:=DB.Create(personal).Error;err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleRead)
	rules:=svc.CollectionReadRules(member)
	if len(rules)!=4{t.Fatalf("want personal, two groups and everyone; got %d rules",len(rules))}
	if !svc.CheckUserReadPrivilege(member,owner.Id,book.Id) ||
		svc.CheckUserWritePrivilege(member,owner.Id,book.Id) {
		t.Fatal("explicit personal read-only permission did not override group rule")
	}
	if err:=DB.Delete(personal).Error;err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleFullControl)
	if err:=AllService.GroupService.ChangeMembers(2,[]uint{member.Id},"remove");err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleRead)
	if err:=AllService.GroupService.ChangeMembers(newGroup.Id,[]uint{member.Id},"remove");err!=nil{t.Fatal(err)}
	check(member,model.ShareAddressBookRuleRuleReadWrite)
	if list:=svc.CollectionReadRules(member);len(list)!=1 {
		t.Fatalf("after leaving groups only everyone rule should remain, got %d",len(list))
	}
	check(owner,model.ShareAddressBookRuleRuleFullControl)
	if svc.UserMaxRule(nil,owner.Id,book.Id)!=0{t.Fatal("nil account granted access")}
}

func TestAddressBookShareRuleUpdateCanSelectEveryone(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBookCollection{},&model.AddressBookCollectionRule{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	book:=&model.AddressBookCollection{UserId:users["anchor"].Id,Name:"Operations"}
	if err:=DB.Create(book).Error;err!=nil{t.Fatal(err)}
	svc:=&AddressBookService{}
	share:=&model.AddressBookCollectionRule{
		UserId:book.UserId,CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleFullControl,
		Type:model.ShareAddressBookRuleTypePersonal,ToId:users["member"].Id,
	}
	if err:=DB.Create(share).Error;err!=nil{t.Fatal(err)}
	share.Type=model.ShareAddressBookRuleTypeAll
	share.ToId=0
	share.Rule=model.ShareAddressBookRuleRuleRead
	if err:=svc.UpdateRule(share);err!=nil{t.Fatal(err)}
	stored:=svc.RuleInfoByOwnerAndTarget(book.UserId,model.ShareAddressBookRuleTypeAll,0,book.Id)
	if stored.Id!=share.Id||stored.Rule!=model.ShareAddressBookRuleRuleRead {
		t.Fatalf("global rule was not saved with to_id=0: %+v",stored)
	}
	if granted:=svc.UserMaxRule(users["other"],book.UserId,book.Id);granted!=model.ShareAddressBookRuleRuleRead{
		t.Fatalf("all users should have read access, got %d",granted)
	}
	wrongOwner:=&model.AddressBookCollectionRule{
		IdModel:model.IdModel{Id:share.Id},UserId:users["other"].Id,
		CollectionId:book.Id,Rule:model.ShareAddressBookRuleRuleFullControl,
		Type:model.ShareAddressBookRuleTypeAll,ToId:0,
	}
	if err:=svc.UpdateRule(wrongOwner);err!=nil{t.Fatal(err)}
	stored=svc.RuleInfoByOwnerAndTarget(book.UserId,model.ShareAddressBookRuleTypeAll,0,book.Id)
	if stored.Rule!=model.ShareAddressBookRuleRuleRead{
		t.Fatalf("another owner modified the sharing permission: %+v",stored)
	}
}
