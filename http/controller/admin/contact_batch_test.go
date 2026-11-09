package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestContactBatchRestrictsOwnershipAndAppliesAtomically(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:contact_batch_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Contact{}); err != nil { t.Fatal(err) }
	oldDB, oldAll := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}}
	defer func() { service.DB, service.AllService = oldDB, oldAll }()
	owner := &model.User{Username: "contact-batch-owner", Status: model.COMMON_STATUS_ENABLE}
	one := &model.User{Username: "contact-batch-one", Status: model.COMMON_STATUS_ENABLE}
	two := &model.User{Username: "contact-batch-two", Status: model.COMMON_STATUS_ENABLE}
	disabled := &model.User{Username: "contact-batch-disabled", Status: model.COMMON_STATUS_DISABLED}
	for _, u := range []*model.User{owner, one, two, disabled} {
		if err := db.Create(u).Error; err != nil { t.Fatal(err) }
	}
	if err := db.Create(&model.Contact{OwnerUserId: one.Id, RecipientUserId: two.Id}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("curUser", owner); c.Next() })
	router.POST("/contact/batch", (&Contact{}).Batch)
	send := func(action string, ids []uint) int {
		t.Helper()
		body, err := json.Marshal(gin.H{"action": action, "user_ids": ids})
		if err != nil { t.Fatal(err) }
		request := httptest.NewRequest(http.MethodPost, "/contact/batch", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK { return response.Code }
		var data struct{ Code int `json:"code"` }
		if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
			t.Fatalf("invalid JSON response: %s: %v", response.Body.String(), err)
		}
		return data.Code
	}
	countGrants := func(sender uint) int64 {
		t.Helper()
		var count int64
		if err := db.Model(&model.Contact{}).Where("owner_user_id = ?", sender).
			Count(&count).Error; err != nil { t.Fatal(err) }
		return count
	}
	if code := send("add", []uint{one.Id, disabled.Id}); code == 0 || countGrants(owner.Id) != 0 {
		t.Fatalf("disabled recipient caused partial sharing: code=%d", code)
	}
	if code := send("add", []uint{one.Id, owner.Id}); code == 0 || countGrants(owner.Id) != 0 {
		t.Fatalf("self sharing caused partial update: code=%d", code)
	}
	if code := send("add", []uint{one.Id, two.Id, one.Id}); code != 0 || countGrants(owner.Id) != 2 {
		t.Fatalf("valid batch sharing failed: code=%d", code)
	}
	if code := send("add", []uint{one.Id, two.Id}); code != 0 || countGrants(owner.Id) != 2 {
		t.Fatalf("repeated contact sharing was not idempotent: code=%d", code)
	}
	if code := send("remove", []uint{one.Id}); code != 0 || countGrants(owner.Id) != 1 {
		t.Fatalf("contact batch removal failed: code=%d", code)
	}
	if code := send("remove", []uint{two.Id}); code != 0 || countGrants(owner.Id) != 0 {
		t.Fatalf("second contact removal failed: code=%d", code)
	}
	if countGrants(one.Id) != 1 {
		t.Fatal("batch contact removal affected a different account's grants")
	}
	if code := send("ignore", []uint{two.Id}); code == 0 {
		t.Fatal("unknown batch action was accepted")
	}
}
