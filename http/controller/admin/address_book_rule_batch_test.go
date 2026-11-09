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

func TestBatchPersonalAddressBookSharingIsOwnerScopedAndAtomic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:address_book_rule_batch_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.AddressBookCollection{},
		&model.AddressBookCollectionRule{}); err != nil { t.Fatal(err) }
	oldDB, oldAll := service.DB, service.AllService
	service.DB = db
	service.AllService = &service.Service{UserService: &service.UserService{}}
	defer func() { service.DB, service.AllService = oldDB, oldAll }()
	owner := &model.User{Username: "sharing-owner", Status: model.COMMON_STATUS_ENABLE}
	other := &model.User{Username: "sharing-other", Status: model.COMMON_STATUS_ENABLE}
	one := &model.User{Username: "sharing-one", Status: model.COMMON_STATUS_ENABLE}
	two := &model.User{Username: "sharing-two", Status: model.COMMON_STATUS_ENABLE}
	disabled := &model.User{Username: "sharing-disabled", Status: model.COMMON_STATUS_DISABLED}
	for _, user := range []*model.User{owner, other, one, two, disabled} {
		if err := db.Create(user).Error; err != nil { t.Fatal(err) }
	}
	book := &model.AddressBookCollection{UserId: owner.Id, Name: "Operations"}
	otherBook := &model.AddressBookCollection{UserId: other.Id, Name: "Private"}
	for _, collection := range []*model.AddressBookCollection{book, otherBook} {
		if err := db.Create(collection).Error; err != nil { t.Fatal(err) }
	}
	otherShare := &model.AddressBookCollectionRule{
		UserId: other.Id, CollectionId: otherBook.Id,
		ToId: one.Id, Type: model.ShareAddressBookRuleTypePersonal, Rule: 1,
	}
	if err := db.Create(otherShare).Error; err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("curUser", owner); c.Next() })
	router.POST("/rules/batch", (&AddressBookCollectionRule{}).BatchPersonalShares)
	send := func(collectionID uint, userIDs []uint, action string, permission int) int {
		t.Helper()
		data, err := json.Marshal(gin.H{
			"collection_id": collectionID, "target_user_ids": userIDs,
			"action": action, "rule": permission,
		})
		if err != nil { t.Fatal(err) }
		request := httptest.NewRequest(http.MethodPost, "/rules/batch", bytes.NewReader(data))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct{ Code int `json:"code"` }
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("invalid JSON: %v %s", err, response.Body.String())
		}
		return envelope.Code
	}
	countGrants := func() int64 {
		t.Helper()
		var count int64
		if err := db.Model(&model.AddressBookCollectionRule{}).
			Where("user_id = ?", owner.Id).Count(&count).Error; err != nil { t.Fatal(err) }
		return count
	}
	if code := send(book.Id, []uint{one.Id, disabled.Id}, "share", 1); code == 0 || countGrants() != 0 {
		t.Fatalf("invalid batch caused partial sharing: code=%d", code)
	}
	if code := send(otherBook.Id, []uint{one.Id}, "share", 1); code == 0 || countGrants() != 0 {
		t.Fatalf("another account's collection was shared: code=%d", code)
	}
	if code := send(book.Id, []uint{one.Id, two.Id, one.Id}, "share", 1); code != 0 || countGrants() != 2 {
		t.Fatalf("bulk sharing failed: code=%d", code)
	}
	if code := send(book.Id, []uint{one.Id}, "share", 3); code != 0 || countGrants() != 2 {
		t.Fatalf("updating a previous share created a duplicate: code=%d", code)
	}
	var updated model.AddressBookCollectionRule
	if err := db.Where("user_id = ? AND collection_id = ? AND to_id = ?",
		owner.Id, book.Id, one.Id).First(&updated).Error; err != nil || updated.Rule != 3 {
		t.Fatalf("share permission not updated: %+v %v", updated, err)
	}
	if code := send(book.Id, []uint{one.Id, two.Id}, "revoke", 0); code != 0 || countGrants() != 0 {
		t.Fatalf("bulk share removal failed: code=%d", code)
	}
	var stillExists int64
	if err := db.Model(&model.AddressBookCollectionRule{}).
		Where("id = ?", otherShare.Id).Count(&stillExists).Error; err != nil || stillExists != 1 {
		t.Fatal("another account's address book sharing was modified")
	}
}
