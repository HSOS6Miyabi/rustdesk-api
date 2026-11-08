package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type groupMembersRequest struct {
	GroupId uint `json:"group_id" binding:"required"`
	UserIds []uint `json:"user_ids"`
	Action string `json:"action"`
}

type groupUser struct {
	Id uint `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

func memberGroupID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Query("group_id"), 10, 32)
	if err != nil || id == 0 {
		response.Fail(c, 101, "需要有效的群组 ID")
		return 0, false
	}
	return uint(id), true
}

func publicUsers(ids []uint) ([]groupUser, error) {
	users := make([]groupUser, 0)
	if len(ids) == 0 {
		return users, nil
	}
	err := service.DB.Model(&model.User{}).Select("id, username, nickname").
		Where("id IN ?", ids).Order("username").Find(&users).Error
	return users, err
}

func (ct *Group) Members(c *gin.Context) {
	id, ok := memberGroupID(c)
	if !ok { return }
	ids, err := service.AllService.GroupService.MemberIds(id)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	users, err := publicUsers(ids)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"group_id": id, "user_ids": ids, "users": users})
}

func (ct *Group) BatchMembers(c *gin.Context) {
	var req groupMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "参数无效: "+err.Error()); return
	}
	if len(req.UserIds) > 10000 {
		response.Fail(c, 101, "单次操作最多包含 10000 名用户"); return
	}
	if err := service.AllService.GroupService.ChangeMembers(req.GroupId, req.UserIds, req.Action); err != nil {
		response.Fail(c, 101, err.Error()); return
	}
	response.Success(c, nil)
}

func (ct *Group) Anchors(c *gin.Context) {
	id, ok := memberGroupID(c)
	if !ok { return }
	ids, err := service.AllService.GroupService.AnchorIds(id)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	users, err := publicUsers(ids)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"group_id": id, "user_ids": ids, "users": users})
}

func (ct *Group) SetAnchors(c *gin.Context) {
	var req groupMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, 101, "参数无效: "+err.Error()); return
	}
	if len(req.UserIds) > 10000 {
		response.Fail(c, 101, "单次操作最多包含 10000 名用户"); return
	}
	if err := service.AllService.GroupService.ReplaceAnchors(req.GroupId, req.UserIds); err != nil {
		response.Fail(c, 101, err.Error()); return
	}
	response.Success(c, nil)
}
