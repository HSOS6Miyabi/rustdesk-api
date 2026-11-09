package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type DeviceSessions struct{}

func (h *DeviceSessions) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	rows,total,err:=service.ListActiveDeviceSessions(page,size)
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *DeviceSessions) Disconnect(c *gin.Context) {
	var form struct {
		DeviceId string `json:"device_id" binding:"required"`
		ConnId int32 `json:"conn_id" binding:"required"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Fail(c,101,"请填写设备 ID 和连接编号");return
	}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Fail(c,403,"请先登录");return}
	if err:=service.RequestDeviceDisconnect(form.DeviceId,form.ConnId,user.Id);err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,gin.H{"status":"pending"})
}

func (h *DeviceSessions) Requests(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	rows,total,err:=service.ListDisconnectRequests(page,size)
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}
