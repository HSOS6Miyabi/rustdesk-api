package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type PasswordRecovery struct{}

func(h *PasswordRecovery) Request(c *gin.Context){
	var form struct{Email string `json:"email" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Error(c,"请填写账户邮箱");return
	}
	// Preserve an identical response for known and unknown accounts.
	_ = service.RequestPasswordReset(form.Email)
	response.Success(c,gin.H{"message":"如果该邮箱属于已启用的账户，将收到密码恢复邮件。"})
}

func(h *PasswordRecovery) Complete(c *gin.Context){
	var form struct{
		Token string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Error(c,"请填写恢复凭据和新密码");return
	}
	if err:=service.CompletePasswordReset(form.Token,form.Password);err!=nil{
		response.Error(c,"恢复链接已失效或新密码无效");return
	}
	response.Success(c,nil)
}
