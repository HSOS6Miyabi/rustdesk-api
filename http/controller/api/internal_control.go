package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type InternalControl struct{}

// Permissions provides the requested device's control role to hbbs.
func (h *InternalControl) Permissions(c *gin.Context) {
	expected:=global.Config.App.InternalKey
	supplied:=c.GetHeader("X-RustDesk-Internal-Key")
	if expected==""||supplied==""||
		subtle.ConstantTimeCompare([]byte(expected),[]byte(supplied))!=1{
		c.JSON(http.StatusForbidden,gin.H{"error":"access denied"})
		return
	}
	id:=strings.TrimSpace(c.Query("device_id"))
	if id==""||len(id)>253{
		c.JSON(http.StatusBadRequest,gin.H{"error":"invalid device ID"})
		return
	}
	bits,assigned,source,err:=service.EffectiveControlPermissions(id)
	if err!=nil{
		c.JSON(http.StatusServiceUnavailable,gin.H{"error":"cannot load control role"})
		return
	}
	c.Header("Cache-Control","no-store")
	c.JSON(http.StatusOK,gin.H{
		"assigned":assigned,
		"permissions":bits,
		"source":source,
	})
}
