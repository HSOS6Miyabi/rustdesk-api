package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/gorm"
)

var invitationUsername = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,63}$`)

func InviteUser(email string,groupID,inviterID uint,days int)error{
	email=strings.TrimSpace(email)
	parsed,err:=mail.ParseAddress(email)
	if err!=nil||parsed.Address!=email||len(email)>253{return errors.New("valid invitee email address is required")}
	if inviterID==0||days<1||days>30{return errors.New("invalid invitation parameters")}
	if groupID==0{groupID=1}
	var groupCount int64
	if err:=DB.Model(&model.Group{}).Where("id = ?",groupID).Count(&groupCount).Error;err!=nil{return err}
	if groupCount!=1{return errors.New("user group does not exist")}
	var exists int64
	if err:=DB.Model(&model.User{}).Where("email = ?",email).Count(&exists).Error;err!=nil{return err}
	if exists!=0{return errors.New("email already belongs to an account")}
	now:=time.Now().Unix()
	if err:=DB.Model(&model.UserInvitation{}).Where(
		"email = ? AND accepted_at = 0 AND revoked_at = 0 AND expires_at > ?",email,now).
		Count(&exists).Error;err!=nil{return err}
	if exists!=0{return errors.New("an active invitation already exists for this email")}
	settings,err:=currentMailSettings()
	if err!=nil{return err}
	if !settings.Enabled||settings.BaseURL==""{
		return errors.New("configure SMTP and the public console URL before inviting users")
	}
	secret:=make([]byte,32)
	if _,err:=rand.Read(secret);err!=nil{return err}
	raw:=base64.RawURLEncoding.EncodeToString(secret)
	invitation:=&model.UserInvitation{
		Email:email,GroupId:groupID,CreatedBy:inviterID,
		TokenHash:passwordResetHash(raw),ExpiresAt:time.Now().AddDate(0,0,days).Unix(),
	}
	if err:=DB.Create(invitation).Error;err!=nil{return err}
	link:=strings.TrimRight(settings.BaseURL,"/")+"/_admin/#/invite?token="+url.QueryEscape(raw)
	body:=fmt.Sprintf("你收到了 RustDesk 自托管团队的账户邀请。\n\n请在 %d 天内打开以下地址创建账户：\n%s\n\n如果不需要加入该团队，可以忽略这封邮件。\n",days,link)
	if err:=sendSMTP(settings,email,"RustDesk 团队账户邀请",body);err!=nil{
		DB.Delete(invitation)
		return err
	}
	return nil
}

func AcceptUserInvitation(raw,username,nickname,password string) error{
	username=strings.ToLower(strings.TrimSpace(username))
	nickname=strings.TrimSpace(nickname)
	if len(raw)!=43||!invitationUsername.MatchString(username)||
		len(password)<8||len(password)>72||len([]rune(nickname))>100{
		return errors.New("invalid account details or invitation")
	}
	if nickname==""{nickname=username}
	secret,err:=utils.EncryptPassword(password)
	if err!=nil{return err}
	now:=time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB)error{
		var invitation model.UserInvitation
		if err:=tx.Where("token_hash = ? AND accepted_at = 0 AND revoked_at = 0 AND expires_at > ?",
			passwordResetHash(raw),now).First(&invitation).Error;err!=nil{
			return errors.New("invitation is expired or already used")
		}
		var existing int64
		if err:=tx.Model(&model.User{}).
			Where("email = ? OR username = ?",invitation.Email,username).
			Count(&existing).Error;err!=nil{return err}
		if existing!=0{return errors.New("email or username is already registered")}
		result:=tx.Model(&model.UserInvitation{}).
			Where("id = ? AND accepted_at = 0 AND revoked_at = 0 AND expires_at > ?",invitation.Id,now).
			Update("accepted_at",now)
		if result.Error!=nil{return result.Error}
		if result.RowsAffected!=1{return errors.New("invitation has already been used")}
		account:=&model.User{
			Username:username,Nickname:nickname,Email:invitation.Email,
			Password:secret,Status:model.COMMON_STATUS_ENABLE,
			GroupId:invitation.GroupId,
		}
		return tx.Create(account).Error
	})
}

func ListUserInvitations(page,pageSize uint) ([]model.UserInvitation,int64,error){
	var rows []model.UserInvitation
	var total int64
	query:=DB.Model(&model.UserInvitation{})
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	if err:=query.Order("id DESC").Scopes(Paginate(page,pageSize)).
		Find(&rows).Error;err!=nil{return nil,0,err}
	return rows,total,nil
}

func RevokeUserInvitation(id uint)error{
	if id==0{return errors.New("invitation ID is required")}
	result:=DB.Model(&model.UserInvitation{}).
		Where("id = ? AND accepted_at = 0 AND revoked_at = 0",id).
		Update("revoked_at",time.Now().Unix())
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}
