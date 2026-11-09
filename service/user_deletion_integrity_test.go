package service

import (
	"testing"
	"strconv"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func userDeletionTestDB(t *testing.T) (map[string]*model.User, func()) {
	t.Helper()
	cleanup:=setupGroupDB(t)
	if err:=DB.AutoMigrate(
		&model.UserThird{},&model.AddressBook{},&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},&model.Tag{},
		&model.UserToken{},&model.ShareRecord{},&model.APIToken{},
		&model.UserTOTP{},&model.TOTPRecoveryCode{},&model.TOTPLoginChallenge{},
		&model.PasswordResetToken{},&model.UserInvitation{},&model.ControlRole{},&model.ControlRoleAssignment{},
		&model.AdminRole{},&model.AdminRoleAssignment{},
		&model.EmailLoginSetting{},&model.EmailCodeChallenge{},
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
	role:=&model.ControlRole{Name:"Account control",Keyboard:1}
	if err:=DB.Create(role).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.ControlRoleAssignment{
		TargetType:"user",TargetId:strconv.FormatUint(uint64(owner.Id),10),RoleId:role.Id,
	}).Error;err!=nil{t.Fatal(err)}
	adminRole:=&model.AdminRole{Name:"One account",Permissions:"users.view",ScopeType:AdminScopeIndividual,ScopeId:owner.Id,Enabled:true}
	if err:=DB.Create(adminRole).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.AdminRoleAssignment{UserId:owner.Id,RoleId:adminRole.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.AdminRoleAssignment{UserId:other.Id,RoleId:adminRole.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.UserToken{UserId:owner.Id,Token:"old-session",ExpiredAt:4102444800}).Error;err!=nil{
		t.Fatal(err)
	}
	for _,row:=range []interface{}{
		&model.UserTOTP{UserId:owner.Id,Enabled:true,SecretCiphertext:"sample-encrypted-secret"},
		&model.TOTPRecoveryCode{UserId:owner.Id,CodeHash:"sample-recovery-hash"},
		&model.TOTPLoginChallenge{UserId:owner.Id,TokenHash:"sample-challenge-hash",ExpiresAt:4102444800},
		&model.PasswordResetToken{UserId:owner.Id,TokenHash:"sample-password-reset",ExpiresAt:4102444800},
		&model.EmailLoginSetting{UserId:owner.Id,Enabled:true,VerifiedEmail:"owner@example.test"},
		&model.EmailCodeChallenge{UserId:owner.Id,Email:"owner@example.test",Purpose:"login",
			TokenHash:"sample-email-token-hash",CodeHash:"sample-email-code-hash",ExpiresAt:4102444800},
		&model.UserInvitation{Email:"invited@example.test",CreatedBy:owner.Id,GroupId:2,
			TokenHash:"sample-invitation",ExpiresAt:4102444800},
	}{
		if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
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
	for _,table:=range []interface{}{
		&model.UserTOTP{},&model.TOTPRecoveryCode{},&model.TOTPLoginChallenge{},
		&model.PasswordResetToken{},&model.EmailLoginSetting{},&model.EmailCodeChallenge{},
	}{
		if err:=DB.Model(table).Where("user_id = ?",owner.Id).Count(&count).Error;err!=nil||count!=0{
			t.Fatalf("user authentication data remains in %T: %d %v",table,count,err)
		}
	}
	var revoked model.UserInvitation
	if err:=DB.Where("created_by = ?",owner.Id).First(&revoked).Error;err!=nil||revoked.RevokedAt==0{
		t.Fatalf("invitation issued by a deleted user remains valid: %+v %v",revoked,err)
	}
	if err:=DB.Model(&model.AddressBookCollectionRule{}).Where(
		"(type = ? AND to_id = ?) OR (type = ? AND to_id = ?)",
		model.ShareAddressBookRuleTypeGroup,group.Id,
		model.ShareAddressBookRuleTypePersonal,owner.Id,
	).Count(&count).Error;err!=nil||count!=0{t.Fatalf("stale shared address book grant remains: %d %v",count,err)}

	if err:=DB.Model(&model.ControlRoleAssignment{}).
		Where("target_type = ? AND target_id = ?","user",strconv.FormatUint(uint64(owner.Id),10)).
		Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted account still has assigned control role: %d %v",count,err)
	}
	if err:=DB.Model(&model.AdminRoleAssignment{}).Where("user_id = ?",owner.Id).
		Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted account still has management role assignment: %d %v",count,err)
	}
	if err:=DB.Model(&model.AdminRole{}).Where("id = ?",adminRole.Id).
		Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted account retained individual management role: %d %v",count,err)
	}
	if err:=DB.Model(&model.AdminRoleAssignment{}).Where("role_id = ?",adminRole.Id).
		Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("deleted role retained assignments for other accounts: %d %v",count,err)
	}
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
