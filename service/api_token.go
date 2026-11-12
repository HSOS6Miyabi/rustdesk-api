package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const apiTokenPrefix = "rdapi_"
var apiTokenPermissions = map[string]bool{
	"devices.assign": true,
	"devices.deploy": true,
	"devices.read": true,
	"devices.manage": true,
	"deployments.read": true,
	"deployments.manage": true,
	"relays.read": true,
	"relays.manage": true,
	"sessions.read": true,
	"sessions.disconnect": true,
	"control_roles.read": true,
	"control_roles.manage": true,
	"access_rules.read": true,
	"access_rules.manage": true,
	"policies.read": true,
	"policies.manage": true,
	"audits.read": true,
    "events.read": true,
}

func normalizeAPITokenPermissions(requested []string) (string,error) {
	if len(requested)==0{return "",errors.New("at least one permission is required")}
	unique:=make(map[string]bool)
	for _,raw:=range requested {
		value:=strings.TrimSpace(raw)
		if !apiTokenPermissions[value] {
			return "",errors.New("unknown API token permission: "+value)
		}
		unique[value]=true
	}
	out:=make([]string,0,len(unique))
	for value:=range unique{out=append(out,value)}
	sort.Strings(out)
	return strings.Join(out,","),nil
}

func apiTokenHash(value string) string {
	digest:=sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// IssueAPIToken returns a plaintext secret exactly once; only its hash is saved.
func IssueAPIToken(userID uint,name string,scopes []string,days int) (*model.APIToken,string,error) {
	name=strings.TrimSpace(name)
	if userID==0||len([]rune(name))<2||len([]rune(name))>100||days<0||days>3650{
		return nil,"",errors.New("invalid API token settings")
	}
	permissions,err:=normalizeAPITokenPermissions(scopes)
	if err!=nil{return nil,"",err}
	var user model.User
	if err:=DB.Where("id = ? AND status = ?",userID,model.COMMON_STATUS_ENABLE).
		First(&user).Error;err!=nil{return nil,"",errors.New("active token owner was not found")}
	var count int64
	if err:=DB.Model(&model.APIToken{}).Where("user_id = ? AND revoked_at = 0",userID).
		Count(&count).Error;err!=nil{return nil,"",err}
	if count>=100{return nil,"",errors.New("this user has too many active API tokens")}
	key:=make([]byte,32)
	if _,err:=rand.Read(key);err!=nil{return nil,"",err}
	raw:=apiTokenPrefix+base64.RawURLEncoding.EncodeToString(key)
	var expires int64
	if days>0{expires=time.Now().AddDate(0,0,days).Unix()}
	item:=&model.APIToken{
		UserId:userID,Name:name,Prefix:raw[:14],TokenHash:apiTokenHash(raw),
		Permissions:permissions,ExpiresAt:expires,
	}
	if err:=DB.Create(item).Error;err!=nil{return nil,"",err}
	return item,raw,nil
}

func AuthenticateAPIToken(raw,permission string) (*model.User,error) {
	if len(raw)!=len(apiTokenPrefix)+43 || !strings.HasPrefix(raw,apiTokenPrefix) {
		return nil,errors.New("invalid API token")
	}
	var item model.APIToken
	if err:=DB.Where("token_hash = ?",apiTokenHash(raw)).First(&item).Error;err!=nil{
		return nil,errors.New("invalid API token")
	}
	now:=time.Now().Unix()
	if item.RevokedAt!=0||(item.ExpiresAt!=0&&item.ExpiresAt<=now){
		return nil,errors.New("API token expired or revoked")
	}
	allowed:=false
	for _,scope:=range strings.Split(item.Permissions,","){
		if scope==permission{allowed=true;break}
	}
	if !allowed{return nil,errors.New("API token does not have the required permission")}
	var user model.User
	if err:=DB.Where("id = ? AND status = ?",item.UserId,model.COMMON_STATUS_ENABLE).
		First(&user).Error;err!=nil {
		return nil,errors.New("API token account is no longer active")
	}
	if now-item.LastUsedAt >= 60 {
		_ = DB.Model(&model.APIToken{}).
			Where("id = ? AND last_used_at <= ?", item.Id, now-60).
			UpdateColumn("last_used_at", now).Error
	}
	return &user,nil
}

func RevokeAPIToken(id uint) error {
	if id==0{return errors.New("token ID is required")}
	result:=DB.Model(&model.APIToken{}).Where("id = ?",id).
		Update("revoked_at",time.Now().Unix())
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

// ListPushStreamTokens exposes only dedicated subscription keys owned by the user.
func ListPushStreamTokens(userID uint) ([]model.APIToken, error) {
    if userID == 0 { return nil, errors.New("user required") }
    var keys []model.APIToken
    err := DB.Where("user_id = ? AND permissions = ?", userID, "events.read").
        Order("id DESC").Limit(100).Find(&keys).Error
    return keys, err
}

// RevokePushStreamToken cannot modify another user's or a management key.
func RevokePushStreamToken(userID, id uint) error {
    if userID == 0 || id == 0 { return errors.New("invalid API key") }
    result := DB.Model(&model.APIToken{}).
        Where("id = ? AND user_id = ? AND permissions = ?", id, userID, "events.read").
        Update("revoked_at", time.Now().Unix())
    if result.Error != nil { return result.Error }
    if result.RowsAffected != 1 { return gorm.ErrRecordNotFound }
    return nil
}
