package admin

import (
    "errors"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
    "github.com/lejianwen/rustdesk-api/v2/config"
)

type Settings struct{}

func (Settings) requireAdministrator(c *gin.Context) (uint, bool) {
    user:=service.AllService.UserService.CurUser(c)
    if user==nil||user.Id==0||!service.AllService.UserService.IsAdmin(user) {
        c.JSON(http.StatusForbidden,gin.H{"error":"只有系统管理员可以访问设置中心"})
        return 0,false
    }
    return user.Id,true
}
func (s Settings) Schema(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    response.Success(c,gin.H{
        "items":config.SettingDefinitions(),
        "categories":service.SettingsCategoryNames(),
    })
}
func (s Settings) Effective(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    view,err:=service.SettingStates()
    if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":err.Error()});return}
    if query:=c.Query("search");query!=""{view.Items=service.SettingsTextSearch(view.Items,query)}
    response.Success(c,view)
}
func (s Settings) Validate(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    var input service.SettingsRevisionInput
    if err:=c.ShouldBindJSON(&input);err!=nil{response.Error(c,err.Error());return}
    if err:=service.ValidateSettingChanges(input.Changes);err!=nil{response.Error(c,err.Error());return}
    response.Success(c,gin.H{"valid":true})
}
func handleSettingsWriteError(c *gin.Context,err error) {
    if errors.Is(err,service.ErrSettingsConflict) {
        c.JSON(http.StatusConflict,gin.H{"error":err.Error()})
    } else {
        response.Error(c,err.Error())
    }
}
func (s Settings) Save(c *gin.Context) {
    actor,ok:=s.requireAdministrator(c);if !ok{return}
    var input service.SettingsRevisionInput
    if err:=c.ShouldBindJSON(&input);err!=nil{response.Error(c,err.Error());return}
    if err:=service.SaveSettingChanges(actor,input);err!=nil{handleSettingsWriteError(c,err);return}
    response.Success(c,gin.H{"saved":true,"requires_restart":true})
}
func (s Settings) Reset(c *gin.Context) {
    actor,ok:=s.requireAdministrator(c);if !ok{return}
    var input service.SettingsRevisionInput
    if err:=c.ShouldBindJSON(&input);err!=nil{response.Error(c,err.Error());return}
    if err:=service.ResetSettingChanges(actor,input);err!=nil{handleSettingsWriteError(c,err);return}
    response.Success(c,gin.H{"saved":true,"requires_restart":true})
}
func (s Settings) History(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    n,_:=strconv.Atoi(c.DefaultQuery("limit","100"))
    rows,err:=service.ListSettingHistory(n)
    if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"无法读取设置修改记录"});return}
    response.Success(c,gin.H{"list":rows})
}
func (s Settings) Overview(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    summary,err:=service.SpecializedSettingsSummary()
    if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"无法读取专项设置状态"});return}
    push,err:=service.ReadOutboxStatus(&model.PushEventOutbox{})
    if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"无法读取推送任务状态"});return}
    notification,err:=service.ReadOutboxStatus(&model.NotificationEventOutbox{})
    if err!=nil{c.JSON(http.StatusInternalServerError,gin.H{"error":"无法读取邮件任务状态"});return}
    response.Success(c,gin.H{"summary":summary,"outbox":gin.H{"push":push,"notification":notification}})
}
func (s Settings) FailedOutbox(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    category:=c.DefaultQuery("category","push")
    entries,err:=service.FailedOutboxEntries(category,30)
    if err!=nil{response.Error(c,err.Error());return}
    response.Success(c,gin.H{"list":entries})
}
func (s Settings) RetryOutbox(c *gin.Context) {
    if _,ok:=s.requireAdministrator(c);!ok{return}
    var input struct { Category string `json:"category"`; ID uint `json:"id"` }
    if err:=c.ShouldBindJSON(&input);err!=nil{response.Error(c,err.Error());return}
    if err:=service.RetryFailedOutboxEvent(input.Category,input.ID);err!=nil{response.Error(c,err.Error());return}
    response.Success(c,gin.H{"retried":true})
}
