package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type DevicePolicy struct{}

func (d *DevicePolicy) List(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	tx:=service.DB.Model(&model.DevicePolicy{})
	var total int64
	if err:=tx.Count(&total).Error;err!=nil{response.Fail(c,101,err.Error());return}
	var list []model.DevicePolicy
	if err:=tx.Order("id DESC").Scopes(service.Paginate(page,size)).Find(&list).Error;err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,gin.H{"list":list,"total":total,"page":page,"page_size":size})
}

func (d *DevicePolicy) Create(c *gin.Context){
	var form model.DevicePolicy
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	form.Id=0
	if err:=service.SaveDevicePolicy(&form);err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,&form)
}

func (d *DevicePolicy) Update(c *gin.Context){
	var form model.DevicePolicy
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	if form.Id==0{response.Fail(c,101,"无效的安全策略编号");return}
	if err:=service.SaveDevicePolicy(&form);err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,&form)
}

func (d *DevicePolicy) Delete(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	if err:=service.DeleteDevicePolicy(form.Id);err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,nil)
}

func (d *DevicePolicy) AssignmentList(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	var total int64
	tx:=service.DB.Model(&model.DevicePolicyAssignment{})
	if err:=tx.Count(&total).Error;err!=nil{response.Fail(c,101,err.Error());return}
	var list []model.DevicePolicyAssignment
	if err:=tx.Order("id DESC").Scopes(service.Paginate(page,size)).Find(&list).Error;err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,gin.H{"list":list,"total":total,"page":page,"page_size":size})
}

func (d *DevicePolicy) AssignmentCreate(c *gin.Context){
	var form model.DevicePolicyAssignment
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	form.Id=0
	if err:=service.SaveDevicePolicyAssignment(&form);err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,&form)
}

func (d *DevicePolicy) AssignmentUpdate(c *gin.Context){
	var form model.DevicePolicyAssignment
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	if form.Id==0{response.Fail(c,101,"无效的策略分配编号");return}
	if err:=service.SaveDevicePolicyAssignment(&form);err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,&form)
}

func (d *DevicePolicy) AssignmentDelete(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Fail(c,101,err.Error());return}
	if err:=service.DeleteDevicePolicyAssignment(form.Id);err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,nil)
}

func (d *DevicePolicy) Current(c *gin.Context){
	deviceId:=c.Query("device_id")
	if len(deviceId)==0||len(deviceId)>128{
		response.Error(c,"设备 ID 无效");return
	}
	user:=service.AllService.UserService.CurUser(c)
	options,assigned,err:=service.EffectiveDevicePolicy(deviceId,user.Id,
		service.AllService.UserService.IsAdmin(user))
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{
		"assigned":assigned,"device_id":deviceId,"options":options,"schema_version":1,
	})
}
