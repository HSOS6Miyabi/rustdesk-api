package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Notifications struct{}

func (n *Notifications) Settings(c *gin.Context) {
	settings, err := service.CurrentNotificationSettings()
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c,gin.H{"settings":settings,"catalog":service.NotificationEventCatalog()})
}

func (n *Notifications) Save(c *gin.Context) {
	var input model.NotificationSetting
	if err:=c.ShouldBindJSON(&input);err!=nil { response.Error(c,"请填写通知设置"); return }
	if err:=service.SaveNotificationSettings(&input);err!=nil {response.Error(c,err.Error());return}
	response.Success(c,&input)
}

func (n *Notifications) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListNotificationDeliveries(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (n *Notifications) Retry(c *gin.Context) {
	var input struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&input);err!=nil{response.Error(c,"请选择需要重新发送的邮件");return}
	if err:=service.RetryNotification(input.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}
