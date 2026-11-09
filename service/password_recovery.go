package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/gorm"
)

const passwordResetLifetimeSeconds int64 = 900

func passwordResetHash(value string)string{
	sum:=sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// RequestPasswordReset uses an email address as the recovery destination.
// Unknown addresses receive the same response as valid addresses.
func RequestPasswordReset(email string)error{
	email=strings.TrimSpace(email)
	address,err:=mail.ParseAddress(email)
	if err!=nil||address.Address!=email||len(email)>253{
		return errors.New("valid email address is required")
	}
	settings,err:=currentMailSettings()
	if err!=nil{return err}
	if !settings.Enabled||settings.BaseURL==""{
		return errors.New("email recovery is not configured")
	}
	var user model.User
	err=DB.Where("email = ? AND status = ?",email,model.COMMON_STATUS_ENABLE).First(&user).Error
	if errors.Is(err,gorm.ErrRecordNotFound){return nil}
	if err!=nil{return err}
	now:=time.Now().Unix()
	var recent int64
	if err:=DB.Model(&model.PasswordResetToken{}).
		Where("user_id = ? AND requested_at > ?",user.Id,now-3600).
		Count(&recent).Error;err!=nil{return err}
	if recent>=5{return nil}
	secret:=make([]byte,32)
	if _,err:=rand.Read(secret);err!=nil{return err}
	raw:=base64.RawURLEncoding.EncodeToString(secret)
	record:=&model.PasswordResetToken{
		UserId:user.Id,TokenHash:passwordResetHash(raw),
		RequestedAt:now,ExpiresAt:now+passwordResetLifetimeSeconds,
	}
	if err:=DB.Create(record).Error;err!=nil{return err}
	confirmation:=strings.TrimRight(settings.BaseURL,"/")+"/_admin/#/password-reset?token="+url.QueryEscape(raw)
	body:=fmt.Sprintf("收到 RustDesk 账户密码恢复请求。\n\n请在 15 分钟内打开下面的地址并设置新密码：\n%s\n\n如果没有请求重设密码，可以忽略此邮件。\n",confirmation)
	if err:=sendSMTP(settings,email,"RustDesk 账户密码恢复",body);err!=nil{
		DB.Delete(record)
		return err
	}
	return nil
}

// CompletePasswordReset changes the local password and revokes login sessions.
// It does not modify OIDC or LDAP settings.
func CompletePasswordReset(raw,password string)error{
	if len(raw)!=43||len(password)<8||len(password)>72{
		return errors.New("invalid recovery token or new password")
	}
	newHash,err:=utils.EncryptPassword(password)
	if err!=nil{return err}
	now:=time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		var token model.PasswordResetToken
		if err:=tx.Where("token_hash = ? AND expires_at > ? AND used_at = 0",
			passwordResetHash(raw),now).First(&token).Error;err!=nil{
			return errors.New("recovery token is expired or already used")
		}
		var user model.User
		if err:=tx.Where("id = ? AND status = ?",token.UserId,model.COMMON_STATUS_ENABLE).
			First(&user).Error;err!=nil{return errors.New("account is no longer available")}
		changed:=tx.Model(&model.PasswordResetToken{}).
			Where("id = ? AND used_at = 0 AND expires_at > ?",token.Id,now).
			Update("used_at",now)
		if changed.Error!=nil{return changed.Error}
		if changed.RowsAffected==0{return errors.New("recovery token is already used")}
		if err:=tx.Model(&model.User{}).Where("id = ?",user.Id).
			Update("password",newHash).Error;err!=nil{return err}
		if err:=tx.Where("user_id = ?",user.Id).
			Delete(&model.UserToken{}).Error;err!=nil{return err}
		return nil
	})
}
