package admin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

// Transfer updates the owning account and the matching deployment record.
// A scoped administrator must have access to both the original and new owner.
func (ct *Peer) Transfer(c *gin.Context) {
	var form struct {
		RowID uint `json:"row_id" binding:"required"`
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil || form.RowID == 0 || form.UserID == 0 {
		response.Fail(c, 101, "请填写设备编号和目标账户编号")
		return
	}
	denied := errors.New("当前账户无权转移此设备")
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		var current model.Peer
		if err := tx.Where("row_id = ?", form.RowID).First(&current).Error; err != nil {
			return err
		}
		var target model.User
		if err := tx.Where("id = ? AND status = ?", form.UserID, model.COMMON_STATUS_ENABLE).
			First(&target).Error; err != nil {
			return err
		}
		if !scopedManagement(c, current.UserId) || !scopedManagement(c, target.Id) {
			return denied
		}
		if current.UserId == target.Id {
			return nil
		}
		if err := tx.Model(&model.Peer{}).Where("row_id = ?", current.RowId).
			Update("user_id", target.Id).Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable(&model.DeviceDeployment{}) {
			if err := tx.Model(&model.DeviceDeployment{}).Where("device_id = ?", current.Id).
				Update("owner_user_id", target.Id).Error; err != nil {
				return err
			}
		}
		if current.Uuid != "" && tx.Migrator().HasTable(&model.UserToken{}) {
			if err := tx.Where("device_uuid = ?", current.Uuid).
				Delete(&model.UserToken{}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, denied) {
			response.Fail(c, 403, denied.Error())
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Fail(c, 101, "设备或目标账户不存在")
		} else {
			response.Fail(c, 101, err.Error())
		}
		return
	}
	response.Success(c, nil)
}
