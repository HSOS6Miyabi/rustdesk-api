package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAdminResetSecondFactorsRevokesSessions(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.UserTOTP{},&model.TOTPRecoveryCode{},
		&model.TOTPLoginChallenge{},&model.EmailLoginSetting{},
		&model.EmailCodeChallenge{},&model.UserToken{});err!=nil{t.Fatal(err)}
	user:=seedGroups(t)["member"]
	for _,row:=range []interface{}{
		&model.UserTOTP{UserId:user.Id,Enabled:true,SecretCiphertext:"test-encrypted"},
		&model.TOTPRecoveryCode{UserId:user.Id,CodeHash:"sample-code"},
		&model.TOTPLoginChallenge{UserId:user.Id,TokenHash:"sample-challenge",ExpiresAt:time.Now().Add(time.Hour).Unix()},
		&model.EmailLoginSetting{UserId:user.Id,Enabled:true,VerifiedEmail:"person@example.test"},
		&model.EmailCodeChallenge{UserId:user.Id,Email:"person@example.test",Purpose:"login",
			TokenHash:"sample-email",CodeHash:"sample-digest",ExpiresAt:time.Now().Add(time.Hour).Unix()},
		&model.UserToken{UserId:user.Id,Token:"already-logged-in",ExpiredAt:time.Now().Add(time.Hour).Unix()},
	}{
		if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
	}
	if err:=ResetUserSecondFactors(user.Id);err!=nil{t.Fatal(err)}
	for _,row:=range []interface{}{
		&model.UserTOTP{},&model.TOTPRecoveryCode{},&model.TOTPLoginChallenge{},
		&model.EmailLoginSetting{},&model.EmailCodeChallenge{},&model.UserToken{},
	}{
		var count int64
		if err:=DB.Model(row).Where("user_id = ?",user.Id).Count(&count).Error;err!=nil||count!=0{
			t.Fatalf("second-factor reset retained data in %T: %d %v",row,count,err)
		}
	}
	if err:=ResetUserSecondFactors(999999);err==nil{t.Fatal("reset an unknown account")}
	if err:=ResetUserSecondFactors(0);err==nil{t.Fatal("reset with empty user ID")}
}
