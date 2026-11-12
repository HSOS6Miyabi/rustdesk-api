package admin

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type ClientBuilder struct{}

// Versions reports available stable official releases.
func (h *ClientBuilder) Versions(c *gin.Context) {
    items,err:=service.ListOfficialClientVersions(c.Request.Context())
    if err!=nil {
        response.Success(c,gin.H{"list":[]service.OfficialClientVersion{},
            "warning":"暂时无法读取官方版本列表，可以手动填写已发布的版本号"})
        return
    }
    response.Success(c,gin.H{"list":items})
}

func (h *ClientBuilder) Profiles(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	items,total,err:=service.ListClientBuildProfiles(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":items,"total":total,"page":page,"page_size":size})
}

func (h *ClientBuilder) SaveProfile(c *gin.Context) {
	var request model.ClientBuildProfile
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"请填写客户端配置");return}
	if err:=service.SaveClientBuildProfile(&request);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&request)
}

func (h *ClientBuilder) DeleteProfile(c *gin.Context) {
	var request struct{ID uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"请选择客户端配置");return}
	if err:=service.DropClientBuildProfile(request.ID);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func (h *ClientBuilder) Tasks(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	items,total,err:=service.ListClientBuildTasks(page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":items,"total":total,"page":page,"page_size":size})
}

func (h *ClientBuilder) Queue(c *gin.Context) {
	var request struct{ProfileID uint `json:"profile_id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"请选择客户端配置");return}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Fail(c,403,"请先登录");return}
	task,err:=service.QueueClientBuild(request.ProfileID,user.Id)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,task)
}

func (h *ClientBuilder) Retry(c *gin.Context) {
	var request struct{ID uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"请选择构建任务");return}
	if err:=service.RetryClientBuild(request.ID);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func (h *ClientBuilder) Download(c *gin.Context) {
	value,err:=strconv.ParseUint(c.Param("id"),10,32)
	if err!=nil||value==0{response.Error(c,"构建任务编号无效");return}
	file,name,size,err:=service.OpenVerifiedClientBuild(uint(value))
	if err!=nil{c.JSON(http.StatusNotFound,gin.H{"error":"客户端安装文件尚未生成或无法验证"});return}
	defer file.Close()
	c.Header("Cache-Control","no-store")
	c.Header("X-Content-Type-Options","nosniff")
	disposition:=mime.FormatMediaType("attachment",map[string]string{"filename":name})
	c.DataFromReader(http.StatusOK,size,"application/octet-stream",file,
		map[string]string{"Content-Disposition":disposition})
}
