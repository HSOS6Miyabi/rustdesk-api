package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type AddressBookTrash struct{}

func (h *AddressBookTrash) list(c *gin.Context,admin bool){
	ownerID:=addressBookCSVOwner(c,admin)
	if ownerID==0{response.Error(c,"无效的账户编号");return}
	page,size,err:=auditPage(c)
	if err!=nil{response.Error(c,"无效的页码");return}
	rows,total,err:=service.ListTrashedAddressBooks(ownerID,page,size)
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,gin.H{"list":rows,"total":total,"page":page,"page_size":size})
}

func (h *AddressBookTrash) act(c *gin.Context,admin,restore bool){
	var form struct{
		RowId uint `json:"row_id" binding:"required"`
		UserId uint `json:"user_id"`
	}
	if err:=c.ShouldBindJSON(&form);err!=nil{response.Error(c,"无效的地址簿条目编号");return}
	user:=service.AllService.UserService.CurUser(c)
	if user==nil||user.Id==0{response.Error(c,"请先登录");return}
	ownerID:=user.Id
	if admin&&form.UserId!=0{ownerID=form.UserId}
	var err error
	if restore{err=service.RestoreAddressBookEntry(ownerID,form.RowId)
	}else{err=service.PermanentlyDeleteAddressBookEntry(ownerID,form.RowId)}
	if err!=nil{response.Error(c,err.Error());return}
	response.Success(c,nil)
}

func(h *AddressBookTrash) MyList(c *gin.Context){h.list(c,false)}
func(h *AddressBookTrash) AdminList(c *gin.Context){h.list(c,true)}
func(h *AddressBookTrash) MyRestore(c *gin.Context){h.act(c,false,true)}
func(h *AddressBookTrash) AdminRestore(c *gin.Context){h.act(c,true,true)}
func(h *AddressBookTrash) MyPurge(c *gin.Context){h.act(c,false,false)}
func(h *AddressBookTrash) AdminPurge(c *gin.Context){h.act(c,true,false)}
