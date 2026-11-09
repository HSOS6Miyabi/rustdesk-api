package service

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// An offline database copy is a documented recovery option for standalone
// SQLite deployments after the API has been stopped.
func TestSQLiteOfflineBackupAndRestore(t *testing.T) {
	work := t.TempDir()
	source := filepath.Join(work, "rustdesk-before-backup.db")
	backup := filepath.Join(work, "rustdesk-restored.db")
	db, err := gorm.Open(sqlite.Open(source), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&model.DevicePolicy{}, &model.AuditConn{},
		&model.ClientBuildProfile{}, &model.ClientBuildTask{},
		&model.NotificationSetting{}, &model.NotificationDelivery{}); err != nil { t.Fatal(err) }
	policy := &model.DevicePolicy{
		Name: "Remote office", IdWhitelist: "desk-001", AccessMode: "view",
		ManagedClientKeys: "access-mode", TrustDurationDays: 30,
	}
	if err := db.Create(policy).Error; err != nil { t.Fatal(err) }
	audit := &model.AuditConn{
		PeerId: "desk-001", FromPeer: "desk-002", ConnId: 45, Action: "close",
	}
	if err := db.Create(audit).Error; err != nil { t.Fatal(err) }
	profile := &model.ClientBuildProfile{
		Name: "Office", Platform: "linux-x64", Version: "1.5.1",
		Host: "desk.example.test", PublicKey: "test-only-public-key",
	}
	if err := db.Create(profile).Error; err != nil { t.Fatal(err) }
	task := &model.ClientBuildTask{
		ProfileID: profile.Id, RequestedBy: 3, Status: "ready",
		Filename: "rustdesk-linux-configured.zip",
		ConfigSnapshot: `{"host":"desk.example.test"}`,
		Platform: profile.Platform, Version: profile.Version,
	}
	if err := db.Create(task).Error; err != nil { t.Fatal(err) }
	connection, err := db.DB()
	if err != nil { t.Fatal(err) }
	if err := connection.Close(); err != nil { t.Fatal(err) }
	sourceFile, err := os.Open(source)
	if err != nil { t.Fatal(err) }
	targetFile, err := os.OpenFile(backup, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil { sourceFile.Close(); t.Fatal(err) }
	_, copyErr := io.Copy(targetFile, sourceFile)
	sourceFile.Close()
	if err := targetFile.Close(); err != nil { t.Fatal(err) }
	if copyErr != nil { t.Fatal(copyErr) }
	restored, err := gorm.Open(sqlite.Open(backup), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	restoreConn, err := restored.DB()
	if err != nil { t.Fatal(err) }
	defer restoreConn.Close()
	var previousPolicy model.DevicePolicy
	if err := restored.First(&previousPolicy, policy.Id).Error; err != nil { t.Fatal(err) }
	if previousPolicy.Name != policy.Name || previousPolicy.AccessMode != "view" ||
		previousPolicy.TrustDurationDays != 30 || previousPolicy.ManagedClientKeys != "access-mode" {
		t.Fatalf("policy was modified during recovery: %+v", previousPolicy)
	}
	var previousAudit model.AuditConn
	if err := restored.First(&previousAudit, audit.Id).Error; err != nil { t.Fatal(err) }
	if previousAudit.PeerId != audit.PeerId || previousAudit.ConnId != audit.ConnId ||
		previousAudit.Action != "close" {
		t.Fatalf("audit was modified during recovery: %+v", previousAudit)
	}
	var previousTask model.ClientBuildTask
	if err := restored.First(&previousTask, task.Id).Error; err != nil { t.Fatal(err) }
	if previousTask.ConfigSnapshot != task.ConfigSnapshot || previousTask.Status != "ready" {
		t.Fatalf("build history was modified during recovery: %+v", previousTask)
	}
	if err := restored.AutoMigrate(&model.DevicePolicy{}, &model.AuditConn{},
		&model.ClientBuildProfile{}, &model.ClientBuildTask{},
		&model.NotificationSetting{}, &model.NotificationDelivery{}); err != nil {
		t.Fatalf("restored database did not support another migration: %v", err)
	}
}
