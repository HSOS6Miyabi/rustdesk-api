package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type AdminRoles struct{}

func (h *AdminRoles) Catalog(c *gin.Context){
	response.Success(c,gin.H{"permissions":service.AdminPermissionCatalog()})
}

func (h *AdminRoles) List(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListAdminRoles(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *AdminRoles) Create(c *gin.Context){
	var role model.AdminRole
	if err:=c.ShouldBindJSON(&role);err!=nil{response.Error(c,"请填写管理员角色");return}
	role.Id=0
	if err:=service.SaveAdminRole(&role);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&role)
}

func (h *AdminRoles) Update(c *gin.Context){
	var role model.AdminRole
	if err:=c.ShouldBindJSON(&role);err!=nil||role.Id==0{
		response.Error(c,"请选择管理员角色");return
	}
	if err:=service.SaveAdminRole(&role);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&role)
}

func (h *AdminRoles) Delete(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请选择管理员角色");return}
	if err:=service.DeleteAdminRole(form.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

type AdminRoleAssignments struct{}

func (h *AdminRoleAssignments) List(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListAdminRoleAssignments(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *AdminRoleAssignments) Create(c *gin.Context){
	var row model.AdminRoleAssignment
	if err:=c.ShouldBindJSON(&row);err!=nil{response.Error(c,"请填写账户与管理员角色");return}
	row.Id=0
	if err:=service.SaveAdminRoleAssignment(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *AdminRoleAssignments) Update(c *gin.Context){
	var row model.AdminRoleAssignment
	if err:=c.ShouldBindJSON(&row);err!=nil||row.Id==0{
		response.Error(c,"请选择角色分配记录");return
	}
	if err:=service.SaveAdminRoleAssignment(&row);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&row)
}

func (h *AdminRoleAssignments) Delete(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请选择角色分配记录");return}
	if err:=service.DeleteAdminRoleAssignment(form.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}
