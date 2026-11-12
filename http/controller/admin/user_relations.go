package admin

import (
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

func (ct *User) Relations(c *gin.Context) {
    value, err := strconv.ParseUint(c.Param("id"), 10, 32)
    if err != nil || value == 0 {
        response.Fail(c, 101, "用户编号无效")
        return
    }
    id := uint(value)
    user := service.AllService.UserService.InfoById(id)
    if user.Id == 0 {
        response.Fail(c, 101, "用户不存在")
        return
    }
    if scopeValue, ok := c.Get("adminRoleScope"); ok {
        scope, valid := scopeValue.(service.AdminRoleAccess)
        if !valid || !service.ScopeCanViewUser(scope, id) {
            response.Fail(c, 403, "当前账户无权查看该用户")
            return
        }
    }

    var membershipIDs []uint
    if err := service.DB.Model(&model.GroupMembership{}).
        Where("user_id = ?", id).Pluck("group_id", &membershipIDs).Error; err != nil {
        response.Error(c, err.Error())
        return
    }
    if user.GroupId > 0 {
        membershipIDs = append(membershipIDs, user.GroupId)
    }
    groups := make([]model.Group, 0)
    if len(membershipIDs) > 0 {
        if err := service.DB.Where("id IN ?", membershipIDs).
            Order("id ASC").Find(&groups).Error; err != nil {
            response.Error(c, err.Error())
            return
        }
    }

    var total int64
    if err := service.DB.Model(&model.Peer{}).
        Where("user_id = ?", id).Count(&total).Error; err != nil {
        response.Error(c, err.Error())
        return
    }
    devices := make([]model.Peer, 0)
    if err := service.DB.Where("user_id = ?", id).
        Order("row_id DESC").Limit(30).Find(&devices).Error; err != nil {
        response.Error(c, err.Error())
        return
    }
    response.Success(c, gin.H{"groups": groups, "devices": devices, "device_total": total})
}
