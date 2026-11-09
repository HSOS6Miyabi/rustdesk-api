package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const (
	totpPeriodSeconds int64=30
	totpPendingSeconds int64=600
	totpChallengeSeconds int64=300
	totpRecoveryCount=6
	totpMaxAttempts=5
)

type TOTPStatus struct {
	Enabled bool `json:"enabled"`
	Pending bool `json:"pending"`
	RemainingRecoveryCodes int64 `json:"remaining_recovery_codes"`
}

type TOTPSetup struct {
	Secret string `json:"secret"`
	URI string `json:"uri"`
	ExpiresAt int64 `json:"expires_at"`
}

func totpAccount(userID uint) (*model.UserTOTP,error) {
	var row model.UserTOTP
	err:=DB.Where("user_id = ?",userID).First(&row).Error
	if errors.Is(err,gorm.ErrRecordNotFound){return nil,nil}
	if err!=nil{return nil,err}
	return &row,nil
}

func GetTOTPStatus(userID uint) (*TOTPStatus,error){
	if userID==0{return nil,errors.New("account ID is required")}
	account,err:=totpAccount(userID)
	if err!=nil{return nil,err}
	status:=&TOTPStatus{}
	if account==nil{return status,nil}
	status.Enabled=account.Enabled
	status.Pending=account.PendingCiphertext!=""&&account.PendingExpiresAt>time.Now().Unix()
	if account.Enabled{
		if err:=DB.Model(&model.TOTPRecoveryCode{}).
			Where("user_id = ? AND used_at = 0",userID).
			Count(&status.RemainingRecoveryCodes).Error;err!=nil{return nil,err}
	}
	return status,nil
}

func newTOTPSecret() (string,error){
	raw:=make([]byte,20)
	if _,err:=rand.Read(raw);err!=nil{return "",err}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw),nil
}

func totpCode(secret string,step int64) (string,error){
	decoded,err:=base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err!=nil||len(decoded)==0{return "",errors.New("invalid TOTP secret")}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:],uint64(step))
	mac:=hmac.New(sha1.New,decoded)
	mac.Write(counter[:])
	h:=mac.Sum(nil)
	offset:=int(h[len(h)-1]&15)
	n:=binary.BigEndian.Uint32(h[offset:offset+4])&0x7fffffff
	return fmt.Sprintf("%06d",n%1000000),nil
}

func matchingTOTPStep(secret,code string,now int64) int64 {
	if len(code)!=6{return -1}
	for _,char:=range code{if char<'0'||char>'9'{return -1}}
	current:=now/totpPeriodSeconds
	for _,step:=range []int64{current,current-1,current+1} {
		if step<0{continue}
		expected,err:=totpCode(secret,step)
		if err==nil&&hmac.Equal([]byte(expected),[]byte(code)){return step}
	}
	return -1
}

func BeginTOTPSetup(userID uint) (*TOTPSetup,error){
	if userID==0{return nil,errors.New("account ID is required")}
	var user model.User
	if err:=DB.First(&user,userID).Error;err!=nil{return nil,err}
	previous,err:=totpAccount(userID)
	if err!=nil{return nil,err}
	if previous!=nil&&previous.Enabled{return nil,errors.New("TOTP is already enabled")}
	secret,err:=newTOTPSecret()
	if err!=nil{return nil,err}
	ciphertext,err:=encryptMailPassword(secret)
	if err!=nil{return nil,err}
	expires:=time.Now().Unix()+totpPendingSeconds
	err=DB.Transaction(func(tx *gorm.DB)error{
		if previous==nil {
			return tx.Create(&model.UserTOTP{
				UserId:userID,PendingCiphertext:ciphertext,PendingExpiresAt:expires,
			}).Error
		}
		return tx.Model(&model.UserTOTP{}).Where("user_id = ? AND enabled = ?",userID,false).
			Updates(map[string]interface{}{
				"pending_ciphertext":ciphertext,"pending_expires_at":expires,
			}).Error
	})
	if err!=nil{return nil,err}
	issuer:="RustDesk OSS API"
	identifier:=issuer+":"+user.Username
	uri:="otpauth://totp/"+url.PathEscape(identifier)+
		"?secret="+url.QueryEscape(secret)+"&issuer="+url.QueryEscape(issuer)+
		"&algorithm=SHA1&digits=6&period=30"
	return &TOTPSetup{Secret:secret,URI:uri,ExpiresAt:expires},nil
}

func createTOTPRecoveryCodes(tx *gorm.DB,userID uint) ([]string,error) {
	if err:=tx.Where("user_id = ?",userID).
		Delete(&model.TOTPRecoveryCode{}).Error;err!=nil{return nil,err}
	codes:=make([]string,0,totpRecoveryCount)
	for i:=0;i<totpRecoveryCount;i++{
		bytes:=make([]byte,10)
		if _,err:=rand.Read(bytes);err!=nil{return nil,err}
		code:=strings.ToUpper(hex.EncodeToString(bytes))
		key:=totpRecoveryHash(userID,code)
		if err:=tx.Create(&model.TOTPRecoveryCode{
			UserId:userID,CodeHash:key,
		}).Error;err!=nil{return nil,err}
		codes=append(codes,code)
	}
	return codes,nil
}

func totpRecoveryHash(userID uint,code string) string {
	hash:=sha256.Sum256([]byte(fmt.Sprintf("%d:%s",userID,strings.ToUpper(strings.TrimSpace(code)))))
	return hex.EncodeToString(hash[:])
}

func ActivateTOTP(userID uint,code string) ([]string,error) {
	if userID==0{return nil,errors.New("account ID is required")}
	codes:=[]string{}
	err:=DB.Transaction(func(tx *gorm.DB)error{
		var account model.UserTOTP
		if err:=tx.Where("user_id = ? AND enabled = ?",userID,false).First(&account).Error;err!=nil{return err}
		now:=time.Now().Unix()
		if account.PendingExpiresAt<=now||account.PendingCiphertext==""{
			return errors.New("TOTP setup has expired")
		}
		secret,err:=decryptMailPassword(account.PendingCiphertext)
		if err!=nil{return err}
		step:=matchingTOTPStep(secret,code,now)
		if step<0{return errors.New("invalid authenticator code")}
		newCodes,err:=createTOTPRecoveryCodes(tx,userID)
		if err!=nil{return err}
		res:=tx.Model(&model.UserTOTP{}).
			Where("id = ? AND enabled = ?",account.Id,false).
			Updates(map[string]interface{}{
				"enabled":true,"secret_ciphertext":account.PendingCiphertext,
				"pending_ciphertext":"","pending_expires_at":int64(0),
				"last_used_step":step,
			})
		if res.Error!=nil{return res.Error}
		if res.RowsAffected!=1{return errors.New("authenticator setup changed")}
		codes=newCodes
		return nil
	})
	if err!=nil{return nil,err}
	return codes,nil
}

func verifyAndConsumeTOTP(tx *gorm.DB,account *model.UserTOTP,code string) error {
	if !account.Enabled{return errors.New("TOTP is not enabled")}
	now:=time.Now().Unix()
	if len(code)==6{
		secret,err:=decryptMailPassword(account.SecretCiphertext)
		if err!=nil{return err}
		step:=matchingTOTPStep(secret,code,now)
		if step>account.LastUsedStep {
			update:=tx.Model(&model.UserTOTP{}).
				Where("id = ? AND last_used_step < ?",account.Id,step).
				Update("last_used_step",step)
			if update.Error!=nil{return update.Error}
			if update.RowsAffected==1{return nil}
		}
	}
	if len(code)==20{
		key:=totpRecoveryHash(account.UserId,code)
		update:=tx.Model(&model.TOTPRecoveryCode{}).
			Where("user_id = ? AND code_hash = ? AND used_at = 0",
				account.UserId,key).
			Update("used_at",now)
		if update.Error!=nil{return update.Error}
		if update.RowsAffected==1{return nil}
	}
	return errors.New("incorrect or previously used authenticator code")
}

func RotateTOTPRecoveryCodes(userID uint,code string)([]string,error){
	var result []string
	err:=DB.Transaction(func(tx *gorm.DB)error{
		var account model.UserTOTP
		if err:=tx.Where("user_id = ?",userID).First(&account).Error;err!=nil{return err}
		if err:=verifyAndConsumeTOTP(tx,&account,code);err!=nil{return err}
		codes,err:=createTOTPRecoveryCodes(tx,userID)
		if err!=nil{return err}
		result=codes
		return nil
	})
	return result,err
}

func DisableTOTP(userID uint,code string)error{
	if userID==0{return errors.New("account ID is required")}
	return DB.Transaction(func(tx *gorm.DB)error{
		var account model.UserTOTP
		if err:=tx.Where("user_id = ?",userID).First(&account).Error;err!=nil{return err}
		if err:=verifyAndConsumeTOTP(tx,&account,code);err!=nil{return err}
		if err:=tx.Where("user_id = ?",userID).
			Delete(&model.TOTPRecoveryCode{}).Error;err!=nil{return err}
		if err:=tx.Where("user_id = ?",userID).
			Delete(&model.TOTPLoginChallenge{}).Error;err!=nil{return err}
		return tx.Delete(&account).Error
	})
}

func TOTPEnabled(userID uint)(bool,error){
	account,err:=totpAccount(userID)
	if err!=nil{return false,err}
	return account!=nil&&account.Enabled,nil
}

func StartTOTPLoginChallenge(userID uint)(string,error){
	if enabled,err:=TOTPEnabled(userID);err!=nil{return "",err}else if !enabled{
		return "",errors.New("TOTP is not enabled")
	}
	random:=make([]byte,32)
	if _,err:=rand.Read(random);err!=nil{return "",err}
	raw:=base64.RawURLEncoding.EncodeToString(random)
	hash:=sha256.Sum256([]byte(raw))
	now:=time.Now().Unix()
	return raw,DB.Create(&model.TOTPLoginChallenge{
		UserId:userID,TokenHash:hex.EncodeToString(hash[:]),
		ExpiresAt:now+totpChallengeSeconds,
	}).Error
}

func CompleteTOTPLoginChallenge(raw,username,code string) (*model.User,error){
	if len(raw)!=43||len(code)<6||len(code)>20{return nil,errors.New("invalid second-factor credentials")}
	hash:=sha256.Sum256([]byte(raw))
	digest:=hex.EncodeToString(hash[:])
	var user *model.User
	err:=DB.Transaction(func(tx *gorm.DB)error{
		now:=time.Now().Unix()
		var challenge model.TOTPLoginChallenge
		if err:=tx.Where("token_hash = ? AND used_at = 0 AND expires_at > ? AND attempts < ?",
			digest,now,totpMaxAttempts).First(&challenge).Error;err!=nil{
			return errors.New("second-factor verification has expired")
		}
		account:=&model.UserTOTP{}
		if err:=tx.Where("user_id = ? AND enabled = ?",challenge.UserId,true).
			First(account).Error;err!=nil{return errors.New("TOTP configuration is missing")}
		u:=&model.User{}
		if err:=tx.Where("id = ? AND username = ? AND status = ?",
			challenge.UserId,username,model.COMMON_STATUS_ENABLE).First(u).Error;err!=nil{
			return errors.New("user account is unavailable")
		}
		if err:=verifyAndConsumeTOTP(tx,account,code);err!=nil{
			// Count wrong attempts without committing a verified session.
			return err
		}
		r:=tx.Model(&model.TOTPLoginChallenge{}).
			Where("id = ? AND used_at = 0",challenge.Id).
			Update("used_at",now)
		if r.Error!=nil{return r.Error}
		if r.RowsAffected!=1{return errors.New("second-factor verification already completed")}
		user=u
		return nil
	})
	if err!=nil{
		_ = DB.Model(&model.TOTPLoginChallenge{}).
			Where("token_hash = ? AND used_at = 0 AND expires_at > ?",digest,time.Now().Unix()).
			Update("attempts",gorm.Expr("attempts + 1")).Error
		return nil,err
	}
	return user,nil
}

func SortedTOTPRecoveryCodes(codes []string) []string{
	copyCodes:=append([]string{},codes...)
	sort.Strings(copyCodes)
	return copyCodes
}
