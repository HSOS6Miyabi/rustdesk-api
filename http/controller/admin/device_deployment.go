package admin

import (
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
