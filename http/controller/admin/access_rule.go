package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type AccessRules struct{}

func (h *AccessRules) List(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListAccessRules(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}
func (h *AccessRules) Create(c *gin.Context){
	var rule model.AccessRule
	if err:=c.ShouldBindJSON(&rule);err!=nil{response.Error(c,"请填写连接规则");return}
	rule.Id=0
	if err:=service.SaveAccessRule(&rule);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&rule)
}
func (h *AccessRules) Update(c *gin.Context){
	var rule model.AccessRule
	if err:=c.ShouldBindJSON(&rule);err!=nil||rule.Id==0{
		response.Error(c,"请选择有效的连接规则");return
	}
	if err:=service.SaveAccessRule(&rule);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&rule)
}
func (h *AccessRules) Delete(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请选择有效的连接规则");return}
	if err:=service.DeleteAccessRule(form.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}
