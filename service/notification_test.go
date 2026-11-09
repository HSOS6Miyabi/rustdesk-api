package service

import (
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestNotificationSettingsQueueAndSMTPDelivery(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.SMTPSetting{}, &model.NotificationSetting{}, &model.NotificationDelivery{}); err != nil { t.Fatal(err) }
	if err := SaveNotificationSettings(&model.NotificationSetting{
		Enabled: true, Recipients: "invalid-email", Events: "deployment.changed",
	}); err == nil { t.Fatal("invalid notification email was accepted") }
	if err := SaveNotificationSettings(&model.NotificationSetting{
		Enabled: true, Recipients: "staff@example.test", Events: "not-a-real-event",
	}); err == nil { t.Fatal("unknown notification event was accepted") }

	setting := &model.NotificationSetting{
		Enabled: true, Recipients: "team@example.test,TEAM@example.test",
		Events: "deployment.changed,session.ended",
	}
	if err := SaveNotificationSettings(setting); err != nil { t.Fatal(err) }
	current, err := CurrentNotificationSettings()
	if err != nil || current.Recipients != "team@example.test" {
		t.Fatalf("recipient normalization was not saved: %+v %v", current, err)
	}
	if err := QueueNotification("login.failure", "login1", "no delivery"); err != nil { t.Fatal(err) }
	if err := QueueNotification("deployment.changed", "deploy-1", "部署已批准"); err != nil { t.Fatal(err) }
	if err := QueueNotification("deployment.changed", "deploy-1", "重复部署事件"); err != nil { t.Fatal(err) }
	if err := QueueNotification("session.ended", "session-1", "会话已结束"); err != nil { t.Fatal(err) }
	var rows []model.NotificationDelivery
	if err := DB.Order("id").Find(&rows).Error; err != nil { t.Fatal(err) }
	if len(rows) != 2 || rows[0].Status != "pending" || rows[1].Status != "pending" {
		t.Fatalf("unexpected notification queue: %+v", rows)
	}
	port, delivered := startMockSMTP(t)
	if _, err := SaveMailSettings(MailSettingsInput{
		Enabled: true, Host: "127.0.0.1", Port: port, Security: "none",
		SenderEmail: "rustdesk@example.test",
	}); err != nil { t.Fatal(err) }
	processed, err := ProcessNotificationQueue(1)
	if err != nil || processed != 1 { t.Fatalf("SMTP delivery failed: %d %v", processed, err) }
	select {
	case body := <-delivered:
		if !strings.Contains(body, "部署已批准") || !strings.Contains(body, "team@example.test") {
			t.Fatalf("notification content is invalid: %s", body)
		}
	case <-time.After(3*time.Second):
		t.Fatal("notification was not delivered")
	}
	var first model.NotificationDelivery
	if err := DB.First(&first, rows[0].Id).Error; err != nil { t.Fatal(err) }
	if first.Status != "sent" || first.Attempts != 1 || first.SentAt == 0 {
		t.Fatalf("delivery result was not recorded: %+v", first)
	}
	if err := DB.Model(&model.NotificationSetting{}).Where("id = ?",1).Update("enabled",false).Error; err != nil { t.Fatal(err) }
	if err := QueueNotification("deployment.changed","deploy-2","after disable"); err != nil { t.Fatal(err) }
	var count int64
	if err := DB.Model(&model.NotificationDelivery{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("disabled notifications produced new messages: %d %v", count, err)
	}
}

func TestNotificationFailureAndManualRetry(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.SMTPSetting{}, &model.NotificationSetting{}, &model.NotificationDelivery{}); err != nil { t.Fatal(err) }
	if err := SaveNotificationSettings(&model.NotificationSetting{
		Enabled: true, Recipients: "owner@example.test", Events: "login.failure",
	}); err != nil { t.Fatal(err) }
	if err := QueueNotification("login.failure", "bad-login", "来源 IP：192.0.2.12"); err != nil { t.Fatal(err) }
	var row model.NotificationDelivery
	if err := DB.First(&row).Error; err != nil { t.Fatal(err) }
	for attempt:=0;attempt<3;attempt++{
		if err:=DB.Model(&model.NotificationDelivery{}).Where("id = ?",row.Id).
			Update("next_attempt_at",time.Now().Unix()-1).Error;err!=nil{t.Fatal(err)}
		if _,err:=ProcessNotificationQueue(1);err!=nil{t.Fatal(err)}
		if err:=DB.First(&row,row.Id).Error;err!=nil{t.Fatal(err)}
	}
	if row.Status!="failed"||row.Attempts!=3||row.LastError=="" {
		t.Fatalf("failed SMTP delivery is not visible: %+v",row)
	}
	if err:=RetryNotification(row.Id);err!=nil{t.Fatal(err)}
	if err:=DB.First(&row,row.Id).Error;err!=nil{t.Fatal(err)}
	if row.Status!="pending"||row.Attempts!=0 {t.Fatalf("manual retry did not reset status: %+v",row)}
	if err:=RetryNotification(row.Id);err==nil{t.Fatal("pending delivery was retried twice")}
}
