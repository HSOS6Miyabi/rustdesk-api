package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type SharedDevices struct {}

func (ct *SharedDevices) List(c *gin.Context) {
	queryPage, _ := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	querySize, _ := strconv.ParseUint(c.DefaultQuery("page_size", "20"), 10, 32)
	querySource, _ := strconv.ParseUint(c.DefaultQuery("source_id", "0"), 10, 32)
	current := service.AllService.UserService.CurUser(c)
	page, err := service.AllService.PeerService.SharedDeviceList(
		current, uint(queryPage), uint(querySize),
		c.Query("search"), c.DefaultQuery("source_type", "all"), uint(querySource),
	)
	if err != nil { response.Fail(c, 101, err.Error()); return }
	response.Success(c, page)
}
