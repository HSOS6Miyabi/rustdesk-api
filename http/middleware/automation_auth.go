package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// AutomationTokenAuth authenticates the separate management API using an
// explicitly scoped API token. Management endpoints require a full administrator.
func AutomationTokenAuth(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		const bearerPrefix = "Bearer "
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, bearerPrefix+"rdapi_") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API token required"})
			return
		}
		token := strings.TrimPrefix(header, bearerPrefix)
		user, err := service.AuthenticateAPIToken(token, permission)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or insufficient API token"})
			return
		}
		if service.AllService == nil || service.AllService.UserService == nil ||
			!service.AllService.UserService.IsAdmin(user) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "administrator account required"})
			return
		}
		c.Set("curUser", user)
		c.Next()
	}
}
