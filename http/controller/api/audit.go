package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	request "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Audit struct {
}

// AuditConn
// @Tags 审计
// @Summary 审计连接
// @Description 审计连接
// @Accept  json
// @Produce  json
// @Param body body request.AuditConnForm true "审计连接"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/conn [post]
func (a *Audit) AuditConn(c *gin.Context) {
	form := &request.AuditConnForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if err := service.AllService.AuditService.SaveAuditConnection(form.ToAuditConn()); err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, "")
}

// AuditFile
// @Tags 审计
// @Summary 审计文件
// @Description 审计文件
// @Accept  json
// @Produce  json
// @Param body body request.AuditFileForm true "审计文件"
// @Success 200 {string} string ""
// @Failure 500 {object} response.Response
// @Router /audit/file [post]
func (a *Audit) AuditFile(c *gin.Context) {
	form := &request.AuditFileForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	record := form.ToAuditFile()
	if err := service.AllService.AuditService.SaveAuditFile(record); err != nil {
		response.Error(c, err.Error())
		return
	}
	service.NotifyEvent("audit.file",fmt.Sprintf("audit-file:%d:%s:%s",record.Id,record.PeerId,record.FromPeer),fmt.Sprintf("设备 %s 记录了来自设备 %s 的文件传输",record.PeerId,record.FromPeer))
	response.Success(c, "")
}
