package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRevokeDeviceSessionsPreservesUnrelatedEmptyUUID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_session_revoke?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	sql, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer sql.Close()
	sql.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.UserToken{}); err != nil { t.Fatal(err) }
	records := []model.UserToken{
		{UserId: 1, DeviceId: "target-device", Token: "target-by-id", ExpiredAt: 4102444800},
		{UserId: 1, DeviceUuid: "target-uuid", Token: "target-by-uuid", ExpiredAt: 4102444800},
		{UserId: 2, DeviceId: "other-device", Token: "unrelated-empty-uuid", ExpiredAt: 4102444800},
	}
	if err := db.Create(&records).Error; err != nil { t.Fatal(err) }
	checkRemaining := func(token string, expected int64) {
		t.Helper()
		var count int64
		if err := db.Model(&model.UserToken{}).Where("token = ?", token).
			Count(&count).Error; err != nil || count != expected {
			t.Fatalf("session %q count %d, expected %d: %v", token, count, expected, err)
		}
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return RevokeDeviceSessions(tx, "", "")
	}); err != nil { t.Fatal(err) }
	checkRemaining("unrelated-empty-uuid", 1)
	if err := db.Transaction(func(tx *gorm.DB) error {
		return RevokeDeviceSessions(tx, "target-device", "")
	}); err != nil { t.Fatal(err) }
	checkRemaining("target-by-id", 0)
	checkRemaining("target-by-uuid", 1)
	checkRemaining("unrelated-empty-uuid", 1)
	if err := db.Transaction(func(tx *gorm.DB) error {
		return RevokeDeviceSessions(tx, "another-device", "target-uuid")
	}); err != nil { t.Fatal(err) }
	checkRemaining("target-by-uuid", 0)
	checkRemaining("unrelated-empty-uuid", 1)
}
