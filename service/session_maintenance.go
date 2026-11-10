package service

import (
	"context"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

const expiredSessionCleanupBatch = 300

var sessionMaintenanceWake = make(chan struct{}, 1)
var sessionMaintenanceOnce sync.Once

func wakeSessionMaintenance() {
	select {
	case sessionMaintenanceWake <- struct{}{}:
	default:
	}
}

// PruneExpiredSessions removes abandoned live-connection rows and expires only
// unfinished disconnect requests. Conditional updates preserve renewed rows.
func PruneExpiredSessions(now time.Time) (int, error) {
	if DB == nil { return 0, nil }
	deleted := 0
	cutoff := now.Unix() - sessionAliveSeconds
	for {
		var ids []uint
		if err := DB.Model(&model.DeviceActiveConnection{}).
			Where("last_seen_at < ?", cutoff).Order("id ASC").
			Limit(expiredSessionCleanupBatch).Pluck("id", &ids).Error; err != nil {
			return deleted, err
		}
		if len(ids) == 0 { break }
		result := DB.Where("id IN ? AND last_seen_at < ?", ids, cutoff).
			Delete(&model.DeviceActiveConnection{})
		if result.Error != nil { return deleted, result.Error }
		deleted += int(result.RowsAffected)
		if len(ids) < expiredSessionCleanupBatch { break }
	}
	for {
		var ids []uint
		if err := DB.Model(&model.DeviceDisconnectRequest{}).
			Where("status IN ? AND expires_at <= ?", []string{"pending", "delivered"}, now.Unix()).
			Order("id ASC").Limit(expiredSessionCleanupBatch).
			Pluck("id", &ids).Error; err != nil {
			return deleted, err
		}
		if len(ids) == 0 { break }
		result := DB.Model(&model.DeviceDisconnectRequest{}).
			Where("id IN ? AND status IN ? AND expires_at <= ?", ids, []string{"pending", "delivered"}, now.Unix()).
			Update("status", "expired")
		if result.Error != nil { return deleted, result.Error }
		deleted += int(result.RowsAffected)
		if len(ids) < expiredSessionCleanupBatch { break }
	}
	if deleted != 0 {
		invalidatePersistedSessions()
		invalidatePendingDisconnectRequests()
	}
	return deleted, nil
}

// The next expiry determines the wake-up time. The worker sleeps without
// querying the database while both live-session tables are empty.
func nextSessionMaintenanceDelay() (time.Duration, bool, error) {
	if DB == nil { return 0, false, nil }
	var active struct { Seen *int64 `gorm:"column:seen"` }
	if err := DB.Model(&model.DeviceActiveConnection{}).
		Select("MIN(last_seen_at) AS seen").Scan(&active).Error; err != nil {
		return 0, false, err
	}
	var request struct { Due *int64 `gorm:"column:due"` }
	if err := DB.Model(&model.DeviceDisconnectRequest{}).
		Select("MIN(expires_at) AS due").
		Where("status IN ?", []string{"pending", "delivered"}).
		Scan(&request).Error; err != nil {
		return 0, false, err
	}
	if active.Seen == nil && request.Due == nil {
		return 0, false, nil
	}
	var deadline int64
	if active.Seen != nil {
		deadline = *active.Seen + sessionAliveSeconds + 1
	}
	if request.Due != nil && (deadline == 0 || *request.Due < deadline) {
		deadline = *request.Due
	}
	delay := time.Until(time.Unix(deadline, 0))
	if delay < 0 { delay = 0 }
	return delay, true, nil
}

func StartSessionMaintenance() {
	sessionMaintenanceOnce.Do(func() {
		launchServiceWorker(func(ctx context.Context) {
			for {
				if ctx.Err() != nil { return }
				if _, err := PruneExpiredSessions(time.Now()); err != nil {
					if Logger != nil { Logger.WithError(err).Warn("session cleanup failed") }
					if !waitForWorkContext(ctx, sessionMaintenanceWake, 10*time.Second, true) { return }
					continue
				}
				delay, hasDeadline, err := nextSessionMaintenanceDelay()
				if err != nil {
					if Logger != nil { Logger.WithError(err).Warn("session scheduling failed") }
					if !waitForWorkContext(ctx, sessionMaintenanceWake, 10*time.Second, true) { return }
					continue
				}
				if !waitForWorkContext(ctx, sessionMaintenanceWake, delay, hasDeadline) { return }
			}
		})
	})
}
