package service

import (
	crand "crypto/rand"
	"encoding/base64"
	"errors"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/gorm"
)

type UserService struct {
}

// InfoById 根据用户id取用户信息
func (us *UserService) InfoById(id uint) *model.User {
	u := &model.User{}
	DB.Where("id = ?", id).First(u)
	return u
}

// InfoByUsername 根据用户名取用户信息
func (us *UserService) InfoByUsername(un string) *model.User {
	u := &model.User{}
	DB.Where("username = ?", un).First(u)
	return u
}

// InfoByEmail 根据邮箱取用户信息
func (us *UserService) InfoByEmail(email string) *model.User {
	u := &model.User{}
	DB.Where("email = ?", email).First(u)
	return u
}

// InfoByOpenid 根据openid取用户信息
func (us *UserService) InfoByOpenid(openid string) *model.User {
	u := &model.User{}
	DB.Where("openid = ?", openid).First(u)
	return u
}

// InfoByUsernamePassword authenticates a configured local password first.
// A local credential remains available when LDAP cannot be reached.
func (us *UserService) InfoByUsernamePassword(username, password string) *model.User {
	u := &model.User{}
	if err := DB.Where("username = ?", username).First(u).Error; err == nil && u.Id != 0 && u.Password != "" {
		ok, newHash, verifyErr := utils.VerifyPassword(u.Password, password)
		if verifyErr == nil && ok {
			if newHash != "" {
				DB.Model(u).Update("password", newHash)
				u.Password = newHash
			}
			return u
		}
	}
	if Config.Ldap.Enable {
		ldapUser, err := AllService.LdapService.Authenticate(username, password)
		if err == nil {
			return ldapUser
		}
		Logger.WithError(err).Warn("LDAP authentication failed; local password did not match")
	}
	return &model.User{}
}

// InfoByAccessToken retrieves the active token and account in one SQL query.
// Checking the database on every request preserves immediate revocation.
func (us *UserService) InfoByAccessToken(token string) (*model.User, *model.UserToken) {
	u := &model.User{}
	ut := &model.UserToken{}
	if token == "" || DB == nil {
		return u, ut
	}
	var row struct {
		model.User `gorm:"embedded"`
		TokenRowID uint `gorm:"column:token_row_id"`
		TokenUserID uint `gorm:"column:token_user_id"`
		TokenDeviceID string `gorm:"column:token_device_id"`
		TokenDeviceUUID string `gorm:"column:token_device_uuid"`
		TokenExpiredAt int64 `gorm:"column:token_expired_at"`
	}
	err := DB.Table("user_tokens AS tokens").
		Select("users.*, tokens.id AS token_row_id, tokens.user_id AS token_user_id, "+
			"tokens.device_id AS token_device_id, tokens.device_uuid AS token_device_uuid, "+
			"tokens.expired_at AS token_expired_at").
		Joins("JOIN users ON users.id = tokens.user_id").
		Where("tokens.token = ?", token).Take(&row).Error
	if err != nil || row.TokenRowID == 0 {
		return u, ut
	}
	ut.Id = row.TokenRowID
	ut.UserId = row.TokenUserID
	ut.DeviceId = row.TokenDeviceID
	ut.DeviceUuid = row.TokenDeviceUUID
	ut.Token = token
	ut.ExpiredAt = row.TokenExpiredAt
	if ut.ExpiredAt <= time.Now().Unix() {
		return u, ut
	}
	// Match hbbs verification for JWT sessions, including expiration.
	if strings.Count(token, ".") == 2 {
		if Jwt == nil || len(Jwt.Key) == 0 { return u, ut }
		id, err := Jwt.ParseToken(token)
		if err != nil || id != row.User.Id { return u, ut }
	}
	return &row.User, ut
}

// GenerateToken 生成token
func (us *UserService) GenerateToken(u *model.User) string {
	if Jwt != nil && len(Jwt.Key) > 0 {
		// hbbs verifies signed access tokens when MUST_LOGIN is enabled.
		signer := *Jwt
		signer.TokenExpireDuration = loginSessionDuration()
		return signer.GenerateToken(u.Id)
	}
	secret := make([]byte, 32)
	if _, err := crand.Read(secret); err != nil {
		panic("cannot create secure access token: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(secret)
}

// Login 登录
func (us *UserService) Login(u *model.User, llog *model.LoginLog) *model.UserToken {
	token := us.GenerateToken(u)
	ut := &model.UserToken{
		UserId:     u.Id,
		Token:      token,
		DeviceUuid: llog.Uuid,
		DeviceId:   llog.DeviceId,
		ExpiredAt:  us.UserTokenExpireTimestamp(),
	}
	DB.Create(ut)
	llog.UserTokenId = ut.Id
	DB.Create(llog)
	if llog.Uuid != "" {
		AllService.PeerService.UuidBindUserId(llog.DeviceId, llog.Uuid, u.Id)
	}
	return ut
}

// CurUser 获取当前用户
func (us *UserService) CurUser(c *gin.Context) *model.User {
	user, _ := c.Get("curUser")
	u, ok := user.(*model.User)
	if !ok {
		return nil
	}
	return u
}

func (us *UserService) List(page, pageSize uint, where func(tx *gorm.DB), sort ...string) (res *model.UserList) {
	res = &model.UserList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.User{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page,pageSize),SortScope(sort...))
	tx.Find(&res.Users)
	return
}

func (us *UserService) ListByIds(ids []uint) (res []*model.User) {
	DB.Where("id in ?", ids).Find(&res)
	return res
}

// ListByGroupId 根据组id取用户列表
func (us *UserService) ListByGroupId(groupId, page, pageSize uint) (res *model.UserList) {
	res = us.List(page, pageSize, func(tx *gorm.DB) {
		tx.Where("group_id = ?", groupId)
	})
	return
}

// ListIdsByGroupId 根据组id取用户id列表
func (us *UserService) ListIdsByGroupId(groupId uint) (ids []uint) {
	DB.Model(&model.User{}).Where("group_id = ?", groupId).Pluck("id", &ids)
	return ids

}

// ListIdAndNameByGroupId 根据组id取用户id和用户名列表
func (us *UserService) ListIdAndNameByGroupId(groupId uint) (res []*model.User) {
	DB.Model(&model.User{}).Where("group_id = ?", groupId).Select("id, username").Find(&res)
	return res
}

// CheckUserEnable 判断用户是否禁用
func (us *UserService) CheckUserEnable(u *model.User) bool {
	return u.Status == model.COMMON_STATUS_ENABLE
}

// Create 创建
func (us *UserService) Create(u *model.User) error {
	// The initial username should be formatted, and the username should be unique
	if us.IsUsernameExists(u.Username) {
		return errors.New("UsernameExists")
	}
	u.Username = us.formatUsername(u.Username)
	var err error
	u.Password, err = utils.EncryptPassword(u.Password)
	if err != nil {
		return err
	}
	res := DB.Create(u).Error
	return res
}

// GetUuidByToken 根据token和user取uuid
func (us *UserService) GetUuidByToken(u *model.User, token string) string {
	ut := &model.UserToken{}
	err := DB.Where("user_id = ? and token = ?", u.Id, token).First(ut).Error
	if err != nil {
		return ""
	}
	return ut.DeviceUuid
}

// Logout 退出登录 -> 删除token, 解绑uuid
func (us *UserService) Logout(u *model.User, token string) error {
	uuid := us.GetUuidByToken(u, token)
	err := DB.Where("user_id = ? and token = ?", u.Id, token).Delete(&model.UserToken{}).Error
	if err != nil {
		return err
	}
	if uuid != "" {
		AllService.PeerService.UuidUnbindUserId(uuid, u.Id)
	}
	return nil
}

// Delete 删除用户和oauth信息
func (us *UserService) Delete(u *model.User) error {
	if u == nil || u.Id == 0 { return errors.New("user ID is required") }
	current := us.InfoById(u.Id)
	if current.Id == 0 { return errors.New("user does not exist") }
	if us.IsAdmin(current) && us.getAdminUserCount() <= 1 {
		return errors.New("The last admin user cannot be deleted")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var ownedGroups []uint
		if err := tx.Model(&model.Group{}).Where("owner_user_id = ?", u.Id).
			Pluck("id", &ownedGroups).Error; err != nil { return err }
		var ownedCollections []uint
		if err := tx.Model(&model.AddressBookCollection{}).Where("user_id = ?",u.Id).
			Pluck("id", &ownedCollections).Error; err != nil { return err }

		if len(ownedGroups) > 0 {
			if err := tx.Model(&model.User{}).Where("group_id IN ?",ownedGroups).
				Update("group_id",1).Error; err != nil { return err }
			for _, row := range []interface{}{&model.GroupMembership{},&model.GroupAnchor{}} {
				if err := tx.Where("group_id IN ?",ownedGroups).Delete(row).Error; err != nil { return err }
			}
			if err := tx.Where("id IN ?",ownedGroups).Delete(&model.Group{}).Error; err != nil { return err }
			if err := tx.Where("type = ? AND to_id IN ?",model.ShareAddressBookRuleTypeGroup,ownedGroups).
				Delete(&model.AddressBookCollectionRule{}).Error; err != nil { return err }
		}
		if len(ownedCollections) > 0 {
			if err := tx.Where("collection_id IN ?",ownedCollections).
				Delete(&model.AddressBookCollectionRule{}).Error; err != nil { return err }
		}
		if err := tx.Where("owner_user_id = ? OR recipient_user_id = ?",u.Id,u.Id).
			Delete(&model.Contact{}).Error; err != nil { return err }
		if err := tx.Where("user_id = ?",u.Id).
			Delete(&model.GroupMembership{}).Error; err != nil { return err }
		if err := tx.Where("user_id = ?",u.Id).
			Delete(&model.GroupAnchor{}).Error; err != nil { return err }
		if err := tx.Where("type = ? AND to_id = ?",model.ShareAddressBookRuleTypePersonal,u.Id).
			Delete(&model.AddressBookCollectionRule{}).Error; err != nil { return err }
		for _, row := range []interface{}{
			&model.UserThird{},&model.AddressBookCollection{},
			&model.AddressBookCollectionRule{},&model.Tag{},
			&model.UserToken{},&model.ShareRecord{},&model.DevicePolicyUserAssignment{},&model.APIToken{},
			&model.UserTOTP{},&model.TOTPRecoveryCode{},&model.TOTPLoginChallenge{},&model.PasswordResetToken{},
			&model.EmailLoginSetting{},&model.EmailCodeChallenge{},
		} {
			if err := tx.Where("user_id = ?",u.Id).Delete(row).Error; err != nil { return err }
		}

        for _,row:=range []interface{}{&model.UserPasskey{},&model.UserPasskeyHandle{},&model.UserPasskeyChallenge{}}{
            if tx.Migrator().HasTable(row){
                if err:=tx.Where("user_id = ?",u.Id).Delete(row).Error;err!=nil{return err}
            }
        }

		if tx.Migrator().HasTable(&model.PushDelivery{}) {
			if err := tx.Where("user_id = ?",u.Id).Delete(&model.PushDelivery{}).Error; err != nil { return err }
		}
		if tx.Migrator().HasTable(&model.PushChannel{}) {
			if err := tx.Where("user_id = ?",u.Id).Delete(&model.PushChannel{}).Error; err != nil { return err }
		}
		if err:=tx.Model(&model.UserInvitation{}).
			Where("created_by = ? AND accepted_at = 0 AND revoked_at = 0",u.Id).
			Update("revoked_at",time.Now().Unix()).Error;err!=nil{return err}

		if err:=tx.Where("target_type = ? AND target_id = ?",
			controlTargetUser,strconv.FormatUint(uint64(u.Id),10)).
			Delete(&model.ControlRoleAssignment{}).Error;err!=nil{return err}
		if tx.Migrator().HasTable(&model.AccessRule{}) {
			if err:=tx.Where("source_type = ? AND source_id = ?",accessSourceUser,u.Id).
				Or("target_type = ? AND target_id = ?",accessTargetUser,strconv.FormatUint(uint64(u.Id),10)).
				Delete(&model.AccessRule{}).Error;err!=nil{return err}
			if len(ownedGroups)>0 {
				if err:=tx.Where("source_type = ? AND source_id IN ?",accessSourceGroup,ownedGroups).
					Delete(&model.AccessRule{}).Error;err!=nil{return err}
			}
		}
		if tx.Migrator().HasTable(&model.AdminRoleAssignment{}) {
			if err:=tx.Where("user_id = ?",u.Id).
				Delete(&model.AdminRoleAssignment{}).Error;err!=nil{return err}
		}
		if tx.Migrator().HasTable(&model.AdminRole{}) {
			var scopedRoles []uint
			scopeQuery:=tx.Model(&model.AdminRole{}).
				Where("scope_type = ? AND scope_id = ?",AdminScopeIndividual,u.Id)
			if len(ownedGroups)>0 {
				scopeQuery=scopeQuery.Or("scope_type = ? AND scope_id IN ?",AdminScopeGroup,ownedGroups)
			}
			if err:=scopeQuery.Pluck("id",&scopedRoles).Error;err!=nil{return err}
			if len(scopedRoles)>0{
				if err:=tx.Where("role_id IN ?",scopedRoles).
					Delete(&model.AdminRoleAssignment{}).Error;err!=nil{return err}
				if err:=tx.Where("id IN ?",scopedRoles).
					Delete(&model.AdminRole{}).Error;err!=nil{return err}
			}
		}
		if err:=tx.Unscoped().Where("user_id = ?",u.Id).
			Delete(&model.AddressBook{}).Error;err!=nil{return err}
		if err := tx.Model(&model.Peer{}).Where("user_id = ?",u.Id).
			Update("user_id",0).Error; err != nil { return err }
		return tx.Delete(current).Error
	})
}

// Update 更新
func (us *UserService) Update(u *model.User) error {
	currentUser := us.InfoById(u.Id)
	// 如果当前用户是管理员并且 IsAdmin 不为空，进行检查
	if us.IsAdmin(currentUser) {
		adminCount := us.getAdminUserCount()
		// 如果这是唯一的管理员，确保不能禁用或取消管理员权限
		if adminCount <= 1 && (!us.IsAdmin(u) || u.Status == model.COMMON_STATUS_DISABLED) {
			return errors.New("The last admin user cannot be disabled or demoted")
		}
	}
	return DB.Model(u).Updates(u).Error
}

// FlushToken 清空token
func (us *UserService) FlushToken(u *model.User) error {
	return DB.Where("user_id = ?", u.Id).Delete(&model.UserToken{}).Error
}

// FlushTokenByUuid 清空token
func (us *UserService) FlushTokenByUuid(uuid string) error {
	return DB.Where("device_uuid = ?", uuid).Delete(&model.UserToken{}).Error
}

// FlushTokenByUuids 清空token
func (us *UserService) FlushTokenByUuids(uuids []string) error {
	return DB.Where("device_uuid in (?)", uuids).Delete(&model.UserToken{}).Error
}

// UpdatePassword 更新密码
func (us *UserService) UpdatePassword(u *model.User, password string) error {
	var err error
	u.Password, err = utils.EncryptPassword(password)
	if err != nil {
		return err
	}
	err = DB.Model(u).Update("password", u.Password).Error
	if err != nil {
		return err
	}
	err = us.FlushToken(u)
	return err
}

var ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")
var ErrCurrentPasswordChanged = errors.New("account password changed during the update")

// ChangeOwnPassword creates a local password for an externally authenticated
// account or changes an existing one, then revokes existing login sessions.
func (us *UserService) ChangeOwnPassword(userID uint, oldPassword, newPassword string) error {
	if userID == 0 {
		return gorm.ErrRecordNotFound
	}
	if len([]rune(newPassword)) < 8 || len(newPassword) > 72 {
		return errors.New("new password must have 8 to 72 characters and fit within 72 bytes")
	}
	u := us.InfoById(userID)
	if u.Id == 0 {
		return gorm.ErrRecordNotFound
	}
	if u.Password != "" {
		ok, _, err := utils.VerifyPassword(u.Password, oldPassword)
		if err != nil || !ok {
			return ErrCurrentPasswordIncorrect
		}
	}
	hashed, err := utils.EncryptPassword(newPassword)
	if err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&model.User{}).
			Where("id = ? AND password = ?", userID, u.Password).
			Update("password", hashed)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return ErrCurrentPasswordChanged
		}
		return tx.Where("user_id = ?", userID).Delete(&model.UserToken{}).Error
	})
}

// IsAdmin 是否管理员
func (us *UserService) IsAdmin(u *model.User) bool {
	return u != nil && u.IsAdmin != nil && *u.IsAdmin
}

// RouteNames
func (us *UserService) RouteNames(u *model.User) []string {
	if us.IsAdmin(u) {
		return model.AdminRouteNames
	}
	names:=append([]string{},model.UserRouteNames...)
	if DB==nil || !DB.Migrator().HasTable(&model.AdminRoleAssignment{}) {
		return names
	}
	permissions,err:=AdminRolePermissionNames(u)
	if err==nil{
		names=append(names,permissions...)
	}
	return names
}

// InfoByOauthId 根据oauth的name和openId取用户信息
func (us *UserService) InfoByOauthId(op string, openId string) *model.User {
	ut := AllService.OauthService.UserThirdInfo(op, openId)
	if ut.Id == 0 {
		return nil
	}
	u := us.InfoById(ut.UserId)
	if u.Id == 0 {
		return nil
	}
	return u
}

// RegisterByOauth 注册
func (us *UserService) RegisterByOauth(oauthUser *model.OauthUser, op string) (error, *model.User) {
	Lock.Lock("registerByOauth")
	defer Lock.UnLock("registerByOauth")
	ut := AllService.OauthService.UserThirdInfo(op, oauthUser.OpenId)
	if ut.Id != 0 {
		return nil, us.InfoById(ut.UserId)
	}
	err, oauthType := AllService.OauthService.GetTypeByOp(op)
	if err != nil {
		return err, nil
	}
	//check if this email has been registered
	email := oauthUser.Email
	// only email is not empty
	if email != "" {
		email = strings.ToLower(email)
		// update email to oauthUser, in case it contain upper case
		oauthUser.Email = email
		// call this, if find user by email, it will update the email to local database
		user, ldapErr := AllService.LdapService.GetUserInfoByEmailLocal(email)
		// If we enable ldap, and the error is not ErrLdapUserNotFound, return the error because we could not sure if the user is not found in ldap
		if !(errors.Is(ldapErr, ErrLdapNotEnabled) || errors.Is(ldapErr, ErrLdapUserNotFound) || ldapErr == nil) {
			return ldapErr, user
		}
		if user.Id == 0 {
			// this means the user is not found in ldap, maybe ldao is not enabled
			user = us.InfoByEmail(email)
		}
		if user.Id != 0 {
			ut.FromOauthUser(user.Id, oauthUser, oauthType, op)
			DB.Create(ut)
			return nil, user
		}
	}

	tx := DB.Begin()
	ut = &model.UserThird{}
	ut.FromOauthUser(0, oauthUser, oauthType, op)
	// The initial username should be formatted
	username := us.formatUsername(oauthUser.Username)
	usernameUnique := us.GenerateUsernameByOauth(username)
	user := &model.User{
		Username: usernameUnique,
		GroupId:  1,
	}
	oauthUser.ToUser(user, false)
	tx.Create(user)
	if user.Id == 0 {
		tx.Rollback()
		return errors.New("OauthRegisterFailed"), user
	}
	ut.UserId = user.Id
	tx.Create(ut)
	tx.Commit()
	return nil, user
}

// GenerateUsernameByOauth 生成用户名
func (us *UserService) GenerateUsernameByOauth(name string) string {
	for us.IsUsernameExists(name) {
		name += strconv.Itoa(rand.Intn(10)) // Append a random digit (0-9)
	}
	return name
}

// UserThirdsByUserId
func (us *UserService) UserThirdsByUserId(userId uint) (res []*model.UserThird) {
	DB.Where("user_id = ?", userId).Find(&res)
	return res
}

func (us *UserService) UserThirdInfo(userId uint, op string) *model.UserThird {
	ut := &model.UserThird{}
	DB.Where("user_id = ? and op = ?", userId, op).First(ut)
	return ut
}

// FindLatestUserIdFromLoginLogByUuid 根据uuid和设备id查找最后登录的用户id
func (us *UserService) FindLatestUserIdFromLoginLogByUuid(uuid string, deviceId string) uint {
	llog := &model.LoginLog{}
	DB.Where("uuid = ? and device_id = ?", uuid, deviceId).Order("id desc").First(llog)
	return llog.UserId
}

// IsPasswordEmptyById 根据用户id判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyById(id uint) bool {
	u := &model.User{}
	if DB.Where("id = ?", id).First(u).Error != nil {
		return false
	}
	return u.Password == ""
}

// IsPasswordEmptyByUsername 根据用户id判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyByUsername(username string) bool {
	u := &model.User{}
	if DB.Where("username = ?", username).First(u).Error != nil {
		return false
	}
	return u.Password == ""
}

// IsPasswordEmptyByUser 判断密码是否为空，主要用于第三方登录的自动注册
func (us *UserService) IsPasswordEmptyByUser(u *model.User) bool {
	return us.IsPasswordEmptyById(u.Id)
}

// Register 注册, 如果用户名已存在则返回nil
func (us *UserService) Register(username string, email string, password string, status model.StatusCode) *model.User {
	u := &model.User{
		Username: username,
		Email:    email,
		Password: password,
		GroupId:  1,
		Status:   status,
	}
	err := us.Create(u)
	if err != nil {
		return nil
	}
	return u
}

func (us *UserService) TokenList(page uint, size uint, f func(tx *gorm.DB), sort ...string) *model.UserTokenList {
	res := &model.UserTokenList{}
	res.Page = int64(page)
	res.PageSize = int64(size)
	tx := DB.Model(&model.UserToken{})
	if f != nil {
		f(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page,size),SortScope(sort...))
	tx.Find(&res.UserTokens)
	return res
}

func (us *UserService) TokenInfoById(id uint) *model.UserToken {
	ut := &model.UserToken{}
	DB.Where("id = ?", id).First(ut)
	return ut
}

func (us *UserService) DeleteToken(l *model.UserToken) error {
	return DB.Delete(l).Error
}

// Helper functions, used for formatting username
func (us *UserService) formatUsername(username string) string {
	username = strings.ReplaceAll(username, " ", "")
	username = strings.ToLower(username)
	return username
}

// Helper functions, getUserCount
func (us *UserService) getUserCount() int64 {
	var count int64
	DB.Model(&model.User{}).Count(&count)
	return count
}

// helper functions, getAdminUserCount
func (us *UserService) getAdminUserCount() int64 {
	var count int64
	DB.Model(&model.User{}).Where("is_admin = ?", true).Count(&count)
	return count
}

const defaultLoginSessionDuration = 30 * 24 * time.Hour

func loginSessionDuration() time.Duration {
	if Config == nil || Config.App.TokenExpire <= 0 {
		return defaultLoginSessionDuration
	}
	return Config.App.TokenExpire
}

// UserTokenExpireTimestamp calculates the expiration of a persistent session.
func (us *UserService) UserTokenExpireTimestamp() int64 {
	return time.Now().Add(loginSessionDuration()).Unix()
}

func (us *UserService) RefreshAccessToken(ut *model.UserToken) {
	if ut == nil || ut.Id == 0 { return }
	newExpiration := us.UserTokenExpireTimestamp()
	result := DB.Model(&model.UserToken{}).
		Where("id = ? AND expired_at > ?", ut.Id, time.Now().Unix()).
		Update("expired_at", newExpiration)
	if result.Error == nil && result.RowsAffected > 0 {
		ut.ExpiredAt = newExpiration
	}
}

func (us *UserService) AutoRefreshAccessToken(ut *model.UserToken) {
	if ut == nil || ut.Id == 0 { return }
	// A signed JWT cannot be extended without replacing the client's token.
	if strings.Count(ut.Token, ".") == 2 { return }
	remaining := time.Until(time.Unix(ut.ExpiredAt, 0))
	if remaining > 0 && remaining <= loginSessionDuration()/3 {
		us.RefreshAccessToken(ut)
	}
}

func (us *UserService) BatchDeleteUserToken(ids []uint) error {
	return DB.Where("id in ?", ids).Delete(&model.UserToken{}).Error
}

func (us *UserService) VerifyJWT(token string) (uint, error) {
	return Jwt.ParseToken(token)
}

// IsUsernameExists 判断用户名是否存在, it will check the internal database and LDAP(if enabled)
func (us *UserService) IsUsernameExists(username string) bool {
	return us.IsUsernameExistsLocal(username) || AllService.LdapService.IsUsernameExists(username)
}

func (us *UserService) IsUsernameExistsLocal(username string) bool {
	u := &model.User{}
	DB.Where("username = ?", username).First(u)
	return u.Id != 0
}

func (us *UserService) IsEmailExistsLdap(email string) bool {
	return AllService.LdapService.IsEmailExists(email)
}
