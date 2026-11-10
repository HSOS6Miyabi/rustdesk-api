package main

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditRetentionIndexesAreIdempotent(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:audit_retention_indexes?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	connection,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer connection.Close()
	connection.SetMaxOpenConns(1)
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
