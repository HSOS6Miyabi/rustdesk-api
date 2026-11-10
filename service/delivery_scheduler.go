package service

import (
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
	if !hasDeadline {
		<-wake
		return
	}
	if due <= 0 { return }
	timer := time.NewTimer(due)
	defer timer.Stop()
	select {
	case <-wake:
	case <-timer.C:
	}
}

// The scheduler restores persisted jobs at startup and after interruption.
// Future wakeups follow task inserts and retry deadlines, without polling.
func runDeliveryScheduler(wake <-chan struct{}, table interface{},
	maxAttempts int, leaseSeconds int64, process func(int) (int, error), category string) {
	for {
		if _, err := process(10); err != nil {
			if Logger != nil { Logger.WithError(err).Warn(fmt.Sprintf("%s delivery error", category)) }
			waitForWork(wake, 10*time.Second, true)
			continue
		}
		delay, pending, err := scheduledDeliveryDelay(table, maxAttempts, leaseSeconds)
		if err != nil {
			if Logger != nil { Logger.WithError(err).Warn(fmt.Sprintf("%s scheduling error", category)) }
			waitForWork(wake, 10*time.Second, true)
			continue
		}
		waitForWork(wake, delay, pending)
	}
}

func startPersonalDeliverySchedulers() {
	for i := 0; i < 2; i++ {
		go runDeliveryScheduler(pushDeliveryWake, &model.PushDelivery{},
			pushMaxAttempts, 60, ProcessPushQueue, "push")
	}
}

func startAdministrativeMailScheduler(wake <-chan struct{}) {
	go runDeliveryScheduler(wake, &model.NotificationDelivery{},
		3, 60, ProcessNotificationQueue, "administrative email")
}

