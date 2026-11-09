package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// DeviceCommandAuth accepts either a normal RustDesk login session or a
// restricted management token, without exposing it to other client API routes.
func DeviceCommandAuth(permission string) gin.HandlerFunc {
	regular:=RustAuth()
	return func(c *gin.Context){
		header:=c.GetHeader("Authorization")
		if !strings.HasPrefix(header,"Bearer "+ "rdapi_") {
			regular(c)
			return
		}
		token:=strings.TrimPrefix(header,"Bearer ")
		user,err:=service.AuthenticateAPIToken(token,permission)
		if err!=nil{
			c.JSON(http.StatusUnauthorized,gin.H{"error":"Unauthorized"})
			c.Abort()
			return
		}
		c.Set("curUser",user)
		c.Set("token",token)
		c.Next()
	}
}
