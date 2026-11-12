package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"strconv"
	"time"
	"strings"
)

type Peer struct {
}

// Detail 设备
// @Tags 设备
// @Summary 设备详情
// @Description 设备详情
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/detail/{id} [get]
// @Security token
func (ct *Peer) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.PeerService.InfoByRowId(uint(iid))
	if u.RowId > 0 {
		if scopeAny,ok:=c.Get("adminRoleScope");ok {
			scope,valid:=scopeAny.(service.AdminRoleAccess)
			if !valid||!service.ScopeCanViewUser(scope,u.UserId){
				response.Fail(c,403,"当前账户无权查看该记录")
				return
			}
		}
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
	return
}

// Create 创建设备
// @Tags 设备
// @Summary 创建设备
// @Description 创建设备
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/create [post]
// @Security token
func (ct *Peer) Create(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	p := f.ToPeer()
	err := service.AllService.PeerService.Create(p)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// List 列表
// @Tags 设备
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
// @Router /admin/peer/list [get]
// @Security token
func (ct *Peer) List(c *gin.Context) {
	query := &admin.PeerQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if scopeAny,ok:=c.Get("adminRoleScope");ok {
		scope,valid:=scopeAny.(service.AdminRoleAccess)
		if !valid||(!scope.Global&&len(scope.UserIDs)==0){
			response.Fail(c,403,"当前账户无权查看该记录")
			return
		}
		if !scope.Global{
			records:=&model.PeerList{}
			records.Page=int64(query.Page)
			records.PageSize=int64(query.PageSize)
			tx:=service.DB.Model(&model.Peer{}).Where("user_id IN ?",scope.UserIDs)
			tx=service.DirectoryPeerKeyword(tx,query.Keyword)
			if query.Id!=""{tx=tx.Where("id LIKE ?","%"+query.Id+"%")}
			if query.Hostname!=""{tx=tx.Where("hostname LIKE ?","%"+query.Hostname+"%")}
			if query.Username!=""{tx=tx.Where("username LIKE ?","%"+query.Username+"%")}
			if query.Ip!=""{tx=tx.Where("last_online_ip LIKE ?","%"+query.Ip+"%")}
			if query.Alias!=""{tx=tx.Where("alias LIKE ?","%"+query.Alias+"%")}
			if query.TimeAgo!=0 {
				lt:=time.Now().Unix()-int64(query.TimeAgo)
				if query.TimeAgo>0{tx=tx.Where("last_online_time < ?",lt)
				}else{tx=tx.Where("last_online_time > ?",lt)}
			}
			if query.Uuids!=""{
				tx=tx.Where("uuid IN ?",strings.Split(query.Uuids,","))
			}
			if err:=tx.Count(&records.Total).Error;err!=nil{response.Error(c,err.Error());return}
			if err:=tx.Scopes(service.Paginate(query.Page,query.PageSize), service.SortScope(query.SortBy,query.SortDir)).
				Find(&records.Peers).Error;err!=nil{response.Error(c,err.Error());return}
			response.Success(c,records)
			return
		}
	}
	res := service.AllService.PeerService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
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
		if query.Username != "" {
			tx.Where("username like ?", "%"+query.Username+"%")
		}
		if query.Ip != "" {
			tx.Where("last_online_ip like ?", "%"+query.Ip+"%")
		}
		if query.Alias != "" {
			tx.Where("alias like ?", "%"+query.Alias+"%")
		}
	},query.SortBy,query.SortDir)
	response.Success(c, res)
}

// Update 编辑
// @Tags 设备
// @Summary 设备编辑
// @Description 设备编辑
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/update [post]
// @Security token
func (ct *Peer) Update(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if hasScopedManagementRole(c) {
		current := service.AllService.PeerService.InfoByRowId(f.RowId)
		if current.RowId == 0 || !scopedManagement(c, current.UserId) ||
			(f.Id != "" && f.Id != current.Id) ||
			(f.Uuid != "" && f.Uuid != current.Uuid) {
			response.Fail(c, 403, "当前账户无权修改此设备或其身份信息")
			return
		}
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToPeer()
	err := service.AllService.PeerService.Update(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete 删除
// @Tags 设备
// @Summary 设备删除
// @Description 设备删除
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/peer/delete [post]
// @Security token
func (ct *Peer) Delete(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.RowId
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.PeerService.InfoByRowId(f.RowId)
	if hasScopedManagementRole(c) && (u.RowId == 0 || !scopedManagement(c, u.UserId)) {
		response.Fail(c, 403, "当前账户无权删除此设备")
		return
	}
	if u.RowId > 0 {
		err := service.AllService.PeerService.Delete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// BatchDelete 批量删除
// @Tags 设备
// @Summary 批量设备删除
// @Description 批量设备删除
// @Accept  json
// @Produce  json
// @Param body body admin.PeerBatchDeleteForm true "设备id"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/peer/batchDelete [post]
// @Security token
func (ct *Peer) BatchDelete(c *gin.Context) {
	f := &admin.PeerBatchDeleteForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.RowIds) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	if hasScopedManagementRole(c) {
		var peers []model.Peer
		if err := service.DB.Where("row_id IN ?", f.RowIds).Find(&peers).Error; err != nil {
			response.Fail(c, 101, err.Error())
			return
		}
		unique := make(map[uint]struct{}, len(f.RowIds))
		for _, id := range f.RowIds { unique[id] = struct{}{} }
		if len(peers) != len(unique) {
			response.Fail(c, 403, "部分设备不存在或不在允许的管理范围内")
			return
		}
		for _, peer := range peers {
			if !scopedManagement(c, peer.UserId) {
				response.Fail(c, 403, "部分设备不在允许的管理范围内")
				return
			}
		}
	}
	err := service.AllService.PeerService.BatchDelete(f.RowIds)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *Peer) SimpleData(c *gin.Context) {
	f := &admin.SimpleDataQuery{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.Ids) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	res := service.AllService.PeerService.List(1, 99999, func(tx *gorm.DB) {
		//可以公开的情报
		tx.Select("id,version")
		tx.Where("id in (?)", f.Ids)
	})
	response.Success(c, res)
}
