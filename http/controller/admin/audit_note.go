package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

func(a *Audit) UpdateNote(c *gin.Context) {
	var request struct{
		Id uint `json:"id" binding:"required"`
		Note string `json:"note"`
	}
	if err:=c.ShouldBindJSON(&request);err!=nil{
		response.Error(c,"请指定连接记录");return
	}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Error(c,"请先登录");return}
	if err:=service.UpdateConnectionNote(request.Id,user.Id,request.Note);err!=nil{
		response.Error(c,err.Error());return
	}
	response.Success(c,nil)
}
