package admin

import (
	"fmt"
	"time"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type DeviceDeployment struct{}

func DeviceDeploymentBind(rg *gin.RouterGroup) {
	handler := &DeviceDeployment{}
	r := rg.Group("/device_deployment").Use(middleware.AdminPrivilege())
	r.GET("/list", handler.List)
	r.POST("/status", handler.SetStatus)
	r.POST("/batchStatus", handler.SetBatchStatus)
}

func (h *DeviceDeployment) List(c *gin.Context) {
	page, size, err := auditPage(c)
	if err != nil {
		response.Fail(c, 101, "无效的页码")
		return
	}
	var total int64
	tx := service.DB.Model(&model.DeviceDeployment{})
	if err := tx.Count(&total).Error; err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	var rows []model.DeviceDeployment
	if err := tx.Order("id DESC").Scopes(service.Paginate(page, size)).Find(&rows).Error; err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}

func (h *DeviceDeployment) SetStatus(c *gin.Context) {
	var request struct {
		Id     uint   `json:"id" binding:"required"`
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, 101, "请选择需要修改的设备部署记录")
		return
	}
	if request.Status != "approved" && request.Status != "revoked" {
		response.Fail(c, 101, "无效的设备部署状态")
		return
	}
	var before model.DeviceDeployment
	if err := service.DB.Where("id = ?",request.Id).Take(&before).Error;err!=nil{
		c.JSON(http.StatusNotFound,gin.H{"error":"设备部署记录不存在"})
		return
	}
	if before.Status == request.Status {
		response.Success(c,nil)
		return
	}
	result := service.DB.Model(&model.DeviceDeployment{}).
		Where("id = ? AND status = ?",request.Id,before.Status).
		Update("status",request.Status)
	if result.Error != nil {
		response.Fail(c,101,result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		response.Fail(c,101,"设备部署状态已改变，请刷新后重试")
		return
	}
	reference:=fmt.Sprintf("deployment:%d:%s:%d",request.Id,request.Status,time.Now().UnixNano())
	service.NotifyEvent("deployment.changed",reference,fmt.Sprintf("部署记录 %d 的状态调整为 %s",request.Id,request.Status))
	actor:=service.AllService.UserService.CurUser(c)
	detail:="部署状态："+request.Status
	if actor!=nil && actor.Id!=0 { detail+="；操作账户："+actor.Username }
	service.PublishPushEvent(service.PushEvent{
		Code:"deployment.changed",Reference:reference,
		DeviceID:before.DeviceId,UserID:before.OwnerUserId,Detail:detail,
	})
	response.Success(c, nil)
}
