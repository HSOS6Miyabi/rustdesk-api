package api

import (
	"net/http"
	"fmt"
	"github.com/gin-gonic/gin"
	request "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

func (a *Audit) AuditAlarm(c *gin.Context) {
	var form request.AuditAlarmForm
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Error(c,err.Error());return
	}
	event,err:=form.ToAuditAlarm()
	if err!=nil {response.Error(c,err.Error());return}
	if err:=service.AllService.AuditService.SaveAuditAlarm(event,form.Nonce,c.ClientIP());err!=nil{
		response.Error(c,err.Error());return
	}
	service.PublishPushEvent(service.PushEvent{
		Code:"audit.alarm",Reference:fmt.Sprintf("alarm:%d:%s",event.Id,form.Nonce),
		DeviceID:event.PeerId,ConnID:event.ConnId,SourceIP:event.SourceIp,
		Detail:fmt.Sprintf("告警类型：%d；信息：%s",event.Typ,event.Info),
	})
	c.JSON(http.StatusOK,gin.H{"code":0,"msg":"success"})
}
