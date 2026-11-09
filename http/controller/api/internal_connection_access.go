package api

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type InternalConnectionAccess struct{}

func (h *InternalConnectionAccess) Authorize(c *gin.Context) {
	expected:=global.Config.App.InternalKey
	key:=c.GetHeader("X-RustDesk-Internal-Key")
	if expected==""||key==""||
		subtle.ConstantTimeCompare([]byte(expected),[]byte(key))!=1{
		c.JSON(http.StatusForbidden,gin.H{"error":"access denied"})
		return
	}
	var data struct{
		DeviceID string `json:"device_id"`
		Token string `json:"token"`
	}
	if err:=c.ShouldBindJSON(&data);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":"invalid connection request"})
		return
	}
	result,err:=service.AuthorizeConnection(data.Token,data.DeviceID)
	if err!=nil{
		c.JSON(http.StatusServiceUnavailable,gin.H{"error":"cannot authorize connection"})
		return
	}
	c.Header("Cache-Control","no-store")
	c.JSON(http.StatusOK,result)
}
