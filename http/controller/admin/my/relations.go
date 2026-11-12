package my

import (
    "strconv"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

type visibleGroup struct {
    Id uint `json:"id"`
    Name string `json:"name"`
    Type int `json:"type"`
}

func (h *Directory) UserRelations(c *gin.Context) {
    current := service.AllService.UserService.CurUser(c)
    if current == nil || current.Id == 0 {
        response.Fail(c, 403, "请先登录")
        return
    }
    value := strings.TrimSpace(c.Query("id"))
    if value == "" || len(value) > 100 {
        response.Fail(c, 101, "用户编号无效")
        return
    }
    query := service.DB.Model(&model.User{}).Where("status = ?", model.COMMON_STATUS_ENABLE)
    if parsed, err := strconv.ParseUint(value, 10, 32); err == nil && parsed > 0 {
        query = query.Where("id = ?", parsed)
    } else {
        query = query.Where("username = ?", value)
    }
    var target publicUser
    if err := query.Select("id, username, nickname").Take(&target).Error; err != nil || target.Id == 0 {
        response.Fail(c, 404, "没有找到可以查看的用户")
        return
    }

    groups := make([]visibleGroup, 0)
    if target.Id == current.Id {
        ids := make([]uint, 0)
        if err := service.DB.Model(&model.GroupMembership{}).
            Where("user_id = ?", current.Id).Pluck("group_id", &ids).Error; err != nil {
            response.Error(c, err.Error())
            return
        }
        if current.GroupId != 0 {
            ids = append(ids, current.GroupId)
        }
        if len(ids) > 0 {
            if err := service.DB.Model(&model.Group{}).
                Where("id IN ?", ids).Select("id, name, type").
                Order("id ASC").Scan(&groups).Error; err != nil {
                response.Error(c, err.Error())
                return
            }
        }
    }

    queryDevices, err := directoryDeviceQuery(current)
    if err != nil {
        response.Fail(c, 101, "查询设备失败")
        return
    }
    queryDevices = queryDevices.Where("p.user_id = ?", target.Id)
    var total int64
    if err := queryDevices.Count(&total).Error; err != nil {
        response.Error(c, err.Error())
        return
    }
    devices := make([]publicDevice, 0)
    if err := queryDevices.Select("p.id, p.alias, p.hostname, p.os, p.user_id").
        Order("p.row_id DESC").Limit(30).Scan(&devices).Error; err != nil {
        response.Error(c, err.Error())
        return
    }
    response.Success(c, gin.H{"groups": groups, "devices": devices, "device_total": total})
}
