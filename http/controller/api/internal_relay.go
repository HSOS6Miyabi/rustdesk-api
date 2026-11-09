package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// InternalRelay provides managed relay configuration only to hbbs.
type InternalRelay struct{}

func (h *InternalRelay) Nodes(c *gin.Context) {
	expected:=global.Config.App.InternalKey
	provided:=c.GetHeader("X-RustDesk-Internal-Key")
	if expected==""||provided==""||
		subtle.ConstantTimeCompare([]byte(expected),[]byte(provided))!=1{
		c.JSON(http.StatusForbidden,gin.H{"error":"access denied"})
		return
	}
	nodes,err:=service.EnabledRelayNodes()
	if err!=nil{
		c.JSON(http.StatusServiceUnavailable,gin.H{"error":"cannot list relay nodes"})
		return
	}
	c.Header("Cache-Control","no-store")
	c.JSON(http.StatusOK,gin.H{"nodes":nodes})
}
