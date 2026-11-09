package service

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
)

func TestPasswordRecoveryByEmail(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.SMTPSetting{},&model.PasswordResetToken{},&model.UserToken{});err!=nil{t.Fatal(err)}
	hash,err:=utils.EncryptPassword("InitialPassword123")
	if err!=nil{t.Fatal(err)}
	user:=&model.User{Username:"recover-account",Email:"person@example.com",Status:1,Password:hash}
	if err:=DB.Create(user).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.UserToken{UserId:user.Id,Token:"existing-session",ExpiredAt:time.Now().Add(time.Hour).Unix()}).Error;err!=nil{t.Fatal(err)}
	port,received:=startMockSMTP(t)
	if _,err:=SaveMailSettings(MailSettingsInput{
		Enabled:true,Host:"127.0.0.1",Port:port,Security:"none",
		SenderEmail:"support@example.com",BaseURL:"https://remote.example.com",
	});err!=nil{t.Fatal(err)}
	if err:=RequestPasswordReset("unknown@example.com");err!=nil{t.Fatal(err)}
	if err:=RequestPasswordReset(user.Email);err!=nil{t.Fatal(err)}
	var email string
	select{
	case email=<-received:
	case <-time.After(3*time.Second):t.Fatal("password recovery email was not delivered")
	}
	link:=regexp.MustCompile(`https://[^\s]+`).FindString(email)
	if !strings.Contains(link,"/_admin/#/password-reset?token="){
		t.Fatalf("password recovery email contains no reset link: %q",email)
	}
	token:=strings.SplitN(link,"token=",2)[1]
	var row model.PasswordResetToken
	if err:=DB.Where("user_id = ?",user.Id).First(&row).Error;err!=nil{t.Fatal(err)}
	if row.TokenHash==token||row.TokenHash==""{t.Fatal("plaintext reset token was stored")}
	if err:=CompletePasswordReset(token,"NewPassword456");err!=nil{t.Fatal(err)}
	if err:=CompletePasswordReset(token,"AnotherPassword789");err==nil{t.Fatal("used reset link accepted")}
	var updated model.User
	if err:=DB.First(&updated,user.Id).Error;err!=nil{t.Fatal(err)}
	ok,_,err:=utils.VerifyPassword(updated.Password,"NewPassword456")
	if err!=nil||!ok{t.Fatal("recovery did not update account password")}
	var count int64
	if err:=DB.Model(&model.UserToken{}).Where("user_id = ?",user.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("old login session remains: %d %v",count,err)
	}
	if err:=CompletePasswordReset("bad-token","ValidPassword456");err==nil{t.Fatal("invalid token accepted")}
	if err:=CompletePasswordReset(token,"short");err==nil{t.Fatal("short password accepted")}
}

func TestPasswordRecoveryRequiresConfiguredMail(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.SMTPSetting{},&model.PasswordResetToken{});err!=nil{t.Fatal(err)}
	if err:=RequestPasswordReset("someone@example.com");err==nil{
		t.Fatal("email recovery succeeded without configured SMTP")
	}
}
