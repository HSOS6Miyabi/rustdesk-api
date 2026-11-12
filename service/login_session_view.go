package service

import (
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

// LoginSessionView exposes session details without returning the credential.
type LoginSessionView struct {
    ID uint `json:"id"`
    UserID uint `json:"user_id"`
    Username string `json:"username"`
    DeviceID string `json:"device_id"`
    DeviceUUID string `json:"device_uuid"`
    LoginIP string `json:"login_ip"`
    Client string `json:"client"`
    Platform string `json:"platform"`
    CreatedAt interface{} `json:"created_at"`
    ExpiredAt int64 `json:"expired_at"`
    Active bool `json:"active"`
    Current bool `json:"current"`
}

func ListLoginSessions(userID uint, all bool, currentToken string, page, size uint, sortBy, sortDir string) ([]LoginSessionView, int64, error) {
    var total int64
    var tokens []model.UserToken
    query := DB.Model(&model.UserToken{})
    if !all {
        query = query.Where("user_id = ?", userID)
    }
    if err := query.Count(&total).Error; err != nil { return nil, 0, err }
    if err := query.Order("id DESC").Scopes(Paginate(page,size), SortScope(sortBy,sortDir)).Find(&tokens).Error; err != nil { return nil, 0, err }
    ids := make([]uint,0,len(tokens))
    seen := map[uint]bool{}
    for _, token := range tokens {
        if !seen[token.UserId] { ids=append(ids,token.UserId);seen[token.UserId]=true }
    }
    users:=[]model.User{}
    if len(ids)>0 {
        if err:=DB.Select("id","username").Where("id IN ?",ids).Find(&users).Error;err!=nil{return nil,0,err}
    }
    names:=map[uint]string{}
    for _, user:=range users { names[user.Id]=user.Username }
    tokenIDs:=make([]uint,0,len(tokens))
    for _,token:=range tokens {tokenIDs=append(tokenIDs,token.Id)}
    logsByID:=map[uint]model.LoginLog{}
    if len(tokenIDs)>0 {
        logs:=[]model.LoginLog{}
        if err:=DB.Where("user_token_id IN ?",tokenIDs).Order("id DESC").Find(&logs).Error;err!=nil {
            return nil,0,err
        }
        for _,log:=range logs {
            if _,ok:=logsByID[log.UserTokenId];!ok { logsByID[log.UserTokenId]=log }
        }
    }
    rows:=make([]LoginSessionView,0,len(tokens))
    for _, token:=range tokens {
        entry:=LoginSessionView{
            ID:token.Id, UserID:token.UserId, Username:names[token.UserId],
            DeviceID:token.DeviceId, DeviceUUID:token.DeviceUuid,
            CreatedAt:token.CreatedAt, ExpiredAt:token.ExpiredAt,
            Active:token.ExpiredAt>time.Now().Unix(),Current:token.Token!=""&&token.Token==currentToken,
        }
        // Prefer the login record linked to this credential. For older
        // records, match the last login from the same registered device.
        log,found:=logsByID[token.Id]
        if found && log.UserId!=token.UserId { found=false }
        if !found && (token.DeviceUuid!="" || token.DeviceId!="") {
            match:=DB.Where("user_id = ?",token.UserId)
            if token.DeviceUuid!="" {
                match=match.Where("uuid = ?",token.DeviceUuid)
            } else {
                match=match.Where("device_id = ?",token.DeviceId)
            }
            if err:=match.Order("id DESC").First(&log).Error;err==nil {found=true}
        }
        if found {
            entry.LoginIP=log.Ip
            entry.Client=log.Client
            entry.Platform=log.Platform
        }
        rows=append(rows,entry)
    }
    return rows,total,nil
}

func RevokeLoginSession(userID,id uint,all bool) (bool,error) {
    var token model.UserToken
    query:=DB.Where("id = ?",id)
    if !all { query=query.Where("user_id = ?",userID) }
    if err:=query.First(&token).Error;err!=nil{return false,err}
    return true,DB.Delete(&token).Error
}
