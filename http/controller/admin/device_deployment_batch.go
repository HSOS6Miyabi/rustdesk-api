package admin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

// SetBatchStatus changes the deployment permissions of multiple devices in
// one transaction, without altering partially matching batches.
func (h *DeviceDeployment) SetBatchStatus(c *gin.Context) {
	var request struct {
		IDs []uint `json:"ids" binding:"required"`
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, "请填写部署记录编号和目标状态")
		return
	}
	if request.Status != "approved" && request.Status != "revoked" {
		response.Error(c, "设备部署状态无效")
		return
	}
	if len(request.IDs) == 0 || len(request.IDs) > 200 {
		response.Error(c, "每次最多修改 200 条设备部署记录")
		return
	}
	seen := make(map[uint]struct{}, len(request.IDs))
	ids := make([]uint, 0, len(request.IDs))
	for _, id := range request.IDs {
		if id == 0 { response.Error(c, "设备部署记录编号必须大于零"); return }
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.DeviceDeployment{}).Where("id IN ?", ids).
			Count(&count).Error; err != nil { return err }
		if count != int64(len(ids)) {
			return errors.New("some deployment records do not exist")
		}
		return tx.Model(&model.DeviceDeployment{}).Where("id IN ?", ids).
			Update("status", request.Status).Error
	})
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, gin.H{"processed": len(ids)})
}
