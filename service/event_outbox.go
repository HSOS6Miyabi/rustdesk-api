package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
// Single API instances use the event-driven processing path. Shared workers
// are optional and claim events before generating delivery records.
func sharedEventWorkersEnabled() bool {
	return os.Getenv("RUSTDESK_API_SHARED_EVENT_WORKERS") == "1"
}

func claimOutboxRow(table interface{}, id uint) (bool, error) {
	now := time.Now().Unix()
	result := DB.Model(table).Where("id = ? AND lease_until <= ?", id, now).
		UpdateColumn("lease_until", now+60)
	if result.Error != nil { return false, result.Error }
	return result.RowsAffected == 1, nil
}

func releaseOutboxRow(table interface{}, id uint) error {
	return DB.Model(table).Where("id = ?", id).UpdateColumn("lease_until", 0).Error
}

func deleteProcessedOutboxRows(table interface{}, ids []uint) error {
	if len(ids) == 0 { return nil }
	return DB.Where("id IN ?", ids).Delete(table).Error
}

func ProcessPushEventOutbox(limit int) (int, error) {
	if DB == nil { return 0, nil }
	if limit <= 0 || limit > 100 { limit = 20 }
	shared := sharedEventWorkersEnabled()
	query := DB.Order("id ASC").Limit(limit)
	if shared { query = query.Where("lease_until <= ?", time.Now().Unix()) }
	var rows []model.PushEventOutbox
	if err := query.Find(&rows).Error; err != nil { return 0, err }
	processed := make([]uint, 0, len(rows))
	var processErr error
	for _, row := range rows {
		if shared {
			var claimed bool
			claimed, processErr = claimOutboxRow(&model.PushEventOutbox{}, row.Id)
			if processErr != nil { break }
			if !claimed { continue }
		}
		var event PushEvent
		processErr = json.Unmarshal([]byte(row.Payload), &event)
		if processErr == nil {
			if event.Reference == "" { event.Reference = fmt.Sprintf("outbox:%d", row.Id) }
			processErr = processPushEvent(event)
		}
		if processErr != nil {
			if shared {
				if releaseErr := releaseOutboxRow(&model.PushEventOutbox{}, row.Id); releaseErr != nil {
					processErr = fmt.Errorf("%w; release event: %v", processErr, releaseErr)
				}
			}
			break
		}
		processed = append(processed, row.Id)
	}
	if err := deleteProcessedOutboxRows(&model.PushEventOutbox{}, processed); err != nil {
		return 0, err
	}
	return len(processed), processErr
}

func ProcessNotificationEventOutbox(limit int) (int, error) {
	if DB == nil { return 0, nil }
	if limit <= 0 || limit > 100 { limit = 20 }
	shared := sharedEventWorkersEnabled()
	query := DB.Order("id ASC").Limit(limit)
	if shared { query = query.Where("lease_until <= ?", time.Now().Unix()) }
	var rows []model.NotificationEventOutbox
	if err := query.Find(&rows).Error; err != nil { return 0, err }
	processed := make([]uint, 0, len(rows))
	var processErr error
	for _, row := range rows {
		if shared {
			var claimed bool
			claimed, processErr = claimOutboxRow(&model.NotificationEventOutbox{}, row.Id)
			if processErr != nil { break }
			if !claimed { continue }
		}
		processErr = QueueNotification(row.Event, row.Reference, row.Description)
		if processErr != nil {
			if shared {
				if releaseErr := releaseOutboxRow(&model.NotificationEventOutbox{}, row.Id); releaseErr != nil {
					processErr = fmt.Errorf("%w; release event: %v", processErr, releaseErr)
				}
			}
			break
		}
		processed = append(processed, row.Id)
	}
	if err := deleteProcessedOutboxRows(&model.NotificationEventOutbox{}, processed); err != nil {
		return 0, err
	}
	return len(processed), processErr
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
		// Shared deployments periodically read events from other instances.
		// A single instance remains idle until its own event is written.
		if sharedEventWorkersEnabled() {
			if !waitForWorkContext(ctx, wake, 5*time.Second, true) { return }
		} else if !waitForWorkContext(ctx, wake, 0, false) { return }
	}
}
