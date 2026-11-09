package admin

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// ExportCSV exports at most 10000 devices in their current account scope.
// Exported values are escaped for spreadsheets and never include API tokens.
func (ct *Peer) ExportCSV(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 128 {
		response.Fail(c, 101, "设备搜索条件过长")
		return
	}
	var ownerID uint64
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err != nil || id == 0 {
			response.Fail(c, 101, "无效的所有者编号")
			return
		}
		ownerID = id
	}
	query := service.DB.Model(&model.Peer{}).
		Select("row_id,id,hostname,alias,uuid,os,username,user_id,last_online_time,last_online_ip,group_id,note,created_at").
		Order("row_id DESC").Limit(10000)
	if ownerID > 0 {
		query = query.Where("user_id = ?", ownerID)
	}
	if search != "" {
		pattern := "%" + strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(search,
			"\\", "\\\\"), "%", "\\%"), "_", "\\_") + "%"
		query = query.Where("(id LIKE ? ESCAPE '\\' OR hostname LIKE ? ESCAPE '\\' OR alias LIKE ? ESCAPE '\\')",
			pattern, pattern, pattern)
	}
	if value, exists := c.Get("adminRoleScope"); exists {
		scope, ok := value.(service.AdminRoleAccess)
		if !ok || (!scope.Global && len(scope.UserIDs) == 0) {
			response.Fail(c, 403, "当前账户无权导出设备")
			return
		}
		if !scope.Global {
			query = query.Where("user_id IN ?", scope.UserIDs)
		}
	}
	rows, err := query.Rows()
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	defer rows.Close()
	columns := []string{"row_id", "id", "hostname", "alias", "uuid",
		"os", "username", "user_id", "last_online_time", "last_online_ip",
		"group_id", "note", "created_at"}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "rustdesk-devices.csv"))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	writer := csv.NewWriter(c.Writer)
	if err := writer.Write(columns); err != nil { return }
	for rows.Next() {
		values := make([]interface{}, len(columns))
		dest := make([]interface{}, len(columns))
		for i := range dest { dest[i] = &values[i] }
		if err := rows.Scan(dest...); err != nil { return }
		record := make([]string, len(columns))
		for i, value := range values { record[i] = csvSafe(value) }
		if err := writer.Write(record); err != nil { return }
	}
	writer.Flush()
}
