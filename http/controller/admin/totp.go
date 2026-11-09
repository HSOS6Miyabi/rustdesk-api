package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type TOTP struct{}

func currentTOTPUser(c *gin.Context) uint {
	user:=service.AllService.UserService.CurUser(c)
	if user==nil{return 0}
	return user.Id
}

func(a *TOTP) Status(c *gin.Context){
	owner:=currentTOTPUser(c)
	if owner==0{response.Error(c,"请先登录");return}
	status,err:=service.GetTOTPStatus(owner)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,status)
}

func(a *TOTP) Setup(c *gin.Context){
	owner:=currentTOTPUser(c)
	if owner==0{response.Error(c,"请先登录");return}
	result,err:=service.BeginTOTPSetup(owner)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,result)
}

func(a *TOTP) Enable(c *gin.Context){
	var form struct{Code string `json:"code" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请输入身份验证器中的动态验证码");return}
	owner:=currentTOTPUser(c)
	if owner==0{response.Error(c,"请先登录");return}
	codes,err:=service.ActivateTOTP(owner,form.Code)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"recovery_codes":codes})
}

func(a *TOTP) Disable(c *gin.Context){
	var form struct{Code string `json:"code" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请填写动态验证码或恢复码");return}
	owner:=currentTOTPUser(c)
	if owner==0{response.Error(c,"请先登录");return}
	if err:=service.DisableTOTP(owner,form.Code);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func(a *TOTP) RegenerateRecoveryCodes(c *gin.Context){
	var form struct{Code string `json:"code" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请填写动态验证码或恢复码");return}
	owner:=currentTOTPUser(c)
	if owner==0{response.Error(c,"请先登录");return}
	codes,err:=service.RotateTOTPRecoveryCodes(owner,form.Code)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"recovery_codes":codes})
}
