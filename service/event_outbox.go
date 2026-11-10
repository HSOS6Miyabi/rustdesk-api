package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

var pushOutboxWake = make(chan struct{}, 1)
var notificationOutboxWake = make(chan struct{}, 1)
var rejectedPushEvents atomic.Int64
var rejectedNotificationEvents atomic.Int64

// RejectedEventCounts reports events that could not be written to the durable
// event table. Callers can observe database failures without silent loss.
func RejectedEventCounts() (push, notifications int64) {
	return rejectedPushEvents.Load(), rejectedNotificationEvents.Load()
}

func wakePushOutbox() {
	select { case pushOutboxWake <- struct{}{}: default: }
}

func wakeNotificationOutbox() {
	select { case notificationOutboxWake <- struct{}{}: default: }
}

// persistPushEventRow can be called inside the same transaction that changes
// device presence, keeping the state transition and its event together.
func persistPushEventRow(tx *gorm.DB, event PushEvent) (*model.PushEventOutbox, error) {
	if tx == nil { return nil, errors.New("database is unavailable") }
	encoded, err := json.Marshal(event)
	if err != nil { return nil, err }
	record := &model.PushEventOutbox{Payload: string(encoded)}
	if err := tx.Create(record).Error; err != nil { return nil, err }
	return record, nil
}

func persistPushEvent(event PushEvent) error {
	if DB == nil { return errors.New("database is unavailable") }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := persistPushEventRow(DB.WithContext(ctx), event); err != nil {
		return err
	}
	wakePushOutbox()
	return nil
}

func persistNotificationEvent(event, reference, description string) error {
	if DB == nil { return errors.New("database is unavailable") }
	if len(reference) > 512 { reference = reference[:512] }
	if len(description) > 2000 { description = description[:2000] }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := DB.WithContext(ctx).Create(&model.NotificationEventOutbox{
		Event: event, Reference: reference, Description: description,
	}).Error; err != nil { return err }
	wakeNotificationOutbox()
	return nil
}

// ProcessPushEventOutbox is idempotent because delivery rows have a unique
// channel/event key. A crash between creating deliveries and deleting the
// outbox record can safely cause the record to be processed again.
func ProcessPushEventOutbox(limit int) (int, error) {
	if DB == nil { return 0, nil }
	if limit <= 0 || limit > 100 { limit = 20 }
	var rows []model.PushEventOutbox
	if err := DB.Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return 0, err
	}
	for _, row := range rows {
		var event PushEvent
		if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
			return 0, err
		}
		if err := processPushEvent(event); err != nil { return 0, err }
		if err := DB.Delete(&row).Error; err != nil { return 0, err }
	}
	return len(rows), nil
}

func ProcessNotificationEventOutbox(limit int) (int, error) {
	if DB == nil { return 0, nil }
	if limit <= 0 || limit > 100 { limit = 20 }
	var rows []model.NotificationEventOutbox
	if err := DB.Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return 0, err
	}
	for _, row := range rows {
		if err := QueueNotification(row.Event, row.Reference, row.Description); err != nil {
			return 0, err
		}
		if err := DB.Delete(&row).Error; err != nil { return 0, err }
	}
	return len(rows), nil
}

func startEventOutboxWorker(ctx context.Context, wake <-chan struct{},
	process func(int) (int, error), name string) {
	for {
		if ctx.Err() != nil { return }
		count, err := process(20)
		if err != nil {
			if Logger != nil { Logger.WithError(err).Warn(name+" event persistence failed") }
			if !waitForWorkContext(ctx, wake, 10*time.Second, true) { return }
			continue
		}
		if count == 20 { continue }
		if !waitForWorkContext(ctx, wake, 0, false) { return }
	}
}
