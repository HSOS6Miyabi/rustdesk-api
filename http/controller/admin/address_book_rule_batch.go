package admin

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

// BatchPersonalShares grants or revokes access to a personal address book for
// multiple accounts. Every change is applied in one database transaction.
func (h *AddressBookCollectionRule) BatchPersonalShares(c *gin.Context) {
	var request struct {
		CollectionID uint `json:"collection_id" binding:"required"`
		TargetUserIDs []uint `json:"target_user_ids" binding:"required"`
		Action string `json:"action" binding:"required"`
		Rule int `json:"rule"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, "请填写地址簿、目标账户和操作类型")
		return
	}
	if request.Action != "share" && request.Action != "revoke" {
		response.Error(c, "地址簿共享操作无效")
		return
	}
	if request.Action == "share" &&
		(request.Rule < model.ShareAddressBookRuleRuleRead ||
			request.Rule > model.ShareAddressBookRuleRuleFullControl) {
		response.Error(c, "地址簿共享权限无效")
		return
	}
	if len(request.TargetUserIDs) == 0 || len(request.TargetUserIDs) > 100 {
		response.Error(c, "每次最多处理 100 个账户")
		return
	}
	current := service.AllService.UserService.CurUser(c)
	if current == nil || current.Id == 0 {
		response.Error(c, "请先登录")
		return
	}
	seen := make(map[uint]struct{}, len(request.TargetUserIDs))
	ids := make([]uint, 0, len(request.TargetUserIDs))
	for _, id := range request.TargetUserIDs {
		if id == 0 || id == current.Id {
			response.Error(c, "不能将地址簿共享给自己或无效账户")
			return
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	err := service.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.AddressBookCollection{}).
			Where("id = ? AND user_id = ?", request.CollectionID, current.Id).
			Count(&count).Error; err != nil { return err }
		if count != 1 { return errors.New("address book collection does not belong to the current account") }
		if request.Action == "revoke" {
			return tx.Where("user_id = ? AND collection_id = ? AND type = ? AND to_id IN ?",
				current.Id, request.CollectionID, model.ShareAddressBookRuleTypePersonal, ids).
				Delete(&model.AddressBookCollectionRule{}).Error
		}
		if err := tx.Model(&model.User{}).
			Where("id IN ? AND status = ?", ids, model.COMMON_STATUS_ENABLE).
			Count(&count).Error; err != nil { return err }
		if count != int64(len(ids)) { return errors.New("some accounts do not exist or are disabled") }
		for _, id := range ids {
			var existing model.AddressBookCollectionRule
			query := tx.Where("user_id = ? AND collection_id = ? AND type = ? AND to_id = ?",
				current.Id, request.CollectionID, model.ShareAddressBookRuleTypePersonal, id).
				First(&existing)
			if query.Error == nil {
				if err := tx.Model(&existing).Update("rule", request.Rule).Error; err != nil {
					return err
				}
			} else if errors.Is(query.Error, gorm.ErrRecordNotFound) {
				record := &model.AddressBookCollectionRule{
					UserId: current.Id, CollectionId: request.CollectionID,
					Type: model.ShareAddressBookRuleTypePersonal, ToId: id, Rule: request.Rule,
				}
				if err := tx.Create(record).Error; err != nil { return err }
			} else {
				return query.Error
			}
		}
		return nil
	})
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, gin.H{"processed": len(ids)})
}
