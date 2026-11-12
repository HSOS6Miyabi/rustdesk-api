package admin

import (
    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/response"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

func (*PushChannels) StreamKeys(c *gin.Context) {
    keys, err := service.ListPushStreamTokens(currentPushUser(c))
    if err != nil { response.Error(c, err.Error()); return }
    response.Success(c, gin.H{"list": keys})
}

func (*PushChannels) CreateStreamKey(c *gin.Context) {
    var input struct {
        Name string `json:"name" binding:"required"`
        ExpiresInDays int `json:"expires_in_days"`
    }
    if err := c.ShouldBindJSON(&input); err != nil {
        response.Error(c, "请填写密钥名称"); return
    }
    record, key, err := service.IssueAPIToken(currentPushUser(c), input.Name, []string{"events.read"}, input.ExpiresInDays)
    if err != nil { response.Error(c, err.Error()); return }
    response.Success(c, gin.H{"id": record.Id, "token": key, "prefix": record.Prefix})
}

func (*PushChannels) RevokeStreamKey(c *gin.Context) {
    var input struct { ID uint `json:"id" binding:"required"` }
    if err := c.ShouldBindJSON(&input); err != nil {
        response.Error(c, "请选择要撤销的密钥"); return
    }
    if err := service.RevokePushStreamToken(currentPushUser(c), input.ID); err != nil {
        response.Error(c, err.Error()); return
    }
    response.Success(c, nil)
}
