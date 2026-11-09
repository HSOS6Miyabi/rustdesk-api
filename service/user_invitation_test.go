package service

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
)

func TestTeamInvitationRegistersOnlySelectedAccount(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.SMTPSetting{},&model.UserInvitation{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	port,email:=startMockSMTP(t)
	if _,err:=SaveMailSettings(MailSettingsInput{
		Enabled:true,Host:"127.0.0.1",Port:port,Security:"none",
		SenderEmail:"team@example.com",BaseURL:"https://remote.example.com",
	});err!=nil{t.Fatal(err)}
	if err:=InviteUser("new.member@example.com",2,users["admin"].Id,2);err!=nil{t.Fatal(err)}
	if err:=InviteUser("new.member@example.com",2,users["admin"].Id,2);err==nil{
		t.Fatal("duplicate active invitation was accepted")
	}
	var message string
	select{case message=<-email:case <-time.After(3*time.Second):t.Fatal("invitation email was not delivered")}
	link:=regexp.MustCompile(`https://[^\s]+`).FindString(message)
	if !strings.Contains(link,"/_admin/#/invite?token="){
		t.Fatalf("invitation link was missing: %s",message)
	}
	raw:=strings.SplitN(link,"token=",2)[1]
	var record model.UserInvitation
	if err:=DB.Where("email = ?","new.member@example.com").First(&record).Error;err!=nil{t.Fatal(err)}
	if record.TokenHash==raw||record.TokenHash==""{t.Fatal("plaintext invitation token was stored")}
	if err:=AcceptUserInvitation(raw,"NEW.Member","New colleague","NewPassword456");err!=nil{t.Fatal(err)}
	if err:=AcceptUserInvitation(raw,"another-member","","NewPassword789");err==nil{
		t.Fatal("invitation was accepted more than once")
	}
	var account model.User
	if err:=DB.Where("username = ?","new.member").First(&account).Error;err!=nil{t.Fatal(err)}
	if account.Email!="new.member@example.com"||account.GroupId!=2||account.Nickname!="New colleague"{
		t.Fatalf("new member was created incorrectly: %+v",account)
	}
	valid,_,err:=utils.VerifyPassword(account.Password,"NewPassword456")
	if err!=nil||!valid{t.Fatal("new member cannot log in with invited password")}
	if err:=AcceptUserInvitation("invalid","new-member","","NewPassword456");err==nil{
		t.Fatal("invalid invitation token was accepted")
	}
	if err:=DB.Where("id = ?",record.Id).First(&record).Error;err!=nil{t.Fatal(err)}
	if record.AcceptedAt<=0{t.Fatal("invitation completion was not recorded")}
}

func TestTeamInvitationRevocationAndInputValidation(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.UserInvitation{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	now:=time.Now().Unix()
	raw:=strings.Repeat("a",43)
	row:=&model.UserInvitation{
		Email:"another@example.com",GroupId:2,CreatedBy:users["admin"].Id,
		TokenHash:passwordResetHash(raw),ExpiresAt:now+3600,
	}
	if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
	if err:=RevokeUserInvitation(row.Id);err!=nil{t.Fatal(err)}
	if err:=RevokeUserInvitation(row.Id);err==nil{t.Fatal("revoked invitation was revoked twice")}
	if err:=AcceptUserInvitation(raw,"new.member","","NewPassword456");err==nil{
		t.Fatal("revoked invitation was accepted")
	}
	if err:=AcceptUserInvitation(raw,"wrong username","","NewPassword456");err==nil{
		t.Fatal("invalid invitation username was accepted")
	}
	expired:=&model.UserInvitation{
		Email:"expired@example.com",GroupId:2,CreatedBy:users["admin"].Id,
		TokenHash:passwordResetHash(strings.Repeat("b",43)),ExpiresAt:now-1,
	}
	if err:=DB.Create(expired).Error;err!=nil{t.Fatal(err)}
	if err:=AcceptUserInvitation(strings.Repeat("b",43),"valid-name","","NewPassword456");err==nil{
		t.Fatal("expired invitation was accepted")
	}
}
