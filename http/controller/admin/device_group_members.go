package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

func (ct *DeviceGroup) Devices(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Query("group_id"), 10, 32)
	if err != nil || groupID == 0 {
		response.Fail(c, 101, "请选择有效的设备分组")
		return
	}
	// Device-group visibility alone does not disclose device identities.
	user := service.AllService.UserService.CurUser(c)
	scope, err := service.RoleAccessForUser(user, "devices.view")
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	if !scope.Global {
		response.Fail(c, 403, "当前账户无权查看设备列表")
		return
	}
	page, size, err := auditPage(c)
	if err != nil {
		response.Fail(c, 101, "无效的页码")
		return
	}
	items, err := service.AllService.GroupService.DeviceGroupDevices(uint(groupID), page, size, c.Query("keyword"))
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, items)
}

func (ct *DeviceGroup) BatchDevices(c *gin.Context) {
	var input struct {
		GroupID    uint   `json:"group_id" binding:"required"`
		PeerRowIDs []uint `json:"peer_row_ids" binding:"required"`
		Action     string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Fail(c, 101, "请填写设备分组及设备记录编号")
		return
	}
	if err := service.AllService.GroupService.ChangeDeviceGroupMembers(input.GroupID, input.PeerRowIDs, input.Action); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, gin.H{"updated": len(input.PeerRowIDs)})
}
