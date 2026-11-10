package api

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"net/http"
	"time"
)

type Index struct {
}

// Index 首页
// @Tags 首页
// @Summary 首页
// @Description 首页
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router / [get]
func (i *Index) Index(c *gin.Context) {
	response.Success(
		c,
		"Hello Gwen",
	)
}

// Heartbeat 心跳
// @Tags 首页
// @Summary 心跳
// @Description 心跳
// @Accept  json
// @Produce  json
// @Success 200 {object} nil
// @Failure 500 {object} response.Response
// @Router /heartbeat [post]
func (i *Index) Heartbeat(c *gin.Context) {
	info := &requstform.PeerInfoInHeartbeat{}
	err := c.ShouldBindJSON(info)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if info.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	peer := service.AllService.PeerService.FindById(info.Id)
	if peer == nil || peer.RowId == 0 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	// Check the device identity and managed deployment before returning policies.
	if !heartbeatDeviceAuthorized(peer, info.Uuid) {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if time.Now().Unix()-peer.LastOnlineTime >= 30 {
		upp := &model.Peer{RowId:peer.RowId,LastOnlineTime:time.Now().Unix(),LastOnlineIp:c.ClientIP()}
		service.AllService.PeerService.Update(upp)
	}
	data,err:=service.HeartbeatDeviceStrategy(peer,info.ModifiedAt)
	if err!=nil {
		if service.Logger!=nil{service.Logger.WithError(err).Warn("failed loading device policy for heartbeat")}
		data=map[string]interface{}{}
	}
	commands,err:=service.SynchronizeDeviceSessions(peer,info.Conns)
	if err!=nil {
		if service.Logger!=nil{service.Logger.WithError(err).Warn("failed updating device sessions")}
	}else if len(commands)>0{
		data["disconnect"]=commands
	}
	service.ReportPushHeartbeat(peer,c.ClientIP())
	c.JSON(http.StatusOK,data)
}

func heartbeatDeviceAuthorized(peer *model.Peer, presentedUUID string) bool {
	if peer == nil || peer.RowId == 0 || peer.Uuid == "" || presentedUUID == "" {
		return false
	}
	identityMatches := peer.Uuid == presentedUUID
	if !identityMatches {
		stored, validStored := normalizedBase64(peer.Uuid, 8, 128)
		presented, validPresented := normalizedBase64(presentedUUID, 8, 128)
		identityMatches = validStored && validPresented && stored == presented
	}
	if !identityMatches {
		return false
	}
	if !global.Config.App.RequireDeviceDeployment {
		return true
	}
	storedUUID, valid := normalizedBase64(peer.Uuid, 8, 128)
	if !valid {
		return false
	}
	var deployment model.DeviceDeployment
	if err := service.DB.Select("uuid", "status", "owner_user_id").
		Where("device_id = ?", peer.Id).Take(&deployment).Error; err != nil {
		return false
	}
	return deployment.Status == "approved" && deployment.OwnerUserId != 0 &&
		deployment.OwnerUserId == peer.UserId && deployment.Uuid == storedUUID
}

// Version 版本
// @Tags 首页
// @Summary 版本
// @Description 版本
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /version [get]
func (i *Index) Version(c *gin.Context) {
	//读取resources/version文件
	v := service.AllService.AppService.GetAppVersion()
	response.Success(
		c,
		v,
	)
}
