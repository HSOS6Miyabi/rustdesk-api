package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type EmailVerification struct{}

func (h *EmailVerification) Status(c *gin.Context) {
	userID:=currentTOTPUser(c)
	if userID==0{response.Error(c,"请先登录");return}
	status,err:=service.GetEmailVerificationStatus(userID)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,status)
}

func (h *EmailVerification) Setup(c *gin.Context) {
	userID:=currentTOTPUser(c)
	if userID==0{response.Error(c,"请先登录");return}
	secret,err:=service.BeginEmailVerification(userID)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"secret":secret,"expires_in_seconds":600})
}

func (h *EmailVerification) Enable(c *gin.Context) {
	var form struct {
		Secret string `json:"secret" binding:"required"`
		Code string `json:"code" binding:"required"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Error(c,"请填写邮箱验证码");return
	}
	userID:=currentTOTPUser(c)
	if userID==0{response.Error(c,"请先登录");return}
	if err:=service.ConfirmEmailVerification(userID,form.Secret,form.Code);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}

func (h *EmailVerification) Disable(c *gin.Context) {
	userID:=currentTOTPUser(c)
	if userID==0{response.Error(c,"请先登录");return}
	if err:=service.DisableEmailVerification(userID);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}
