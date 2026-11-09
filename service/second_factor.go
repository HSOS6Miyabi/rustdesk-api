package service

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// CompleteSecondFactorLogin accepts TOTP, recovery code, or an emailed code.
// The opaque challenge identifies the method; clients share the same login route.
func CompleteSecondFactorLogin(secret,username,code string) (*model.User,error) {
	digest:=sha256.Sum256([]byte(secret))
	hash:=hex.EncodeToString(digest[:])
	var row model.TOTPLoginChallenge
	err:=DB.Where("token_hash = ?",hash).First(&row).Error
	if err==nil{
		return CompleteTOTPLoginChallenge(secret,username,code)
	}
	if err!=gorm.ErrRecordNotFound{return nil,err}
	return CompleteEmailLogin(secret,username,code)
}
