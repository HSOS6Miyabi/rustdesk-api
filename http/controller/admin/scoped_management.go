package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// scopedManagement permits access only to accounts selected by the current
// administrator role. Full administrators have no additional account restriction.
func scopedManagement(c *gin.Context, userID uint) bool {
	value, exists := c.Get("adminRoleScope")
	if !exists {
		return true
	}
	scope, valid := value.(service.AdminRoleAccess)
	return valid && service.ScopeCanViewUser(scope, userID)
}

func hasScopedManagementRole(c *gin.Context) bool {
	value, exists := c.Get("adminRoleScope")
	if !exists {
		return false
	}
	scope, valid := value.(service.AdminRoleAccess)
	return !valid || !scope.Global
}
