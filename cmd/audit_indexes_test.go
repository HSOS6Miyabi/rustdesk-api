package main

import (
	"os"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditRetentionIndexesAreIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_retention_indexes?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	connection, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer connection.Close()
	connection.SetMaxOpenConns(1)
	checkAuditRetentionIndexes(t, db)
}

// The database-upgrade workflow supplies dedicated temporary MySQL and
// PostgreSQL databases, so index creation is checked on all three drivers.
func TestAuditRetentionIndexesAcrossDrivers(t *testing.T) {
	databases := []struct {
		name string
		open func() (*gorm.DB, error)
	}{}
	if dsn := os.Getenv("RUSTDESK_TEST_MYSQL_DSN"); dsn != "" {
		databases = append(databases, struct {
			name string
			open func() (*gorm.DB, error)
		}{"mysql", func() (*gorm.DB, error) {
			return gorm.Open(mysql.Open(dsn), &gorm.Config{})
		}})
	}
	if dsn := os.Getenv("RUSTDESK_TEST_POSTGRES_DSN"); dsn != "" {
		databases = append(databases, struct {
			name string
			open func() (*gorm.DB, error)
		}{"postgresql", func() (*gorm.DB, error) {
			return gorm.Open(postgres.Open(dsn), &gorm.Config{})
		}})
	}
	if len(databases) == 0 { t.Skip("external database test DSNs are not configured") }
	for _, database := range databases {
		t.Run(database.name, func(t *testing.T) {
			db, err := database.open()
			if err != nil { t.Fatal(err) }
			connection, err := db.DB()
			if err != nil { t.Fatal(err) }
			defer connection.Close()
			checkAuditRetentionIndexes(t, db)
		})
	}
}

func checkAuditRetentionIndexes(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _,table:=range []interface{}{
		&model.AuditConn{},&model.AuditFile{},&model.AuditAlarm{},
		&model.LoginLog{},&model.AuditOperation{},
	}{
		if err:=db.AutoMigrate(table);err!=nil{t.Fatal(err)}
	}
	for i:=0;i<2;i++ {
		if err:=ensureAuditCleanupIndexes(db);err!=nil{t.Fatal(err)}
	}
	for _,entry:=range []struct{table interface{};name string}{
		{&model.AuditConn{},"idx_audit_conns_created_at"},
		{&model.AuditFile{},"idx_audit_files_created_at"},
		{&model.AuditAlarm{},"idx_audit_alarms_created_at"},
		{&model.LoginLog{},"idx_login_logs_created_at"},
		{&model.AuditOperation{},"idx_audit_operations_created_at"},
	}{
		if !db.Migrator().HasIndex(entry.table,entry.name){
			t.Fatalf("missing retention index %s",entry.name)
		}
	}
}
