package my

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"time"
	"strings"
	"unicode/utf8"
	"github.com/lejianwen/rustdesk-api/v2/model"
)

type Peer struct {
}

// List 列表
// @Tags 我的设备
// @Summary 设备列表
// @Description 设备列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Param time_ago query int false "时间"
// @Param id query string false "ID"
// @Param hostname query string false "主机名"
// @Param uuids query string false "uuids 用逗号分隔"
// @Success 200 {object} response.Response{data=model.PeerList}
// @Failure 500 {object} response.Response
// @Router /admin/my/peer/list [get]
// @Security token
func (ct *Peer) List(c *gin.Context) {
	query := &admin.PeerQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	u := service.AllService.UserService.CurUser(c)
	res := service.AllService.PeerService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		tx.Where("user_id = ?", u.Id)
		service.DirectoryPeerKeyword(tx,query.Keyword)
		if query.TimeAgo > 0 {
			lt := time.Now().Unix() - int64(query.TimeAgo)
			tx.Where("last_online_time < ?", lt)
		}
		if query.TimeAgo < 0 {
			lt := time.Now().Unix() + int64(query.TimeAgo)
			tx.Where("last_online_time > ?", lt)
		}
		if query.Id != "" {
			tx.Where("id like ?", "%"+query.Id+"%")
		}
		if query.Hostname != "" {
			tx.Where("hostname like ?", "%"+query.Hostname+"%")
		}
		if query.Uuids != "" {
			tx.Where("uuid in (?)", query.Uuids)
		}
	})
	response.Success(c, res)
}

func (ct *Peer) Update(c *gin.Context) {
	var input struct {
		RowId uint `json:"row_id"`
		Alias string `json:"alias"`
		Note string `json:"note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.RowId == 0 {
		response.Fail(c, 101, "请填写有效的设备记录编号")
		return
	}
	input.Alias = strings.TrimSpace(input.Alias)
	input.Note = strings.TrimSpace(input.Note)
	if utf8.RuneCountInString(input.Alias) > 128 || utf8.RuneCountInString(input.Note) > 4096 {
		response.Fail(c, 101, "设备备注的长度超出限制")
		return
	}
	user := service.AllService.UserService.CurUser(c)
	result := service.DB.Model(&model.Peer{}).Where("row_id = ? AND user_id = ?",input.RowId,user.Id).
		Updates(map[string]interface{}{"alias":input.Alias,"note":input.Note})
	if result.Error != nil {
		response.Error(c,result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c,403,"您没有修改该设备的权限")
		return
	}
	response.Success(c,nil)
}

func (ct *Peer) Delete(c *gin.Context) {
	var input struct { RowId uint `json:"row_id"` }
	if err := c.ShouldBindJSON(&input); err != nil || input.RowId == 0 {
		response.Fail(c,101,"请填写有效的设备记录编号")
		return
	}
	user := service.AllService.UserService.CurUser(c)
	peer := service.AllService.PeerService.InfoByRowId(input.RowId)
	if peer.RowId == 0 || peer.UserId != user.Id {
		response.Fail(c,403,"您没有删除该设备的权限")
		return
	}
	if err := service.AllService.PeerService.Delete(peer); err != nil {
		response.Error(c,err.Error())
		return
	}
	response.Success(c,nil)
}
