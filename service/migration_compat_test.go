package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/model/custom_types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyMigrationGroup struct {
	Id uint `gorm:"primaryKey"`
	Name string
	Type int
}
func (legacyMigrationGroup) TableName() string { return "groups" }

type legacyMigrationConn struct {
	Id uint `gorm:"primaryKey"`
	PeerId string
	FromPeer string
	ConnId int64
	FromName string
}
func (legacyMigrationConn) TableName() string { return "audit_conns" }

type legacyMigrationFile struct {
	Id uint `gorm:"primaryKey"`
	PeerId string
	FromPeer string
	Path string
}
func (legacyMigrationFile) TableName() string { return "audit_files" }

func TestUpgradeExistingSQLiteData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:legacy_upgrade_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	sql, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer sql.Close()
	sql.SetMaxOpenConns(1)

	if err := db.AutoMigrate(
		&legacyMigrationGroup{}, &legacyMigrationConn{}, &legacyMigrationFile{},
		&model.AddressBook{}, &model.AddressBookCollection{},
	); err != nil { t.Fatal(err) }
	legacyGroups := []legacyMigrationGroup{
		{Id: 1, Name: "default", Type: 1},
		{Id: 7, Name: "Existing shared group", Type: 2},
	}
	for _, group := range legacyGroups {
		if err := db.Create(&group).Error; err != nil { t.Fatal(err) }
	}
	if err := db.Create(&legacyMigrationConn{
		Id: 10, ConnId: 1234, PeerId: "target-device", FromPeer: "source-device", FromName: "Alice",
	}).Error; err != nil { t.Fatal(err) }
	if err := db.Create(&legacyMigrationFile{
		Id: 11, PeerId: "target-device", FromPeer: "source-device", Path: "/old/report.pdf",
	}).Error; err != nil { t.Fatal(err) }

	collection := &model.AddressBookCollection{Name:"Existing address book", UserId: 3}
	if err := db.Create(collection).Error; err != nil { t.Fatal(err) }
	ab := &model.AddressBook{
		Id: "shared-device", UserId: 3, Alias: "Original alias", CollectionId: collection.Id,
		Tags: custom_types.AutoJson([]byte("[]")),
	}
	if err := db.Create(ab).Error; err != nil { t.Fatal(err) }

	// These are the additional tables and model fields deployed to existing users.
	if err := db.AutoMigrate(
		&model.Group{}, &model.GroupMembership{}, &model.GroupAnchor{},
		&model.Contact{}, &model.AuditConn{}, &model.AuditFile{},
		&model.AddressBook{}, &model.AddressBookCollection{},
	); err != nil { t.Fatal(err) }

	var migratedGroup model.Group
	if err := db.First(&migratedGroup, 7).Error; err != nil { t.Fatal(err) }
	if migratedGroup.Name != "Existing shared group" ||
		migratedGroup.Type != model.GroupTypeShare || migratedGroup.OwnerUserId != 0 {
		t.Fatalf("existing group changed: %+v", migratedGroup)
	}

	var migratedConn model.AuditConn
	if err := db.First(&migratedConn, 10).Error; err != nil { t.Fatal(err) }
	if migratedConn.ConnId != 1234 || migratedConn.PeerId != "target-device" ||
		migratedConn.FromPeer != "source-device" || migratedConn.FromName != "Alice" ||
		migratedConn.FromUserId != 0 || migratedConn.ToUserId != 0 {
		t.Fatalf("existing connection history changed: %+v", migratedConn)
	}

	var migratedFile model.AuditFile
	if err := db.First(&migratedFile, 11).Error; err != nil { t.Fatal(err) }
	if migratedFile.Path != "/old/report.pdf" || migratedFile.FromPeer != "source-device" {
		t.Fatalf("existing file history changed: %+v", migratedFile)
	}

	var migratedBook model.AddressBook
	if err := db.Where("id = ? AND user_id = ?", "shared-device", 3).First(&migratedBook).Error; err != nil {
		t.Fatal(err)
	}
	if migratedBook.CollectionId != collection.Id || migratedBook.Alias != "Original alias" {
		t.Fatalf("existing address book changed: %+v", migratedBook)
	}
	var migratedCollection model.AddressBookCollection
	if err := db.First(&migratedCollection, collection.Id).Error; err != nil { t.Fatal(err) }
	if migratedCollection.Name != "Existing address book" {
		t.Fatalf("existing address book collection changed: %+v", migratedCollection)
	}

	for _, name := range []string{"group_memberships", "group_anchors", "contacts"} {
		if !db.Migrator().HasTable(name) { t.Fatalf("missing new table %s", name) }
	}
	for _, index := range []struct{ table interface{}; name string }{
		{&model.GroupMembership{}, "idx_group_membership_user"},
		{&model.Contact{}, "idx_contact_recipient"},
		{&model.AuditConn{}, "idx_audit_conn_peer_connection"},
	} {
		if !db.Migrator().HasIndex(index.table,index.name) {
			t.Fatalf("missing query index %s after upgrade",index.name)
		}
	}

	// Re-applying migrations should preserve the data and the schema.
	if err := db.AutoMigrate(&model.Group{}, &model.GroupMembership{}, &model.GroupAnchor{},
		&model.Contact{}, &model.AuditConn{}, &model.AuditFile{}); err != nil { t.Fatal(err) }
	var count int64
	if err := db.Model(&model.Group{}).Where("name = ?", "Existing shared group").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 { t.Fatalf("repeated migration duplicated groups: %d", count) }
}
