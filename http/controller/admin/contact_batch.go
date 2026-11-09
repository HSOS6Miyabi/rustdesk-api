package admin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Batch shares the current account's devices with multiple selected accounts,
// or withdraws several of its outgoing contact grants in one transaction.
func (ct *Contact) Batch(c *gin.Context) {
	var request struct {
		UserIDs []uint `json:"user_ids" binding:"required"`
		Action string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, "请选择联系人及操作类型")
		return
	}
	if request.Action != "add" && request.Action != "remove" {
		response.Error(c, "无效的联系人操作")
		return
	}
	if len(request.UserIDs) == 0 || len(request.UserIDs) > 100 {
		response.Error(c, "每次最多处理 100 位联系人")
		return
	}
	current := service.AllService.UserService.CurUser(c)
	if current == nil || current.Id == 0 {
		response.Error(c, "请先登录")
		return
	}
	seen := make(map[uint]struct{}, len(request.UserIDs))
	ids := make([]uint, 0, len(request.UserIDs))
	for _, id := range request.UserIDs {
		if id == 0 || id == current.Id {
			response.Error(c, "不能将自己或无效账户添加为联系人")
			return
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		if request.Action == "add" {
			var count int64
			if err := tx.Model(&model.User{}).
				Where("id IN ? AND status = ?", ids, model.COMMON_STATUS_ENABLE).
				Count(&count).Error; err != nil { return err }
			if count != int64(len(ids)) {
				return errors.New("some accounts do not exist or have been disabled")
			}
			for _, id := range ids {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
					Create(&model.Contact{
						OwnerUserId: current.Id, RecipientUserId: id,
					}).Error; err != nil { return err }
			}
		} else {
			if err := tx.Where("owner_user_id = ? AND recipient_user_id IN ?",
				current.Id, ids).Delete(&model.Contact{}).Error; err != nil { return err }
		}
		return nil
	})
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, gin.H{"processed": len(ids)})
}
