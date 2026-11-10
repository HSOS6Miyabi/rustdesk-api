package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"sync"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var notificationDeliveryWake = make(chan struct{}, 1)
var notificationEvents = make(chan administrativeEvent, 1024)
var notificationWorkerOnce sync.Once

type administrativeEvent struct {
	event, reference, description string
}

func wakeNotificationDelivery() {
	select { case notificationDeliveryWake <- struct{}{}: default: }
}

var notificationEventNames = map[string]string{
	"login.failure": "登录异常",
	"deployment.changed": "设备登记许可变更",
	"session.disconnect_requested": "远程会话结束请求",
	"session.ended": "远程会话已结束",
	"audit.file": "文件传输记录",
}

func NotificationEventCatalog() []string {
	events := make([]string, 0, len(notificationEventNames))
	for name := range notificationEventNames { events = append(events, name) }
	sort.Strings(events)
	return events
}

func CurrentNotificationSettings() (*model.NotificationSetting, error) {
	record := &model.NotificationSetting{}
	if DB == nil || !DB.Migrator().HasTable(record) {
		return &model.NotificationSetting{IdModel: model.IdModel{Id: 1}}, nil
	}
	err := DB.Where("id = ?", 1).First(record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &model.NotificationSetting{IdModel: model.IdModel{Id: 1}}, nil
	}
	return record, err
}

func SaveNotificationSettings(settings *model.NotificationSetting) error {
	if settings == nil { return errors.New("notification settings are required") }
	seen := map[string]bool{}
	recipients := make([]string, 0, 5)
	for _, part := range strings.FieldsFunc(settings.Recipients, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	}) {
		s := strings.ToLower(strings.TrimSpace(part))
		if s == "" { continue }
		addr, err := mail.ParseAddress(s)
		if err != nil || addr.Address != s || strings.ContainsAny(s, "\r\n") {
			return errors.New("notification email address is invalid")
		}
		if !seen[s] { seen[s] = true; recipients = append(recipients, s) }
	}
	if len(recipients) > 5 { return errors.New("no more than five notification recipients are supported") }
	seen = map[string]bool{}
	events := make([]string, 0, len(notificationEventNames))
	for _, part := range strings.FieldsFunc(settings.Events, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	}) {
		event := strings.TrimSpace(part)
		if event == "" { continue }
		if _, allowed := notificationEventNames[event]; !allowed {
			return fmt.Errorf("unsupported notification event %s", event)
		}
		if !seen[event] { seen[event] = true; events = append(events, event) }
	}
	if settings.Enabled && (len(recipients) == 0 || len(events) == 0) {
		return errors.New("enabled notifications require recipients and events")
	}
	sort.Strings(recipients)
	sort.Strings(events)
	settings.Recipients = strings.Join(recipients, ",")
	settings.Events = strings.Join(events, ",")
	settings.Id = 1
	return DB.Transaction(func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"enabled", "recipients", "events", "updated_at"}),
		}).Create(settings).Error
	})
}

func notificationReference(ref string) string {
	if ref == "" {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err == nil { ref = hex.EncodeToString(salt) }
	}
	sum := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(sum[:])
}

// QueueNotification writes an optional email to the delivery queue. It does
// not contact an SMTP server from within a user or device request.
func QueueNotification(event, reference, description string) error {
	label, supported := notificationEventNames[event]
	if !supported { return errors.New("unsupported notification event") }
	if DB == nil || !DB.Migrator().HasTable(&model.NotificationDelivery{}) { return nil }
	config, err := CurrentNotificationSettings()
	if err != nil || !config.Enabled { return err }
	enabled := false
	for _, item := range strings.Split(config.Events, ",") {
		if item == event { enabled = true; break }
	}
	if !enabled { return nil }
	if len(description) > 2000 { description = description[:2000] }
	description = strings.Map(func(r rune) rune {
		if r == '\x00' { return -1 }; return r
	}, description)
	now := time.Now().Unix()
	key := notificationReference(reference)
	for _, recipient := range strings.Split(config.Recipients, ",") {
		if recipient == "" { continue }
		row := &model.NotificationDelivery{
			EventType: event, EventKey: key, Recipient: recipient,
			Subject: "[RustDesk] " + label, Body: label + "\n\n" + description,
			Status: "pending", NextAttemptAt: now,
		}
		if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error; err != nil {
			return err
		}
	}
	wakeNotificationDelivery()
	return nil
}

// NotifyEvent records an optional alert without changing the result of the
// originating operation when email delivery is unavailable.
func NotifyEvent(event, reference, description string) {
	select {
	case notificationEvents <- administrativeEvent{event, reference, description}:
	default:
		if Logger != nil { Logger.Warn("administrative notification buffer is full") }
	}
}

func ListNotificationDeliveries(page, size uint) ([]model.NotificationDelivery, int64, error) {
	if page == 0 { page = 1 }
	if size == 0 { size = 20 }
	if size > 100 { size = 100 }
	query := DB.Model(&model.NotificationDelivery{})
	var total int64
	if err := query.Count(&total).Error; err != nil { return nil, 0, err }
	var rows []model.NotificationDelivery
	err := query.Order("id DESC").Scopes(Paginate(page, size)).Find(&rows).Error
	return rows, total, err
}

func RetryNotification(id uint) error {
	if id == 0 { return errors.New("invalid notification ID") }
	r := DB.Model(&model.NotificationDelivery{}).
		Where("id = ? AND status = ?", id, "failed").
		Updates(map[string]interface{}{"status": "pending", "next_attempt_at": time.Now().Unix(), "attempts": 0, "last_error": ""})
	if r.Error != nil { return r.Error }
	if r.RowsAffected == 0 { return errors.New("delivery is not eligible for retry") }
	wakeNotificationDelivery()
	return nil
}

// ProcessNotificationQueue sends a bounded batch with an atomic database
// claim. Sending status expires after 60 seconds if a worker was interrupted.
func ProcessNotificationQueue(limit int) (int, error) {
	if DB == nil || !DB.Migrator().HasTable(&model.NotificationDelivery{}) { return 0, nil }
	if limit < 1 || limit > 20 { limit = 10 }
	now := time.Now().Unix()
	var rows []model.NotificationDelivery
	err := DB.Where("(status IN ? AND attempts < ? AND next_attempt_at <= ?) OR (status = ? AND locked_at < ?)",
		[]string{"pending", "retry"}, 3, now, "sending", now - 60).
		Order("id ASC").Limit(limit).Find(&rows).Error
	if err != nil { return 0, err }
	sent := 0
	for _, row := range rows {
		now = time.Now().Unix()
		claim := DB.Model(&model.NotificationDelivery{}).
			Where("id = ? AND status = ? AND attempts = ? AND locked_at = ?", row.Id, row.Status, row.Attempts, row.LockedAt).
			Updates(map[string]interface{}{"status":"sending", "locked_at":now})
		if claim.Error != nil { return sent, claim.Error }
		if claim.RowsAffected != 1 { continue }
		sendErr := SendMail(row.Recipient, row.Subject, row.Body)
		values := map[string]interface{}{"attempts": row.Attempts+1, "locked_at":int64(0)}
		if sendErr == nil {
			values["status"] = "sent"
			values["sent_at"] = time.Now().Unix()
			values["last_error"] = ""
			sent++
		} else {
			message := sendErr.Error()
			if len(message) > 500 { message = message[:500] }
			values["last_error"] = message
			if row.Attempts+1 >= 3 { values["status"] = "failed" } else {
				values["status"] = "retry"
				values["next_attempt_at"] = time.Now().Unix() + int64(60*(row.Attempts+1))
			}
		}
		if err := DB.Model(&model.NotificationDelivery{}).Where("id = ? AND status = ?",row.Id,"sending").
			Updates(values).Error; err != nil { return sent, err }
	}
	return sent, nil
}

func StartNotificationWorker() {
	notificationWorkerOnce.Do(func() {
		go func() {
			for message := range notificationEvents {
				if err := QueueNotification(message.event, message.reference, message.description); err != nil && Logger != nil {
					Logger.WithError(err).Warn("unable to queue administrative notification")
				}
			}
		}()
		startAdministrativeMailScheduler(notificationDeliveryWake)
	})
}
