package service

import (
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// Multiple API instances recheck outstanding disconnect requests within a few
// seconds. The common no-request case performs no per-device SQL query.
const disconnectRequestRefresh = 5 * time.Second
const disconnectRequestSnapshotLimit = 1024

var pendingDisconnectActivity = struct {
	sync.Mutex
	refresh sync.Mutex
	db *gorm.DB
	generation uint64
	expires time.Time
	devices map[string]struct{}
	large bool
}{}

func invalidatePendingDisconnectRequests() {
	pendingDisconnectActivity.Lock()
	pendingDisconnectActivity.generation++
	pendingDisconnectActivity.expires = time.Time{}
	pendingDisconnectActivity.Unlock()
}

func deviceHasPendingDisconnectRequest(deviceID string) (bool, error) {
	if DB == nil { return false, nil }
	for {
		db := DB
		pendingDisconnectActivity.Lock()
		if pendingDisconnectActivity.db == db && time.Now().Before(pendingDisconnectActivity.expires) {
			_, present := pendingDisconnectActivity.devices[deviceID]
			large := pendingDisconnectActivity.large
			pendingDisconnectActivity.Unlock()
			return large || present, nil
		}
		pendingDisconnectActivity.Unlock()

		pendingDisconnectActivity.refresh.Lock()
		pendingDisconnectActivity.Lock()
		if pendingDisconnectActivity.db == db && time.Now().Before(pendingDisconnectActivity.expires) {
			_, present := pendingDisconnectActivity.devices[deviceID]
			large := pendingDisconnectActivity.large
			pendingDisconnectActivity.Unlock()
			pendingDisconnectActivity.refresh.Unlock()
			return large || present, nil
		}
		generation := pendingDisconnectActivity.generation
		pendingDisconnectActivity.Unlock()

		var rows []struct { DeviceId string }
		err := db.Model(&model.DeviceDisconnectRequest{}).Select("device_id").
			Where("status IN ? AND expires_at > ?", []string{"pending", "delivered"}, time.Now().Unix()).
			Limit(disconnectRequestSnapshotLimit+1).Find(&rows).Error
		devices := make(map[string]struct{}, len(rows))
		large := len(rows) > disconnectRequestSnapshotLimit
		if !large {
			for _, row := range rows { devices[row.DeviceId] = struct{}{} }
		}

		pendingDisconnectActivity.Lock()
		changed := pendingDisconnectActivity.generation != generation || DB != db
		if !changed && err == nil {
			pendingDisconnectActivity.db = db
			pendingDisconnectActivity.devices = devices
			pendingDisconnectActivity.large = large
			pendingDisconnectActivity.expires = time.Now().Add(disconnectRequestRefresh)
		}
		pendingDisconnectActivity.Unlock()
		pendingDisconnectActivity.refresh.Unlock()
		if changed { continue }
		if err != nil { return false, err }
		_, present := devices[deviceID]
		return large || present, nil
	}
}
