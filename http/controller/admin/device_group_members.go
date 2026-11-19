package admin

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

func (ct *DeviceGroup) Devices(c *gin.Context)    { ct.listDevices(c, false) }
func (ct *DeviceGroup) Candidates(c *gin.Context) { ct.listDevices(c, true) }

func (ct *DeviceGroup) listDevices(c *gin.Context, candidates bool) {
	id, err := strconv.ParseUint(c.Query("group_id"), 10, 32)
	if err != nil || id == 0 {
		response.Fail(c, 101, "请选择设备分组")
		return
	}
	page, size, err := auditPage(c)
	if err != nil {
		response.Fail(c, 101, "请填写有效的页码和每页数量")
		return
	}
	rows, total, err := service.ListDeviceGroupDevices(uint(id), candidates, c.Query("keyword"), page, size)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, 101, "设备分组不存在")
		} else {
			response.Error(c, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}

func (ct *DeviceGroup) BatchDevices(c *gin.Context) {
	var form struct {
		GroupId uint                              `json:"group_id" binding:"required"`
		Action  string                            `json:"action" binding:"required"`
		Devices []service.DeviceGroupMemberChange `json:"devices" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, 101, "请选择设备和操作")
		return
	}
	affected, err := service.ChangeDeviceGroupMembers(form.GroupId, form.Devices, form.Action)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDeviceGroupChanged):
			response.Fail(c, 409, err.Error())
		case errors.Is(err, gorm.ErrRecordNotFound):
			response.Fail(c, 101, "设备分组不存在")
		default:
			response.Fail(c, 101, err.Error())
		}
		return
	}
	response.Success(c, gin.H{"affected": affected})
}
