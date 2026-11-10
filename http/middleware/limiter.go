package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"net/http"
)

func Limiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		loginLimiter := global.LoginLimiter
		if loginLimiter == nil || !loginLimiter.BanEnabled() {
			c.Next()
			return
		}
		if loginLimiter.IsIPBanned(c.ClientIP()) {
			response.Fail(c, http.StatusLocked, response.TranslateMsg(c, "Banned"))
			c.Abort()
			return
		}
		c.Next()
	}
}
