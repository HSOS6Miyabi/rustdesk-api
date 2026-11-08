package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func userDeletionTestDB(t *testing.T) (map[string]*model.User, func()) {
	t.Helper()
	cleanup:=setupGroupDB(t)
	if err:=DB.AutoMigrate(
		&model.UserThird{},&model.AddressBook{},&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},&model.Tag{},
		&model.UserToken{},&model.ShareRecord{},
		&model.AuditConn{},&model.AuditFile{},
	);err!=nil{
		cleanup()
		t.Fatal(err)
	}
	AllService.UserService=&UserService{}
	return seedGroups(t),cleanup
}

func TestDeleteUserRevokesContactsAndGroupMemberships(t *testing.T) {
	users,cleanup:=userDeletionTestDB(t)
	defer cleanup()
	owner,other,admin:=users["member"],users["other"],users["admin"]

	group,err:=AllService.GroupService.CreateOwnedGroup(owner.Id,"owned",model.GroupTypeShare)
	if err!=nil {t.Fatal(err)}
	if err:=DB.Model(other).Update("group_id",group.Id).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.GroupMembership{UserId:owner.Id,GroupId:2}).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.GroupAnchor{UserId:owner.Id,GroupId:3}).Error;err!=nil{t.Fatal(err)}
	for _,record:=range []model.Contact{
		{OwnerUserId:owner.Id,RecipientUserId:other.Id},
		{OwnerUserId:admin.Id,RecipientUserId:owner.Id},
	}{
		if err:=DB.Create(&record).Error;err!=nil{t.Fatal(err)}
	}
	peer:=&model.Peer{Id:"owned-device",UserId:owner.Id,Hostname:"Owner's PC"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.UserToken{UserId:owner.Id,Token:"old-session",ExpiredAt:4102444800}).Error;err!=nil{
		t.Fatal(err)
	}
	if err:=DB.Create(&model.AuditConn{
		ConnId:2,PeerId:"owned-device",FromPeer:"remote-peer",
		FromUserId:other.Id,ToUserId:owner.Id,FromAccount:other.Username,ToAccount:owner.Username,
	}).Error;err!=nil{t.Fatal(err)}
	for _,r:=range []model.AddressBookCollectionRule{
		{UserId:admin.Id,Type:model.ShareAddressBookRuleTypeGroup,ToId:group.Id,CollectionId:100,Rule:1},
		{UserId:admin.Id,Type:model.ShareAddressBookRuleTypePersonal,ToId:owner.Id,CollectionId:101,Rule:1},
	}{
		if err:=DB.Create(&r).Error;err!=nil{t.Fatal(err)}
	}

	if err:=AllService.UserService.Delete(owner);err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.User{}).Where("id = ?",owner.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted account still exists: %d %v",count,err)
	}
	if err:=DB.Model(&model.Group{}).Where("owner_user_id = ?",owner.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("owned group remains: %d %v",count,err)
	}
	if err:=DB.Model(&model.GroupMembership{}).
		Where("user_id = ? OR group_id = ?",owner.Id,group.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("group membership remains: %d %v",count,err)
	}
	if err:=DB.Model(&model.GroupAnchor{}).Where("user_id = ? OR group_id = ?",owner.Id,group.Id).
		Count(&count).Error;err!=nil||count!=0{t.Fatalf("public anchor remains: %d %v",count,err)}
	if err:=DB.Model(&model.Contact{}).Where("owner_user_id = ? OR recipient_user_id = ?",owner.Id,owner.Id).
		Count(&count).Error;err!=nil||count!=0{t.Fatalf("contact grant remains: %d %v",count,err)}
	if err:=DB.Model(&model.UserToken{}).Where("user_id = ?",owner.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("stale login session remains: %d %v",count,err)
	}
	if err:=DB.Model(&model.AddressBookCollectionRule{}).Where(
		"(type = ? AND to_id = ?) OR (type = ? AND to_id = ?)",
		model.ShareAddressBookRuleTypeGroup,group.Id,
		model.ShareAddressBookRuleTypePersonal,owner.Id,
	).Count(&count).Error;err!=nil||count!=0{t.Fatalf("stale shared address book grant remains: %d %v",count,err)}
	var preservedPeer model.Peer
	if err:=DB.Where("id = ?",peer.Id).First(&preservedPeer).Error;err!=nil||preservedPeer.UserId!=0 {
		t.Fatalf("device was not detached: %+v %v",preservedPeer,err)
	}
	var remainingUser model.User
	if err:=DB.First(&remainingUser,other.Id).Error;err!=nil||remainingUser.GroupId!=1{
		t.Fatalf("other group member lost account or primary group: %+v %v",remainingUser,err)
	}
	if err:=DB.Model(&model.AuditConn{}).Where("peer_id = ?",peer.Id).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("historical connection record was changed: %d %v",count,err)
	}
}

func TestDeleteUserRollsBackWhenAssociationCleanupFails(t *testing.T) {
	users,cleanup:=userDeletionTestDB(t)
	defer cleanup()
	owner:=users["member"]
	group,err:=AllService.GroupService.CreateOwnedGroup(owner.Id,"atomic",model.GroupTypeShare)
	if err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.Contact{OwnerUserId:owner.Id,RecipientUserId:users["other"].Id}).Error;err!=nil{
		t.Fatal(err)
	}
	if err:=DB.Exec("CREATE TRIGGER reject_contact_delete BEFORE DELETE ON contacts BEGIN SELECT RAISE(FAIL, 'blocked'); END").Error;err!=nil{
		t.Fatal(err)
	}
	defer DB.Exec("DROP TRIGGER IF EXISTS reject_contact_delete")
	if err:=AllService.UserService.Delete(owner);err==nil{
		t.Fatal("user deletion committed despite failed association cleanup")
	}
	var ownerCount,groupCount int64
	if err:=DB.Model(&model.User{}).Where("id = ?",owner.Id).Count(&ownerCount).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Model(&model.Group{}).Where("id = ?",group.Id).Count(&groupCount).Error;err!=nil{t.Fatal(err)}
	if ownerCount!=1||groupCount!=1{
		t.Fatalf("transaction partially committed: user=%d group=%d",ownerCount,groupCount)
	}
}
