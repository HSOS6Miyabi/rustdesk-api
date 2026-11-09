package service

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func prepareEmailVerification(t *testing.T)(*model.User,func()) {
	t.Helper()
	cleanup:=setupGroupDB(t)
	if err:=DB.AutoMigrate(&model.SMTPSetting{},&model.EmailLoginSetting{},
		&model.EmailCodeChallenge{},&model.TOTPLoginChallenge{});err!=nil{
		cleanup()
		t.Fatal(err)
	}
	user:=seedGroups(t)["member"]
	user.Email="member@example.test"
	if err:=DB.Model(user).Update("email",user.Email).Error;err!=nil{
		cleanup()
		t.Fatal(err)
	}
	return user,cleanup
}

func oneTimeEmailCode(t *testing.T,received <-chan string)string{
	t.Helper()
	var body string
	select{
	case body=<-received:
	case <-time.After(5*time.Second):t.Fatal("email code was not delivered")
	}
	match:=regexp.MustCompile(`验证码：([0-9]{6})`).FindStringSubmatch(body)
	if len(match)!=2{t.Fatalf("SMTP email does not contain the code: %q",body)}
	return match[1]
}

func setUpEmailSMTP(t *testing.T)(<-chan string){
	t.Helper()
	port,received:=startMockSMTP(t)
	if _,err:=SaveMailSettings(MailSettingsInput{
		Enabled:true,Host:"127.0.0.1",Port:port,Security:"none",
		SenderEmail:"team@example.test",
	});err!=nil{t.Fatal(err)}
	return received
}

func TestEmailLoginEnrollmentAndVerification(t *testing.T){
	user,cleanup:=prepareEmailVerification(t)
	defer cleanup()
	initial,err:=GetEmailVerificationStatus(user.Id)
	if err!=nil||initial.Enabled{t.Fatalf("email login must be opt-in: %+v %v",initial,err)}
	if enabled,err:=EmailLoginEnabled(user.Id,user.Email);err!=nil||enabled{t.Fatal("email verification unexpectedly enabled")}
	received:=setUpEmailSMTP(t)
	enrollment,err:=BeginEmailVerification(user.Id)
	if err!=nil{t.Fatal(err)}
	code:=oneTimeEmailCode(t,received)
	if err:=ConfirmEmailVerification(user.Id,enrollment,"wrong!");err==nil{
		t.Fatal("invalid email code accepted")
	}
	if err:=ConfirmEmailVerification(user.Id,enrollment,code);err!=nil{t.Fatal(err)}
	if err:=ConfirmEmailVerification(user.Id,enrollment,code);err==nil{
		t.Fatal("enrollment code was reused")
	}
	status,err:=GetEmailVerificationStatus(user.Id)
	if err!=nil||!status.Enabled||status.Email!=user.Email||!status.MailConfigured{
		t.Fatalf("confirmed email verification status incorrect: %+v %v",status,err)
	}
	received=setUpEmailSMTP(t)
	loginSecret,err:=BeginEmailLogin(user.Id)
	if err!=nil{t.Fatal(err)}
	loginCode:=oneTimeEmailCode(t,received)
	if _,err:=CompleteSecondFactorLogin(loginSecret,user.Username,"abcdef");err==nil{
		t.Fatal("malformed email code accepted")
	}
	verified,err:=CompleteSecondFactorLogin(loginSecret,user.Username,loginCode)
	if err!=nil||verified.Id!=user.Id{t.Fatalf("email challenge rejected: %+v %v",verified,err)}
	if _,err:=CompleteSecondFactorLogin(loginSecret,user.Username,loginCode);err==nil{
		t.Fatal("email verification was reused")
	}
	if err:=DisableEmailVerification(user.Id);err!=nil{t.Fatal(err)}
	if enabled,err:=EmailLoginEnabled(user.Id,user.Email);err!=nil||enabled{
		t.Fatal("email verification could not be disabled")
	}
	if _,err:=BeginEmailLogin(user.Id);err==nil{
		t.Fatal("email login challenge was issued after disabling")
	}
}

func TestEmailLoginRespectsAccountEmailChangesAndRateLimits(t *testing.T) {
	user,cleanup:=prepareEmailVerification(t)
	defer cleanup()
	received:=setUpEmailSMTP(t)
	secret,err:=BeginEmailVerification(user.Id)
	if err!=nil{t.Fatal(err)}
	code:=oneTimeEmailCode(t,received)
	if err:=DB.Model(user).Update("email","changed@example.test").Error;err!=nil{t.Fatal(err)}
	if err:=ConfirmEmailVerification(user.Id,secret,code);err==nil{
		t.Fatal("previous email address could enable current account")
	}
	if err:=DB.Model(user).Update("email","member@example.test").Error;err!=nil{t.Fatal(err)}
	if err:=ConfirmEmailVerification(user.Id,secret,code);err!=nil{t.Fatal(err)}
	user.Email="member@example.test"
	if enabled,err:=EmailLoginEnabled(user.Id,"changed@example.test");err!=nil||enabled{
		t.Fatal("old verification still applies to changed email")
	}
	now:=time.Now().Unix()
	for i:=0;i<int(emailCodeHourlyLimit);i++ {
		row:=&model.EmailCodeChallenge{
			UserId:user.Id,Email:user.Email,Purpose:"login",
			TokenHash:strings.Repeat(string(rune('a'+i)),64),
			CodeHash:strings.Repeat(string(rune('1'+i)),64),
			IssuedAt:now,ExpiresAt:now+60,
		}
		if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
	}
	if _,err:=BeginEmailLogin(user.Id);err==nil{
		t.Fatal("email verification issue limit ignored")
	}
}

func TestEmailCodeChallengeAttemptLimit(t *testing.T) {
	user,cleanup:=prepareEmailVerification(t)
	defer cleanup()
	now:=time.Now().Unix()
	secret:=strings.Repeat("a",43)
	row:=&model.EmailCodeChallenge{
		UserId:user.Id,Email:user.Email,Purpose:"login",
		TokenHash:passwordResetHash(secret),CodeHash:emailCodeHash(secret,"123456"),
		IssuedAt:now,ExpiresAt:now+600,
	}
	if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.EmailLoginSetting{
		UserId:user.Id,VerifiedEmail:user.Email,Enabled:true,
	}).Error;err!=nil{t.Fatal(err)}
	for i:=0;i<emailCodeMaxAttempts;i++{
		if _,err:=CompleteEmailLogin(secret,user.Username,"000000x");err==nil{
			t.Fatal("malformed code was accepted")
		}
	}
	if _,err:=CompleteEmailLogin(secret,user.Username,"123456");err==nil{
		t.Fatal("verification request allowed more than five attempts")
	}
}
