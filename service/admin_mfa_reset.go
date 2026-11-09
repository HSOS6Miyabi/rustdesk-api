package service

import (
	"errors"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// ResetUserSecondFactors lets an administrator recover an account that no
// longer has access to its authenticator or registered email address.
func ResetUserSecondFactors(userID uint) error {
	if userID==0{return errors.New("valid account ID is required")}
	return DB.Transaction(func(tx *gorm.DB)error{
		var count int64
		if err:=tx.Model(&model.User{}).Where("id = ?",userID).Count(&count).Error;err!=nil{return err}
		if count!=1{return gorm.ErrRecordNotFound}
		for _,item:=range []interface{}{
			&model.UserTOTP{},&model.TOTPRecoveryCode{},&model.TOTPLoginChallenge{},
			&model.EmailLoginSetting{},&model.EmailCodeChallenge{},
			&model.UserToken{},
		}{
			if err:=tx.Where("user_id = ?",userID).Delete(item).Error;err!=nil{return err}
		}
		return nil
	})
}
