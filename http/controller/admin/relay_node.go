package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type RelayNodes struct{}

func (h *RelayNodes) List(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	nodes,total,err:=service.ListRelayNodes(page,size,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":nodes,"total":total,"page":page,"page_size":size})
}

func (h *RelayNodes) Create(c *gin.Context) {
	var node model.RelayNode
	if err:=c.ShouldBindJSON(&node);err!=nil{response.Error(c,"请填写中继节点信息");return}
	node.Id=0
	if err:=service.SaveRelayNode(&node);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&node)
}

func (h *RelayNodes) Update(c *gin.Context) {
	var node model.RelayNode
	if err:=c.ShouldBindJSON(&node);err!=nil||node.Id==0{
		response.Error(c,"请选择要修改的中继节点");return
	}
	if err:=service.SaveRelayNode(&node);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,&node)
}

func (h *RelayNodes) Delete(c *gin.Context) {
	var request struct{Id uint `json:"id" binding:"required"`}
	if err:=c.ShouldBindJSON(&request);err!=nil{response.Error(c,"无效的中继节点编号");return}
	if err:=service.DeleteRelayNode(request.Id);err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}
