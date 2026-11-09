package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// AdminPrivilege ...
func AdminPrivilege() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)

		if service.AllService.UserService.IsAdmin(u) {
			c.Next()
			return
		}
		permission := service.AdminPermissionForRoute(c.Request.Method,c.FullPath())
		if permission=="" {
			response.Fail(c,403,"当前账户没有此管理权限")
			c.Abort()
			return
		}
		scope,err:=service.RoleAccessForUser(u,permission)
		if err!=nil || (!scope.Global && len(scope.UserIDs)==0) {
			response.Fail(c,403,"当前账户没有此管理权限")
			c.Abort()
			return
		}
		c.Set("adminRoleScope",scope)
		c.Next()
	}
}
