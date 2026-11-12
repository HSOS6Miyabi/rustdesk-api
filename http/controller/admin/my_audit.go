package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type MyAudit struct{}

func (m *MyAudit) Connections(c *gin.Context) {
	var query admin.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil { response.Fail(c, 101, err.Error()); return }
	user := service.AllService.UserService.CurUser(c)
	list, err := service.AllService.AuditService.OwnConnections(user.Id, query.Page, query.PageSize,query.SortBy,query.SortDir)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, list)
}

func (m *MyAudit) Files(c *gin.Context) {
	var query admin.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil { response.Fail(c, 101, err.Error()); return }
	user := service.AllService.UserService.CurUser(c)
	list, err := service.AllService.AuditService.OwnFiles(user.Id, query.Page, query.PageSize,query.SortBy,query.SortDir)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, list)
}
