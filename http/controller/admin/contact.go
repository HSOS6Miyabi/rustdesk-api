package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm/clause"
)

type Contact struct {}

type contactRequest struct {
	UserId uint `json:"user_id"`
	Username string `json:"username"`
}

type contactAccount struct {
	UserId uint `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

func (ct *Contact) List(c *gin.Context) {
	current := service.AllService.UserService.CurUser(c)
	outgoing := make([]contactAccount, 0)
	incoming := make([]contactAccount, 0)
	db := service.DB
	err := db.Table("contacts").
		Select("users.id AS user_id, users.username, users.nickname").
		Joins("JOIN users ON users.id = contacts.recipient_user_id").
		Where("contacts.owner_user_id = ?", current.Id).
		Order("users.username").Scan(&outgoing).Error
	if err == nil {
		err = db.Table("contacts").
			Select("users.id AS user_id, users.username, users.nickname").
			Joins("JOIN users ON users.id = contacts.owner_user_id").
			Where("contacts.recipient_user_id = ?", current.Id).
			Order("users.username").Scan(&incoming).Error
	}
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"outgoing": outgoing, "incoming": incoming})
}

func resolveContactUser(req contactRequest) *model.User {
	if req.UserId > 0 {
		return service.AllService.UserService.InfoById(req.UserId)
	}
	if strings.TrimSpace(req.Username) != "" {
		return service.AllService.UserService.InfoByUsername(strings.TrimSpace(req.Username))
	}
	return nil
}

func (ct *Contact) Add(c *gin.Context) {
	var req contactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "参数无效: "+err.Error()); return
	}
	current := service.AllService.UserService.CurUser(c)
	recipient := resolveContactUser(req)
	if recipient == nil || recipient.Id == 0 || !service.AllService.UserService.CheckUserEnable(recipient) {
		response.Fail(c, 101, "用户不存在或已停用"); return
	}
	if recipient.Id == current.Id {
		response.Fail(c, 101, "无需将自己添加为联系人"); return
	}
	err := service.DB.Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.Contact{OwnerUserId: current.Id, RecipientUserId: recipient.Id}).Error
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"user_id": recipient.Id, "username": recipient.Username})
}

func (ct *Contact) Delete(c *gin.Context) {
	var req contactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "参数无效: "+err.Error()); return
	}
	target := resolveContactUser(req)
	if target == nil || target.Id == 0 {
		response.Fail(c, 101, "找不到指定的用户"); return
	}
	current := service.AllService.UserService.CurUser(c)
	err := service.DB.Where("owner_user_id = ? AND recipient_user_id = ?", current.Id, target.Id).
		Delete(&model.Contact{}).Error
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, nil)
}
