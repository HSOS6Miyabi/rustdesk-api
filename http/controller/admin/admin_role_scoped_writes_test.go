package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestScopedAdministratorsCanModifyOnlyAssignedAccountsAndDevices(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scoped_writes_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{}, &model.Peer{}, &model.Group{},
		&model.GroupMembership{}, &model.AdminRole{}, &model.AdminRoleAssignment{},
		&model.DeviceDeployment{}, &model.UserToken{},
	); err != nil { t.Fatal(err) }
	oldDB, oldService := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{
		UserService: &service.UserService{},
		PeerService: &service.PeerService{},
	}
	defer func() { service.DB, service.AllService = oldDB, oldService }()
	oldStruct, oldVar := global.Validator.ValidStruct, global.Validator.ValidVar
	global.Validator.ValidStruct = func(*gin.Context, interface{}) []string { return nil }
	global.Validator.ValidVar = func(*gin.Context, interface{}, string) []string { return nil }
	defer func() {
		global.Validator.ValidStruct = oldStruct
		global.Validator.ValidVar = oldVar
	}()
	if err := db.Create(&model.Group{IdModel: model.IdModel{Id: 2}, Name: "Support", Type: 1}).Error; err != nil { t.Fatal(err) }
	no := false
	operator := &model.User{Username: "operator", GroupId: 1, Status: model.COMMON_STATUS_ENABLE, IsAdmin: &no}
	allowed := &model.User{Username: "team-owner", GroupId: 2, Status: model.COMMON_STATUS_ENABLE, IsAdmin: &no}
	recipient := &model.User{Username: "team-recipient", GroupId: 2, Status: model.COMMON_STATUS_ENABLE, IsAdmin: &no}
	private := &model.User{Username: "private-owner", GroupId: 1, Status: model.COMMON_STATUS_ENABLE, IsAdmin: &no}
	for _, user := range []*model.User{operator, allowed, recipient, private} {
		if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	}
	visible := &model.Peer{Id: "scoped-desk-001", Uuid: "scoped-uuid-001", UserId: allowed.Id, Hostname: "visible"}
	hidden := &model.Peer{Id: "scoped-desk-002", Uuid: "scoped-uuid-002", UserId: private.Id, Hostname: "private"}
	for _, peer := range []*model.Peer{visible, hidden} {
		if err := db.Create(peer).Error; err != nil { t.Fatal(err) }
	}
	deployment := &model.DeviceDeployment{
		DeviceId: visible.Id, Uuid: visible.Uuid, PublicKey: "test-key",
		OwnerUserId: allowed.Id, Status: "approved",
	}
	if err := db.Create(deployment).Error; err != nil { t.Fatal(err) }
	role := &model.AdminRole{
		Name: "Group support editor",
		Permissions: "users.view,users.edit,users.delete,devices.view,devices.edit_info,devices.delete,devices.transfer",
		ScopeType: service.AdminScopeGroup, ScopeId: 2, Enabled: true,
	}
	if err := service.SaveAdminRole(role); err != nil { t.Fatal(err) }
	if err := service.SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId: operator.Id, RoleId: role.Id,
	}); err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("curUser", operator); c.Next() })
	r := router.Group("/api/admin")
	users := &User{}
	peers := &Peer{}
	r.POST("/user/update", middleware.AdminPrivilege(), users.Update)
	r.POST("/user/delete", middleware.AdminPrivilege(), users.Delete)
	r.POST("/peer/update", middleware.AdminPrivilege(), peers.Update)
	r.POST("/peer/delete", middleware.AdminPrivilege(), peers.Delete)
	r.POST("/peer/batchDelete", middleware.AdminPrivilege(), peers.BatchDelete)
	r.POST("/peer/transfer", middleware.AdminPrivilege(), peers.Transfer)

	send := func(path string, payload interface{}) string {
		t.Helper()
		data, err := json.Marshal(payload)
		if err != nil { t.Fatal(err) }
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	success := func(body string) {
		t.Helper()
		if !strings.Contains(body, "\"code\":0") {
			t.Fatalf("expected successful operation, got: %s", body)
		}
	}
	denied := func(body string) {
		t.Helper()
		if !strings.Contains(body, "\"code\":403") {
			t.Fatalf("expected scoped permission rejection, got: %s", body)
		}
	}

	success(send("/api/admin/peer/update", map[string]interface{}{
		"row_id": visible.RowId, "id": visible.Id, "uuid": visible.Uuid, "note": "updated by team support",
	}))
	denied(send("/api/admin/peer/update", map[string]interface{}{
		"row_id": hidden.RowId, "id": hidden.Id, "note": "unwanted",
	}))
	denied(send("/api/admin/peer/update", map[string]interface{}{
		"row_id": visible.RowId, "id": "another-id", "note": "unwanted",
	}))
	var check model.Peer
	if err := db.Where("row_id = ?", visible.RowId).First(&check).Error; err != nil { t.Fatal(err) }
	if check.Note != "updated by team support" || check.Id != visible.Id {
		t.Fatalf("device editing did not maintain identity: %+v", check)
	}

	denied(send("/api/admin/peer/transfer", map[string]interface{}{
		"row_id": visible.RowId, "user_id": private.Id,
	}))
	denied(send("/api/admin/peer/transfer", map[string]interface{}{
		"row_id": hidden.RowId, "user_id": allowed.Id,
	}))
	success(send("/api/admin/peer/transfer", map[string]interface{}{
		"row_id": visible.RowId, "user_id": recipient.Id,
	}))
	if err := db.Where("row_id = ?", visible.RowId).First(&check).Error; err != nil { t.Fatal(err) }
	if check.UserId != recipient.Id { t.Fatalf("device ownership was not updated: %+v", check) }
	if err := db.Where("device_id = ?", visible.Id).First(deployment).Error; err != nil { t.Fatal(err) }
	if deployment.OwnerUserId != recipient.Id { t.Fatalf("deployment ownership was not updated: %+v", deployment) }

	denied(send("/api/admin/peer/batchDelete", map[string]interface{}{
		"row_ids": []uint{visible.RowId, hidden.RowId},
	}))
	var peerCount int64
	if err := db.Model(&model.Peer{}).Count(&peerCount).Error; err != nil { t.Fatal(err) }
	if peerCount != 2 { t.Fatalf("batch delete partially affected devices: %d", peerCount) }
	denied(send("/api/admin/peer/delete", map[string]interface{}{"row_id": hidden.RowId}))
	success(send("/api/admin/user/update", map[string]interface{}{
		"id": allowed.Id, "username": allowed.Username, "group_id": allowed.GroupId,
		"is_admin": false, "status": 1, "nickname": "Updated by support",
	}))
	denied(send("/api/admin/user/update", map[string]interface{}{
		"id": allowed.Id, "username": allowed.Username, "group_id": 1,
		"is_admin": false, "status": 1,
	}))
	denied(send("/api/admin/user/update", map[string]interface{}{
		"id": allowed.Id, "username": allowed.Username, "group_id": allowed.GroupId,
		"is_admin": true, "status": 1,
	}))
	denied(send("/api/admin/user/delete", map[string]interface{}{"id": private.Id}))
	var unchanged model.User
	if err := db.First(&unchanged, allowed.Id).Error; err != nil { t.Fatal(err) }
	if unchanged.Nickname != "Updated by support" || unchanged.GroupId != 2 ||
		unchanged.IsAdmin == nil || *unchanged.IsAdmin {
		t.Fatalf("scoped user update altered protected details: %+v", unchanged)
	}
}
