package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Mail struct{}

func(h *Mail) Settings(c *gin.Context){
	settings,err:=service.GetPublicMailSettings()
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,settings)
}

func(h *Mail) Save(c *gin.Context){
	var form service.MailSettingsInput
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"SMTP 配置格式无效");return}
	settings,err:=service.SaveMailSettings(form)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,settings)
}

func(h *Mail) Test(c *gin.Context){
	var form struct{To string `json:"to" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"请填写测试邮件的收件地址");return}
	if err:=service.SendMail(form.To,"RustDesk 邮件服务测试",
		"RustDesk API 邮件服务已配置完成。");err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,gin.H{"sent":true})
}
