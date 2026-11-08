package admin

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type MyGroup struct{}

type personalGroupForm struct {
	Id uint `json:"id"`
	Name string `json:"name" binding:"required"`
	Type int `json:"type" binding:"required"`
}

func myGroupId(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Query("group_id"), 10, 32)
	if err != nil || id == 0 {
		response.Fail(c, 101, "需要有效的群组 ID")
		return 0, false
	}
	return uint(id), true
}

func (m *MyGroup) List(c *gin.Context) {
	queryPage, _ := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	querySize, _ := strconv.ParseUint(c.DefaultQuery("page_size", "20"), 10, 32)
	if querySize > 100 { querySize = 100 }
	user := service.AllService.UserService.CurUser(c)
	result := service.AllService.GroupService.OwnedGroups(user.Id, uint(queryPage), uint(querySize))
	response.Success(c, result)
}

func (m *MyGroup) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 { response.Fail(c, 101, "群组 ID 无效"); return }
	user := service.AllService.UserService.CurUser(c)
	group, err := service.AllService.GroupService.OwnedGroup(user.Id, uint(id))
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, group)
}

func (m *MyGroup) Create(c *gin.Context) {
	var input personalGroupForm
	if err := c.ShouldBindJSON(&input); err != nil { response.Fail(c, 101, err.Error()); return }
	user := service.AllService.UserService.CurUser(c)
	group, err := service.AllService.GroupService.CreateOwnedGroup(user.Id, input.Name, input.Type)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, group)
}

func (m *MyGroup) Update(c *gin.Context) {
	var input personalGroupForm
	if err := c.ShouldBindJSON(&input); err != nil { response.Fail(c, 101, err.Error()); return }
	user := service.AllService.UserService.CurUser(c)
	if err := service.AllService.GroupService.UpdateOwnedGroup(user.Id, input.Id, input.Name, input.Type); err != nil {
		response.Fail(c, 101, err.Error()); return
	}
	response.Success(c, nil)
}

func (m *MyGroup) Delete(c *gin.Context) {
	var input struct {Id uint `json:"id" binding:"required"`}
	if err := c.ShouldBindJSON(&input); err != nil { response.Fail(c, 101, err.Error()); return }
	user := service.AllService.UserService.CurUser(c)
	if err := service.AllService.GroupService.DeleteOwnedGroup(user.Id, input.Id); err != nil {
		response.Fail(c, 101, err.Error()); return
	}
	response.Success(c, nil)
}

func (m *MyGroup) Members(c *gin.Context) {
	id, ok := myGroupId(c)
	if !ok { return }
	user := service.AllService.UserService.CurUser(c)
	ids, err := service.AllService.GroupService.OwnedGroupMembers(user.Id, id)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	users, err := publicUsers(ids)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"group_id": id, "user_ids": ids, "users": users})
}

func (m *MyGroup) BatchMembers(c *gin.Context) {
	var input groupMembersRequest
	if err := c.ShouldBindJSON(&input); err != nil { response.Fail(c, 101, err.Error()); return }
	if len(input.UserIds) > 10000 { response.Fail(c, 101, "单次最多管理 10000 名成员"); return }
	user := service.AllService.UserService.CurUser(c)
	err := service.AllService.GroupService.ChangeOwnedMembers(user.Id, input.GroupId, input.UserIds, input.Action)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, nil)
}

func (m *MyGroup) Anchors(c *gin.Context) {
	id, ok := myGroupId(c)
	if !ok { return }
	user := service.AllService.UserService.CurUser(c)
	ids, err := service.AllService.GroupService.OwnedGroupAnchors(user.Id, id)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	users, err := publicUsers(ids)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, gin.H{"group_id": id, "user_ids": ids, "users": users})
}

func (m *MyGroup) SetAnchors(c *gin.Context) {
	var input groupMembersRequest
	if err := c.ShouldBindJSON(&input); err != nil { response.Fail(c, 101, err.Error()); return }
	if len(input.UserIds) > 10000 { response.Fail(c, 101, "单次最多设置 10000 名公共成员"); return }
	user := service.AllService.UserService.CurUser(c)
	if err := service.AllService.GroupService.ChangeOwnedAnchors(user.Id, input.GroupId, input.UserIds); err != nil {
		response.Fail(c, 101, err.Error()); return
	}
	response.Success(c, nil)
}

// UserLookup requires an exact username. Personal group managers do not receive
// an unrestricted system user directory.
func (m *MyGroup) UserLookup(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	if len(username) < 2 || len(username) > 32 {
		response.Fail(c, 101, "请输入完整的用户名")
		return
	}
	user := service.AllService.UserService.InfoByUsername(username)
	if user == nil || user.Id == 0 || !service.AllService.UserService.CheckUserEnable(user) {
		response.Fail(c, 101, "没有找到该用户")
		return
	}
	response.Success(c, gin.H{"id": user.Id, "username": user.Username, "nickname": user.Nickname})
}
