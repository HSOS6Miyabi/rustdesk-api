package admin

import (
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
	result := service.DB.Model(&model.DeviceDeployment{}).
		Where("id = ?", request.Id).Update("status", request.Status)
	if result.Error != nil {
		response.Fail(c, 101, result.Error.Error())
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备部署记录不存在"})
		return
	}
	response.Success(c, nil)
}
