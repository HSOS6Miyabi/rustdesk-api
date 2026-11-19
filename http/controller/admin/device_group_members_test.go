package admin

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviceGroupMemberRoutesEnforcePermissionsAndConflicts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_group_members_api?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	if err := db.AutoMigrate(&model.User{}, &model.Peer{}, &model.Group{}, &model.GroupMembership{}, &model.DeviceGroup{}, &model.AdminRole{}, &model.AdminRoleAssignment{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldService := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}, GroupService: &service.GroupService{}}
	defer func() { service.DB, service.AllService = oldDB, oldService }()
	no := false
	viewer := &model.User{Username: "viewer", Status: 1, IsAdmin: &no}
	editor := &model.User{Username: "editor", Status: 1, IsAdmin: &no}
	ordinary := &model.User{Username: "ordinary", Status: 1, IsAdmin: &no}
	for _, user := range []*model.User{viewer, editor, ordinary} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, user := range []*model.User{viewer, editor} {
		permission := []string{"device_groups.view", "device_groups.view,device_groups.edit"}[i]
		role := &model.AdminRole{Name: user.Username, Permissions: permission, ScopeType: "global", Enabled: true}
		if err := service.SaveAdminRole(role); err != nil {
			t.Fatal(err)
		}
		if err := service.SaveAdminRoleAssignment(&model.AdminRoleAssignment{UserId: user.Id, RoleId: role.Id}); err != nil {
			t.Fatal(err)
		}
	}
	group := &model.DeviceGroup{Name: "Servers"}
	db.Create(group)
	peer := &model.Peer{Id: "server-1", Alias: "Application server", UserId: ordinary.Id}
	db.Create(peer)
	current := viewer
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("curUser", current); c.Next() })
	endpoints := router.Group("/api/admin/device_group", middleware.AdminPrivilege())
	controller := &DeviceGroup{}
	endpoints.GET("/devices", controller.Devices)
	endpoints.GET("/candidates", controller.Candidates)
	endpoints.POST("/devices/batch", controller.BatchDevices)
	send := func(method, path string, payload interface{}) map[string]interface{} {
		t.Helper()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(method, "/api/admin/device_group"+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		var result map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	check := func(code int, method, path string, body interface{}) map[string]interface{} {
		t.Helper()
		result := send(method, path, body)
		if result["code"] != float64(code) {
			t.Fatalf("%s %s: %+v", method, path, result)
		}
		return result
	}
	form := map[string]interface{}{"group_id": group.Id, "action": "add", "devices": []map[string]interface{}{{"row_id": peer.RowId, "expected_group_id": 0}}}
	check(0, "GET", "/devices?group_id=1", nil)
	check(403, "GET", "/candidates?group_id=1", nil)
	check(403, "POST", "/devices/batch", form)
	current = ordinary
	check(403, "GET", "/devices?group_id=1", nil)
	current = editor
	reply := check(0, "GET", "/candidates?group_id=1&keyword=server-1", nil)
	data := reply["data"].(map[string]interface{})
	if data["total"] != float64(1) {
		t.Fatalf("missing candidate: %+v", data)
	}
	check(0, "POST", "/devices/batch", form)
	check(409, "POST", "/devices/batch", form)
	check(101, "GET", "/devices?group_id=-1", nil)
	check(101, "GET", "/devices?group_id=1&page=invalid", nil)
	check(101, "POST", "/devices/batch", map[string]interface{}{"group_id": 1, "action": "remove", "devices": []map[string]interface{}{{"row_id": peer.RowId}}})
	form["action"] = "remove"
	form["devices"] = []map[string]interface{}{{"row_id": peer.RowId, "expected_group_id": group.Id}}
	check(0, "POST", "/devices/batch", form)
	var saved model.Peer
	db.First(&saved, "row_id = ?", peer.RowId)
	if saved.GroupId != 0 || saved.UserId != ordinary.Id || saved.Alias != "Application server" {
		t.Fatalf("device changed unexpectedly: %+v", saved)
	}
}
