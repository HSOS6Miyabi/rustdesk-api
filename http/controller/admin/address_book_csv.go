package admin

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// AddressBookCSV provides CSV import and export for one account at a time.
type AddressBookCSV struct{}

func addressBookCSVOwner(c *gin.Context, allowOther bool) uint {
	user := service.AllService.UserService.CurUser(c)
	if user == nil || user.Id == 0 { return 0 }
	if !allowOther { return user.Id }
	raw := strings.TrimSpace(c.Query("user_id"))
	if raw == "" { return user.Id }
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 { return 0 }
	return uint(value)
}

func (h *AddressBookCSV) export(c *gin.Context, admin bool) {
	ownerID := addressBookCSVOwner(c, admin)
	if ownerID == 0 { response.Error(c, "无效的账户编号"); return }
	var contents bytes.Buffer
	if err := service.ExportAddressBookCSV(ownerID, &contents); err != nil {
		response.Error(c, err.Error())
		return
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=rustdesk-addressbook.csv")
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", contents.Bytes())
}

func (h *AddressBookCSV) importCSV(c *gin.Context, admin bool) {
	ownerID := addressBookCSVOwner(c, admin)
	if ownerID == 0 { response.Error(c, "无效的账户编号"); return }
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
	defer c.Request.Body.Close()
	result, err := service.ImportAddressBookCSV(ownerID, c.Request.Body)
	if err != nil { response.Error(c, err.Error()); return }
	response.Success(c, result)
}

func (h *AddressBookCSV) MyExport(c *gin.Context) { h.export(c, false) }
func (h *AddressBookCSV) MyImport(c *gin.Context) { h.importCSV(c, false) }
func (h *AddressBookCSV) AdminExport(c *gin.Context) { h.export(c, true) }
func (h *AddressBookCSV) AdminImport(c *gin.Context) { h.importCSV(c, true) }
