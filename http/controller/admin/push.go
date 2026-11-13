package admin

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type PushChannels struct{}

func currentPushUser(c *gin.Context) uint {
	user := service.AllService.UserService.CurUser(c)
	if user == nil { return 0 }
	return user.Id
}

func (h *PushChannels) Catalog(c *gin.Context) {
	response.Success(c, service.PushEventCatalog())
}

func (h *PushChannels) List(c *gin.Context) {
	items, err := service.ListPushChannels(currentPushUser(c))
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, gin.H{"list": items})
}

func (h *PushChannels) Save(c *gin.Context) {
	var input service.PushChannelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, "请填写推送渠道配置"); return
	}
	item, err := service.SavePushChannel(currentPushUser(c), input)
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, item)
}

func (h *PushChannels) Delete(c *gin.Context) {
	var body struct { Id uint `json:"id" binding:"required"` }
	if c.ShouldBindJSON(&body) != nil { response.Error(c, "请选择推送渠道"); return }
	if err := service.DeletePushChannel(currentPushUser(c), body.Id); err != nil {
		response.Error(c, err.Error()); return
	}
	response.Success(c, nil)
}

func (h *PushChannels) Test(c *gin.Context) {
	var body struct { Id uint `json:"id" binding:"required"` }
	if c.ShouldBindJSON(&body) != nil { response.Error(c, "请选择推送渠道"); return }
	if err := service.QueuePushTest(currentPushUser(c), body.Id); err != nil {
		response.Error(c, err.Error()); return
	}
	response.Success(c, gin.H{"status": "queued"})
}

func (h *PushChannels) History(c *gin.Context) {
	page, size, err := auditPage(c)
	if err != nil { response.Error(c, "无效的页码"); return }
	rows, total, err := service.ListPushDeliveries(currentPushUser(c), page, size)
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, gin.H{"list": rows, "total": total, "page": page, "page_size": size})
}

func (h *PushChannels) Retry(c *gin.Context) {
	var body struct { Id uint `json:"id" binding:"required"` }
	if c.ShouldBindJSON(&body) != nil { response.Error(c, "请选择发送记录"); return }
	if err := service.RetryPushDelivery(currentPushUser(c), body.Id); err != nil {
		response.Error(c, err.Error()); return
	}
	response.Success(c, nil)
}

// GeoIPLookup resolves both IP locations and device presence in one bounded
// authenticated request, avoiding per-device queries from the console.
func (h *PushChannels) GeoIPLookup(c *gin.Context) {
	var input struct {
		IPs       []string `json:"ips"`
		DeviceIDs []string `json:"device_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil ||
		len(input.IPs) > 256 || len(input.DeviceIDs) > 256 ||
		(len(input.IPs) == 0 && len(input.DeviceIDs) == 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error":"每次最多查询 256 个 IP 地址和 256 台设备"})
		return
	}
	user := service.AllService.UserService.CurUser(c)
	if user == nil || user.Id == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error":"请先登录"})
		return
	}
	deviceStatus, deviceNames, err := service.DevicePresenceDetailsForUser(
		user, input.DeviceIDs, service.AllService.UserService.IsAdmin(user),
	)
	if err != nil {
		response.Error(c, "无法读取设备在线状态")
		return
	}
	locations := service.LookupGeoIPBatch(input.IPs)
	c.Header("Cache-Control","no-store")
	response.Success(c,gin.H{
		"locations":locations, "device_status":deviceStatus, "device_names":deviceNames,
		"database_ready":service.GeoIPReady(),
	})
}
