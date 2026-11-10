package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

type Peer struct {
}

// SysInfo
// @Tags System
// @Summary 提交系统信息
// @Description 提交系统信息
// @Accept  json
// @Produce  json
// @Param body body requstform.PeerForm true "系统信息表单"
// @Success 200 {string} string "SYSINFO_UPDATED,ID_NOT_FOUND"
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo [post]
func (p *Peer) SysInfo(c *gin.Context) {
	f := &requstform.PeerForm{}
	err := c.ShouldBindBodyWith(f, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	// In managed mode, only the identity approved by --deploy may upload
	// inventory information. A revoked deployment cannot modify a peer.
	var approvedOwner uint
	if global.Config.App.RequireDeviceDeployment {
		if !nonEmptyDeviceID(f.Id) {
			c.Status(http.StatusForbidden)
			return
		}
		var deployment model.DeviceDeployment
		err := service.DB.Where("device_id = ?", f.Id).First(&deployment).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusForbidden)
			return
		}
		if err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		uuid, valid := normalizedBase64(f.Uuid, 8, 128)
		if deployment.Status != "approved" || !valid || uuid != deployment.Uuid ||
			deployment.OwnerUserId == 0 {
			c.Status(http.StatusForbidden)
			return
		}
		f.Uuid = uuid
		approvedOwner = deployment.OwnerUserId
	}
	fpe := f.ToPeer()
	pe := service.AllService.PeerService.FindById(f.Id)
	if approvedOwner != 0 && pe.RowId != 0 && pe.Uuid != "" {
		storedUUID, valid := normalizedBase64(pe.Uuid, 8, 128)
		if !valid || storedUUID != f.Uuid {
			c.Status(http.StatusForbidden)
			return
		}
	}
	if pe.RowId == 0 {
		pe = f.ToPeer()
		if approvedOwner != 0 {
			pe.UserId = approvedOwner
		} else {
			pe.UserId = service.AllService.UserService.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		}
		err = service.AllService.PeerService.Create(pe)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
		service.PublishPushEvent(service.PushEvent{
			Code:"device.registered", Reference:fmt.Sprintf("registration:%d",pe.RowId),
			PeerRowID:pe.RowId, DeviceID:pe.Id, SourceIP:c.ClientIP(), UserID:pe.UserId,
		})
	} else {
		ownerID := pe.UserId
		if ownerID == 0 {
			ownerID = service.AllService.UserService.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		}
		if approvedOwner != 0 {
			ownerID = approvedOwner
		}
		updates := make(map[string]interface{})
		for _, field := range []struct{ column, previous, next string }{
			{"cpu", pe.Cpu, fpe.Cpu},
			{"hostname", pe.Hostname, fpe.Hostname},
			{"memory", pe.Memory, fpe.Memory},
			{"os", pe.Os, fpe.Os},
			{"username", pe.Username, fpe.Username},
			{"uuid", pe.Uuid, fpe.Uuid},
			{"version", pe.Version, fpe.Version},
		} {
			// Some official clients omit optional inventory fields. Preserve
			// saved values until a non-empty replacement is supplied.
			if field.next != "" && field.previous != field.next {
				updates[field.column] = field.next
			}
		}
		if pe.UserId != ownerID {
			updates["user_id"] = ownerID
		}
		if len(updates) != 0 {
			err = service.DB.Model(&model.Peer{}).Where("row_id = ?", pe.RowId).
				Updates(updates).Error
			if err != nil {
				response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
				return
			}
		}
	}
	//SYSINFO_UPDATED 上传成功
	//ID_NOT_FOUND 下次心跳会上传
	//直接响应文本
	c.String(http.StatusOK, "SYSINFO_UPDATED")
}

// SysInfoVer
// @Tags System
// @Summary 获取系统版本信息
// @Description 获取系统版本信息
// @Accept  json
// @Produce  json
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo_ver [post]
func (p *Peer) SysInfoVer(c *gin.Context) {
	//读取resources/version文件
	v := service.AllService.AppService.GetAppVersion()
	// 加上启动时间，方便client上传信息
	v = fmt.Sprintf("%s\n%s", v, service.AllService.AppService.GetStartTime())
	c.String(http.StatusOK, v)
}
