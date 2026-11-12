package my

import (
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

type SessionViews struct{}

func sessionPage(c *gin.Context) (uint,uint) {
    page,_:=strconv.Atoi(c.DefaultQuery("page","1"))
    size,_:=strconv.Atoi(c.DefaultQuery("page_size","20"))
    if page<1 {page=1}
    if size<1 {size=20}
    if size>100 {size=100}
    return uint(page),uint(size)
}

func (h *SessionViews) Active(c *gin.Context) {
    user:=service.AllService.UserService.CurUser(c)
    page,size:=sessionPage(c)
    rows,total,err:=service.ListOwnedActiveDeviceSessions(user.Id,page,size,c.Query("sort_by"),c.Query("sort_dir"))
    if err!=nil {response.Fail(c,101,err.Error());return}
    details,err:=service.EnrichActiveConnections(rows)
    if err!=nil{response.Fail(c,101,err.Error());return}
    response.Success(c,gin.H{"list":details,"total":total,"page":page,"page_size":size})
}

func (h *SessionViews) Disconnections(c *gin.Context) {
    user:=service.AllService.UserService.CurUser(c)
    page,size:=sessionPage(c)
    rows,total,err:=service.ListOwnedDisconnectRequests(user.Id,page,size,c.Query("sort_by"),c.Query("sort_dir"))
    if err!=nil {response.Fail(c,101,err.Error());return}
    response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *SessionViews) Disconnect(c *gin.Context) {
    var form struct {
        DeviceID string `json:"device_id" binding:"required"`
        ConnID int32 `json:"conn_id" binding:"required"`
    }
    if err:=c.ShouldBindJSON(&form);err!=nil {response.Fail(c,101,"请选择需要结束的连接");return}
    user:=service.AllService.UserService.CurUser(c)
    if err:=service.RequestOwnedDeviceDisconnect(user.Id,form.DeviceID,form.ConnID);err!=nil {
        response.Fail(c,101,err.Error());return
    }
    response.Success(c,gin.H{"status":"pending"})
}

func (h *SessionViews) Logins(c *gin.Context) {
    user:=service.AllService.UserService.CurUser(c)
    page,size:=sessionPage(c)
    rows,total,err:=service.ListLoginSessions(user.Id,false,c.GetHeader("api-token"),page,size,c.Query("sort_by"),c.Query("sort_dir"))
    if err!=nil {response.Fail(c,101,err.Error());return}
    response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *SessionViews) RevokeLogin(c *gin.Context) {
    var form struct { ID uint `json:"id" binding:"required"` }
    if err:=c.ShouldBindJSON(&form);err!=nil {response.Fail(c,101,"请选择登录会话");return}
    user:=service.AllService.UserService.CurUser(c)
    if _,err:=service.RevokeLoginSession(user.Id,form.ID,false);err!=nil {
        response.Fail(c,101,"登录会话不存在或已结束");return
    }
    response.Success(c,nil)
}
