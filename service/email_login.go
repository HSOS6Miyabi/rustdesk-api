package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	emailCodeLifetimeSeconds int64 = 600
	emailCodeMaxAttempts = 5
	emailCodeHourlyLimit int64 = 6
)

type EmailVerificationStatus struct {
	Enabled bool `json:"enabled"`
	Email string `json:"email"`
	MailConfigured bool `json:"mail_configured"`
}

func currentEmailSetting(userID uint) (*model.EmailLoginSetting,error) {
	var setting model.EmailLoginSetting
	err:=DB.Where("user_id = ?",userID).First(&setting).Error
	if errors.Is(err,gorm.ErrRecordNotFound){return nil,nil}
	if err!=nil{return nil,err}
	return &setting,nil
}

func GetEmailVerificationStatus(userID uint) (*EmailVerificationStatus,error) {
	if userID==0{return nil,errors.New("account ID is required")}
	var user model.User
	if err:=DB.Where("id = ?",userID).First(&user).Error;err!=nil{return nil,err}
	setting,err:=currentEmailSetting(userID)
	if err!=nil{return nil,err}
	settings,err:=currentMailSettings()
	if err!=nil{return nil,err}
	return &EmailVerificationStatus{
		Enabled:setting!=nil&&setting.Enabled&&setting.VerifiedEmail==user.Email&&user.Email!="",
		Email:user.Email,MailConfigured:settings.Enabled&&settings.Host!=""&&settings.SenderEmail!="",
	},nil
}

func EmailLoginEnabled(userID uint,email string) (bool,error) {
	setting,err:=currentEmailSetting(userID)
	if err!=nil{return false,err}
	return setting!=nil&&setting.Enabled&&email!=""&&email==setting.VerifiedEmail,nil
}

func emailCodeHash(secret,code string) string {
	digest:=sha256.Sum256([]byte(secret+":"+code))
	return hex.EncodeToString(digest[:])
}

// createEmailChallenge issues a random login or enrollment code by SMTP.
func createEmailChallenge(userID uint,purpose string) (string,error) {
	if userID==0||(purpose!="enroll"&&purpose!="login"){
		return "",errors.New("invalid email verification request")
	}
	var user model.User
	if err:=DB.Where("id = ? AND status = ?",userID,model.COMMON_STATUS_ENABLE).
		First(&user).Error;err!=nil{return "",errors.New("active account was not found")}
	email:=strings.TrimSpace(user.Email)
	address,err:=mail.ParseAddress(email)
	if err!=nil||address.Address!=email||email==""{
		return "",errors.New("account email is missing or invalid")
	}
	settings,err:=currentMailSettings()
	if err!=nil{return "",err}
	if !settings.Enabled{return "",errors.New("SMTP email service is disabled")}
	if purpose=="login" {
		enabled,err:=EmailLoginEnabled(userID,email)
		if err!=nil{return "",err}
		if !enabled{return "",errors.New("email verification is not enabled")}
	}
	now:=time.Now().Unix()
	var recent int64
	if err:=DB.Model(&model.EmailCodeChallenge{}).
		Where("user_id = ? AND purpose = ? AND issued_at > ?",userID,purpose,now-3600).
		Count(&recent).Error;err!=nil{return "",err}
	if recent>=emailCodeHourlyLimit {
		return "",errors.New("too many email verification requests; try again later")
	}
	secretBytes:=make([]byte,32)
	if _,err:=rand.Read(secretBytes);err!=nil{return "",err}
	secret:=base64.RawURLEncoding.EncodeToString(secretBytes)
	number,err:=rand.Int(rand.Reader,big.NewInt(1000000))
	if err!=nil{return "",err}
	code:=fmt.Sprintf("%06d",number.Int64())
	record:=&model.EmailCodeChallenge{
		UserId:userID,Email:email,Purpose:purpose,
		TokenHash:passwordResetHash(secret),CodeHash:emailCodeHash(secret,code),
		IssuedAt:now,ExpiresAt:now+emailCodeLifetimeSeconds,
	}
	if err:=DB.Create(record).Error;err!=nil{return "",err}
	activity:="启用邮件登录验证"
	if purpose=="login"{activity="登录 RustDesk"}
	body:=fmt.Sprintf("你正在%s。\n\n本次邮箱验证码：%s\n\n验证码有效期为 10 分钟，请在当前验证页面输入。如果没有发起该操作，请忽略这封邮件。",activity,code)
	if err:=sendSMTP(settings,email,"RustDesk 邮箱验证码",body);err!=nil{
		_ = DB.Where("id = ?",record.Id).Delete(&model.EmailCodeChallenge{}).Error
		return "",err
	}
	return secret,nil
}

func BeginEmailVerification(userID uint)(string,error) {
	return createEmailChallenge(userID,"enroll")
}

func BeginEmailLogin(userID uint)(string,error) {
	return createEmailChallenge(userID,"login")
}

// consumeEmailChallenge enforces a single-use code with an attempt limit.
func consumeEmailChallenge(tx *gorm.DB,secret,code,purpose,username string) (*model.User,error) {
	if len(secret)!=43||len(code)!=6{return nil,errors.New("invalid email verification request")}
	for _,digit:=range code{if digit<'0'||digit>'9'{return nil,errors.New("invalid email verification code")}}
	now:=time.Now().Unix()
	var challenge model.EmailCodeChallenge
	err:=tx.Where("token_hash = ? AND purpose = ? AND consumed_at = 0 AND expires_at > ? AND attempts < ?",
		passwordResetHash(secret),purpose,now,emailCodeMaxAttempts).
		First(&challenge).Error
	if err!=nil{return nil,errors.New("verification request has expired")}
	var user model.User
	if err:=tx.Where("id = ? AND status = ?",challenge.UserId,model.COMMON_STATUS_ENABLE).
		First(&user).Error;err!=nil{return nil,errors.New("account is unavailable")}
	if user.Email!=challenge.Email||username!=""&&username!=user.Username {
		return nil,errors.New("account email or username has changed")
	}
	if emailCodeHash(secret,code)!=challenge.CodeHash {
		return nil,errors.New("incorrect email verification code")
	}
	result:=tx.Model(&model.EmailCodeChallenge{}).
		Where("id = ? AND consumed_at = 0",challenge.Id).
		Update("consumed_at",now)
	if result.Error!=nil{return nil,result.Error}
	if result.RowsAffected!=1{return nil,errors.New("verification request already used")}
	return &user,nil
}

func incrementEmailCodeAttempts(secret string) {
	if len(secret)!=43{return}
	_ = DB.Model(&model.EmailCodeChallenge{}).
		Where("token_hash = ? AND consumed_at = 0 AND expires_at > ?",passwordResetHash(secret),time.Now().Unix()).
		Update("attempts",gorm.Expr("attempts + 1")).Error
}

func ConfirmEmailVerification(userID uint,secret,code string)error {
	if userID==0{return errors.New("account ID is required")}
	err:=DB.Transaction(func(tx *gorm.DB)error {
		user,err:=consumeEmailChallenge(tx,secret,code,"enroll","")
		if err!=nil{return err}
		if user.Id!=userID{return errors.New("verification request belongs to a different account")}
		return tx.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"user_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"verified_email","enabled","updated_at"}),
		}).Create(&model.EmailLoginSetting{
			UserId:userID,VerifiedEmail:user.Email,Enabled:true,
		}).Error
	})
	if err!=nil{incrementEmailCodeAttempts(secret)}
	return err
}

func CompleteEmailLogin(secret,username,code string) (*model.User,error) {
	var user *model.User
	err:=DB.Transaction(func(tx *gorm.DB)error {
		verified,err:=consumeEmailChallenge(tx,secret,code,"login",username)
		if err!=nil{return err}
		var setting model.EmailLoginSetting
		if err:=tx.Where("user_id = ? AND verified_email = ? AND enabled = ?",
			verified.Id,verified.Email,true).First(&setting).Error;err!=nil{
			return errors.New("email login verification is no longer enabled")
		}
		user=verified
		return nil
	})
	if err!=nil{
		incrementEmailCodeAttempts(secret)
		return nil,err
	}
	return user,nil
}

func DisableEmailVerification(userID uint) error {
	if userID==0{return errors.New("account ID is required")}
	result:=DB.Model(&model.EmailLoginSetting{}).
		Where("user_id = ?",userID).Update("enabled",false)
	return result.Error
}
