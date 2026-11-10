package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

// scheduledDeliveryDelay reads the earliest task deadline. There is no
// periodic database scan while the queue is empty.
func scheduledDeliveryDelay(table interface{}, attempts int, leaseSeconds int64) (time.Duration, bool, error) {
	if DB == nil { return 0, false, errors.New("database is not initialized") }
	var pending struct {
		Due *int64 `gorm:"column:due"`
	}
	if err := DB.Model(table).
		Select("MIN(next_attempt_at) AS due").
		Where("status IN ? AND attempts < ?", []string{"pending", "retry"}, attempts).
		Scan(&pending).Error; err != nil {
		return 0, false, err
	}
	var sending struct {
		Due *int64 `gorm:"column:due"`
	}
	if err := DB.Model(table).
		Select("MIN(locked_at) AS due").
		Where("status = ?", "sending").
		Scan(&sending).Error; err != nil {
		return 0, false, err
	}
	if pending.Due == nil && sending.Due == nil { return 0, false, nil }
	var deadline int64
	if pending.Due != nil { deadline = *pending.Due }
	if sending.Due != nil {
		recoveryTime := *sending.Due + leaseSeconds + 1
		if pending.Due == nil || recoveryTime < deadline {
			deadline = recoveryTime
		}
	}
	delay := time.Until(time.Unix(deadline, 0))
	if delay < 0 { delay = 0 }
	return delay, true, nil
}

// A timer only runs when a persisted task has a known deadline. New messages
// interrupt it through the wake channel.
func waitForWork(wake <-chan struct{}, due time.Duration, hasDeadline bool) {
	waitForWorkContext(context.Background(), wake, due, hasDeadline)
}

// The scheduler restores persisted jobs at startup and after interruption.
// Future wakeups follow task inserts and retry deadlines, without polling.
func runDeliveryScheduler(ctx context.Context, wake <-chan struct{}, table interface{},
	maxAttempts int, leaseSeconds int64, process func(int) (int, error), category string) {
	for {
		if ctx.Err() != nil { return }
		if _, err := process(10); err != nil {
			if Logger != nil { Logger.WithError(err).Warn(fmt.Sprintf("%s delivery error", category)) }
			if !waitForWorkContext(ctx, wake, 10*time.Second, true) { return }
			continue
		}
		delay, pending, err := scheduledDeliveryDelay(table, maxAttempts, leaseSeconds)
		if err != nil {
			if Logger != nil { Logger.WithError(err).Warn(fmt.Sprintf("%s scheduling error", category)) }
			if !waitForWorkContext(ctx, wake, 10*time.Second, true) { return }
			continue
		}
		if !waitForWorkContext(ctx, wake, delay, pending) { return }
	}
}

func startPersonalDeliverySchedulers() {
	for i := 0; i < 2; i++ {
		launchServiceWorker(func(ctx context.Context) {
			runDeliveryScheduler(ctx, pushDeliveryWake, &model.PushDelivery{},
				pushMaxAttempts, 60, ProcessPushQueue, "push")
		})
	}
}

func startAdministrativeMailScheduler(wake <-chan struct{}) {
	launchServiceWorker(func(ctx context.Context) {
		runDeliveryScheduler(ctx, wake, &model.NotificationDelivery{},
			3, 60, ProcessNotificationQueue, "administrative email")
	})
}

