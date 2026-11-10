package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestIdleClientBuildSchedulerHasNoDeadline(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.ClientBuildTask{}); err != nil { t.Fatal(err) }
	if _, waiting, err := nextClientBuildDelay(); err != nil || waiting {
		t.Fatalf("idle worker scheduled a query: %v %v", waiting, err)
	}
	now := time.Now().Unix()
	task := &model.ClientBuildTask{Status:"pending", Filename:"installer.exe"}
	if err := DB.Create(task).Error; err != nil { t.Fatal(err) }
	if delay, waiting, err := nextClientBuildDelay(); err != nil || !waiting || delay != 0 {
		t.Fatalf("pending task was not immediately due: %v %v %v", delay, waiting, err)
	}
	if err := DB.Model(task).Updates(map[string]interface{}{"status":"building","started_at":now}).Error; err != nil {t.Fatal(err)}
	if delay, waiting, err := nextClientBuildDelay(); err != nil || !waiting || delay < 9*time.Minute {
		t.Fatalf("interrupted build deadline incorrect: %v %v %v", delay, waiting, err)
	}
	if err := DB.Model(task).Update("status","ready").Error; err != nil {t.Fatal(err)}
	if _, waiting, err := nextClientBuildDelay(); err != nil || waiting {
		t.Fatalf("completed task still has a deadline: %v %v", waiting, err)
	}
}
