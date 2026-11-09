package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func setupTOTPTest(t *testing.T)(*model.User,func()){
	t.Helper()
	cleanup:=setupGroupDB(t)
	if err:=DB.AutoMigrate(&model.UserTOTP{},&model.TOTPRecoveryCode{},
		&model.TOTPLoginChallenge{});err!=nil{cleanup();t.Fatal(err)}
	t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","local-test-only-encryption-key-please-change")
	user:=seedGroups(t)["member"]
	return user,cleanup
}

func TestTOTPStandardTimeStep(t *testing.T){
	secret:="GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	got,err:=totpCode(secret,59/30)
	if err!=nil||got!="287082"{t.Fatalf("RFC 6238 sample failed: %s %v",got,err)}
	if step:=matchingTOTPStep(secret,"287082",59);step!=1{
		t.Fatalf("expected time step 1, got %d",step)
	}
	if matchingTOTPStep(secret,"287083",59)>=0{
		t.Fatal("wrong TOTP accepted")
	}
	if matchingTOTPStep(secret,"123456a",59)>=0{
		t.Fatal("malformed TOTP accepted")
	}
}

func TestTOTPEnrollmentAndRecovery(t *testing.T){
	user,cleanup:=setupTOTPTest(t)
	defer cleanup()
	status,err:=GetTOTPStatus(user.Id)
	if err!=nil||status.Enabled{t.Fatalf("TOTP should be disabled: %+v %v",status,err)}
	setup,err:=BeginTOTPSetup(user.Id)
	if err!=nil{t.Fatal(err)}
	if !strings.Contains(setup.URI,"otpauth://totp/")||!strings.Contains(setup.URI,"secret="){
		t.Fatalf("authenticator URI is invalid: %s",setup.URI)
	}
	var private model.UserTOTP
	if err:=DB.Where("user_id = ?",user.Id).First(&private).Error;err!=nil{t.Fatal(err)}
	if strings.Contains(private.PendingCiphertext,setup.Secret){
		t.Fatal("unencrypted authenticator secret was stored")
	}
	code,err:=totpCode(setup.Secret,time.Now().Unix()/30)
	if err!=nil{t.Fatal(err)}
	codes,err:=ActivateTOTP(user.Id,code)
	if err!=nil||len(codes)!=6{t.Fatalf("activation failed: %v %v",codes,err)}
	if _,err:=ActivateTOTP(user.Id,code);err==nil{t.Fatal("TOTP activated twice")}
	status,err=GetTOTPStatus(user.Id)
	if err!=nil||!status.Enabled||status.RemainingRecoveryCodes!=6{
		t.Fatalf("TOTP status incorrect: %+v %v",status,err)
	}
	challenge,err:=StartTOTPLoginChallenge(user.Id)
	if err!=nil{t.Fatal(err)}
	nextCode,err:=totpCode(setup.Secret,time.Now().Unix()/30+1)
	if err!=nil{t.Fatal(err)}
	authenticated,err:=CompleteTOTPLoginChallenge(challenge,user.Username,nextCode)
	if err!=nil||authenticated.Id!=user.Id{t.Fatalf("valid code rejected: %v %v",authenticated,err)}
	if _,err:=CompleteTOTPLoginChallenge(challenge,user.Username,nextCode);err==nil{
		t.Fatal("login challenge reused after verification")
	}
	recoveryLogin,err:=StartTOTPLoginChallenge(user.Id)
	if err!=nil{t.Fatal(err)}
	if _,err:=CompleteTOTPLoginChallenge(recoveryLogin,user.Username,codes[0]);err!=nil{t.Fatal(err)}
	status,err=GetTOTPStatus(user.Id)
	if err!=nil||status.RemainingRecoveryCodes!=5{t.Fatal("consumed recovery code was not marked used")}
	replay,err:=StartTOTPLoginChallenge(user.Id)
	if err!=nil{t.Fatal(err)}
	if _,err:=CompleteTOTPLoginChallenge(replay,user.Username,codes[0]);err==nil{
		t.Fatal("used recovery code accepted again")
	}
	if _,err:=CompleteTOTPLoginChallenge(replay,"different-user",codes[1]);err==nil{
		t.Fatal("challenge accepted a different username")
	}
	newCodes,err:=RotateTOTPRecoveryCodes(user.Id,codes[1])
	if err!=nil||len(newCodes)!=6{t.Fatalf("recovery code regeneration failed: %v %v",newCodes,err)}
	for _,item:=range newCodes{
		if item==codes[0]||item==codes[1]{t.Fatal("old recovery codes were retained")}
	}
	status,err=GetTOTPStatus(user.Id)
	if err!=nil||status.RemainingRecoveryCodes!=6{t.Fatalf("regenerated recovery codes: %+v %v",status,err)}
	if err:=DisableTOTP(user.Id,newCodes[0]);err!=nil{t.Fatal(err)}
	status,err=GetTOTPStatus(user.Id)
	if err!=nil||status.Enabled||status.RemainingRecoveryCodes!=0{
		t.Fatalf("TOTP remains enabled: %+v %v",status,err)
	}
	var persisted model.UserTOTP
	if err:=DB.Where("user_id = ?",user.Id).First(&persisted).Error;err==nil{
		t.Fatal("disabled authenticator secret is still stored")
	}
}

func TestTOTPChallengeAttemptLimitAndNoSecretLeak(t *testing.T){
	user,cleanup:=setupTOTPTest(t)
	defer cleanup()
	setup,err:=BeginTOTPSetup(user.Id)
	if err!=nil{t.Fatal(err)}
	code,err:=totpCode(setup.Secret,time.Now().Unix()/30)
	if err!=nil{t.Fatal(err)}
	recovery,err:=ActivateTOTP(user.Id,code)
	if err!=nil{t.Fatal(err)}
	challenge,err:=StartTOTPLoginChallenge(user.Id)
	if err!=nil{t.Fatal(err)}
	for i:=0;i<totpMaxAttempts;i++{
		if _,err:=CompleteTOTPLoginChallenge(challenge,user.Username,"000000x");err==nil{
			t.Fatal("invalid login code accepted")
		}
	}
	if _,err:=CompleteTOTPLoginChallenge(challenge,user.Username,recovery[0]);err==nil{
		t.Fatal("challenge allowed more than the maximum number of attempts")
	}
	var saved model.UserTOTP
	if err:=DB.Where("user_id = ?",user.Id).First(&saved).Error;err!=nil{t.Fatal(err)}
	bytes,err:=json.Marshal(saved)
	if err!=nil{t.Fatal(err)}
	if strings.Contains(string(bytes),setup.Secret)||strings.Contains(string(bytes),"ciphertext"){
		t.Fatal("authenticator secret leaked in API response")
	}
}
