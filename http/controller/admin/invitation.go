package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Invitation struct{}

func(h *Invitation) List(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListUserInvitations(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func(h *Invitation) Create(c *gin.Context){
	var form struct{
		Email string `json:"email" binding:"required"`
		GroupId uint `json:"group_id"`
		ExpiresInDays int `json:"expires_in_days"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请填写邀请邮箱");return}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Error(c,"请先登录");return}
	if form.ExpiresInDays==0{form.ExpiresInDays=7}
	if err:=service.InviteUser(form.Email,form.GroupId,user.Id,form.ExpiresInDays);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}

func(h *Invitation) Revoke(c *gin.Context){
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请选择有效的邀请");return}
	if err:=service.RevokeUserInvitation(form.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func(h *Invitation) Accept(c *gin.Context){
	var form struct{
		Token string `json:"token" binding:"required"`
		Username string `json:"username" binding:"required"`
		Nickname string `json:"nickname"`
		Password string `json:"password" binding:"required"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请填写账户信息");return}
	if err:=service.AcceptUserInvitation(form.Token,form.Username,form.Nickname,form.Password);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}
