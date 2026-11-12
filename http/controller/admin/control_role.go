package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type ControlRoles struct{}

func (h *ControlRoles) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	roles,total,err:=service.ListControlRoles(page,size,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":roles,"total":total,"page":page,"page_size":size})
}

func (h *ControlRoles) Create(c *gin.Context) {
	var row model.ControlRole
	if err:=c.ShouldBindJSON(&row);err!=nil{response.Error(c,"无效的控制角色");return}
	row.Id=0
	if err:=service.SaveControlRole(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *ControlRoles) Update(c *gin.Context) {
	var row model.ControlRole
	if err:=c.ShouldBindJSON(&row);err!=nil||row.Id==0{
		response.Error(c,"请选择要修改的控制角色");return
	}
	if err:=service.SaveControlRole(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *ControlRoles) Delete(c *gin.Context) {
	var request struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"无效的控制角色编号");return}
	if err:=service.DeleteControlRole(request.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func (h *ControlRoles) Effective(c *gin.Context) {
	deviceID:=strings.TrimSpace(c.Query("device_id"))
	bits,assigned,source,err:=service.EffectiveControlPermissions(deviceID)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{
		"device_id":deviceID,"assigned":assigned,
		"permissions":bits,"source":source,
	})
}

type ControlAssignments struct{}

func (h *ControlAssignments) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListControlRoleAssignments(page,size,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *ControlAssignments) Create(c *gin.Context) {
	var row model.ControlRoleAssignment
	if err:=c.ShouldBindJSON(&row);err!=nil{response.Error(c,"无效的角色分配");return}
	row.Id=0
	if err:=service.SaveControlRoleAssignment(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *ControlAssignments) Update(c *gin.Context) {
	var row model.ControlRoleAssignment
	if err:=c.ShouldBindJSON(&row);err!=nil||row.Id==0{
		response.Error(c,"请选择要修改的角色分配");return
	}
	if err:=service.SaveControlRoleAssignment(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *ControlAssignments) Delete(c *gin.Context) {
	var request struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"无效的分配编号");return}
	if err:=service.DeleteControlRoleAssignment(request.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}
