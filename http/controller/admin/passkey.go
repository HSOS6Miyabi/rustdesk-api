package admin

import (
    "encoding/json"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

type Passkey struct {}

func passkeyError(c *gin.Context,err error){
    if err==nil{return}
    response.Fail(c,101,err.Error())
}

func (h *Passkey) List(c *gin.Context){
    current:=service.AllService.UserService.CurUser(c)
    if current==nil||current.Id==0 {response.Fail(c,403,"请先登录");return}
    rows,err:=service.ListAccountPasskeys(current.Id)
    if err!=nil{passkeyError(c,err);return}
    response.Success(c,gin.H{"list":rows,"total":len(rows)})
}

func (h *Passkey) RegisterBegin(c *gin.Context){
    current:=service.AllService.UserService.CurUser(c)
    if current==nil||current.Id==0{response.Fail(c,403,"请先登录");return}
    options,session,err:=service.BeginPasskeyRegistration(
        current.Id,c.GetHeader("Origin"),c.Request.Host,c.ClientIP())
    if err!=nil{passkeyError(c,err);return}
    response.Success(c,gin.H{"options":options,"session_id":session})
}

func (h *Passkey) RegisterFinish(c *gin.Context){
    var form struct{
        SessionID string `json:"session_id" binding:"required"`
        Name string `json:"name"`
        Credential json.RawMessage `json:"credential" binding:"required"`
    }
    c.Request.Body=http.MaxBytesReader(c.Writer,c.Request.Body,32768)
    if err:=c.ShouldBindJSON(&form);err!=nil{
        response.Fail(c,101,"请提交有效的 Passkey 注册信息");return
    }
    current:=service.AllService.UserService.CurUser(c)
    if current==nil||current.Id==0{response.Fail(c,403,"请先登录");return}
    row,err:=service.FinishPasskeyRegistration(
        current.Id,c.GetHeader("Origin"),c.Request.Host,form.SessionID,form.Name,form.Credential)
    if err!=nil{passkeyError(c,err);return}
    response.Success(c,gin.H{"id":row.Id,"name":row.Name})
}

func (h *Passkey) Revoke(c *gin.Context){
    var form struct{ID uint `json:"id" binding:"required"`}
    if err:=c.ShouldBindJSON(&form);err!=nil{
        response.Fail(c,101,"请选择需要移除的 Passkey");return
    }
    current:=service.AllService.UserService.CurUser(c)
    if current==nil||current.Id==0{response.Fail(c,403,"请先登录");return}
    if err:=service.RevokeAccountPasskey(current.Id,form.ID);err!=nil{
        response.Fail(c,101,"Passkey 不存在或已被移除");return
    }
    response.Success(c,nil)
}

func (h *Passkey) LoginBegin(c *gin.Context){
    options,session,err:=service.BeginPasskeyLogin(
        c.GetHeader("Origin"),c.Request.Host,c.ClientIP())
    if err!=nil{passkeyError(c,err);return}
    response.Success(c,gin.H{"options":options,"session_id":session})
}

func (h *Passkey) LoginFinish(c *gin.Context){
    var form struct{
        SessionID string `json:"session_id" binding:"required"`
        Credential json.RawMessage `json:"credential" binding:"required"`
    }
    c.Request.Body=http.MaxBytesReader(c.Writer,c.Request.Body,32768)
    if err:=c.ShouldBindJSON(&form);err!=nil{
        response.Fail(c,101,"请提交有效的 Passkey 登录信息");return
    }
    user,err:=service.FinishPasskeyLogin(
        c.GetHeader("Origin"),c.Request.Host,form.SessionID,c.ClientIP(),form.Credential)
    if err!=nil{
        if globalLimiter:=service.AllService;globalLimiter!=nil {
            // Invalid assertions never issue an access token.
        }
        response.Fail(c,101,"Passkey 验证失败，请重新尝试");return
    }
    session:=service.AllService.UserService.Login(user,&model.LoginLog{
        UserId:user.Id,Client:model.LoginLogClientWebAdmin,
        Ip:c.ClientIP(),Type:"passkey",Platform:"web",
    })
    responseLoginSuccess(c,user,session.Token)
}
