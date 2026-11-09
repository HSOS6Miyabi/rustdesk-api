package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

func (ct *User) ResetSecondFactors(c *gin.Context){
	var request struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{
		response.Error(c,"请选择要重置双重认证的用户");return
	}
	if err:=service.ResetUserSecondFactors(request.Id);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}
