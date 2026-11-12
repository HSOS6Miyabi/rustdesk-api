package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type APITokens struct{}

func (h *APITokens) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	query:=service.DB.Model(&model.APIToken{})
	var total int64
	if err:=query.Count(&total).Error;err!=nil{response.Fail(c,101,err.Error());return}
	var rows []model.APIToken
	if err:=query.Order("id DESC").Scopes(service.Paginate(page,size),service.SortScope(c.Query("sort_by"),c.Query("sort_dir"))).Find(&rows).Error;err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *APITokens) Create(c *gin.Context) {
	var form struct{
		Name string `json:"name" binding:"required"`
		UserId uint `json:"user_id"`
		Permissions string `json:"permissions"`
		ExpiresInDays int `json:"expires_in_days"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Fail(c,101,"请填写管理令牌名称");return
	}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Fail(c,403,"请先登录");return}
	userID:=user.Id
	if form.UserId!=0{userID=form.UserId}
	scopes:=[]string{"devices.assign","devices.deploy"}
	if strings.TrimSpace(form.Permissions)!="" {
		scopes=strings.Split(form.Permissions,",")
	}
	record,secret,err:=service.IssueAPIToken(userID,form.Name,scopes,form.ExpiresInDays)
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"id":record.Id,"token":secret,"prefix":record.Prefix})
}

func (h *APITokens) Delete(c *gin.Context) {
	var form struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&form);err!=nil{
		response.Fail(c,101,"管理令牌编号无效");return
	}
	if err:=service.RevokeAPIToken(form.Id);err!=nil{
		response.Fail(c,101,err.Error());return
	}
	response.Success(c,nil)
}
