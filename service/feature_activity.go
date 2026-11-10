package service

import (
	"strings"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const optionalFeatureRefresh = 30 * time.Second

// State mutexes protect snapshots only. A separate refresh mutex serializes
// database work without blocking readers of still-valid snapshots.
var policyActivity = struct {
	sync.Mutex
	refresh sync.Mutex
	db *gorm.DB
	generation uint64
	checked time.Time
	assigned bool
	snapshot policyAssignmentSnapshot
}{}

func invalidatePolicyActivity() {
	policyActivity.Lock()
	policyActivity.checked = time.Time{}
	policyActivity.generation++
	policyActivity.Unlock()
	heartbeatPolicyCache.Lock()
	heartbeatPolicyCache.generation++
	heartbeatPolicyCache.entries = make(map[uint]heartbeatPolicyEntry)
	heartbeatPolicyCache.Unlock()
}

// Each assignment level is read at most once per refresh interval. Small
// assignments are kept as immutable ID sets, allowing unrelated devices to
// skip all policy queries. Large assignments use indexed database lookups.
const sparsePolicyAssignmentLimit = 1024

type policyAssignmentSnapshot struct {
	devices map[string]struct{}
	users map[uint]struct{}
	groups map[uint]struct{}
	largeDevices bool
	largeUsers bool
	largeGroups bool
}

func (snapshot policyAssignmentSnapshot) mayApply(peer *model.Peer) bool {
	if peer == nil { return false }
	_, byDevice := snapshot.devices[peer.Id]
	_, byUser := snapshot.users[peer.UserId]
	_, byGroup := snapshot.groups[peer.GroupId]
	return snapshot.largeDevices || snapshot.largeUsers || snapshot.largeGroups ||
		byDevice || (peer.UserId != 0 && byUser) || (peer.GroupId != 0 && byGroup)
}

func readPolicyAssignmentSnapshot(db *gorm.DB) (policyAssignmentSnapshot, bool, error) {
	result := policyAssignmentSnapshot{
		devices: make(map[string]struct{}),
		users: make(map[uint]struct{}),
		groups: make(map[uint]struct{}),
	}
	assigned := false
	if db.Migrator().HasTable(&model.DevicePolicyAssignment{}) {
		var rows []struct { DeviceId string }
		if err := db.Model(&model.DevicePolicyAssignment{}).Select("device_id").
			Limit(sparsePolicyAssignmentLimit+1).Find(&rows).Error; err != nil { return result, true, err }
		assigned = assigned || len(rows) > 0
		if len(rows) > sparsePolicyAssignmentLimit {
			result.largeDevices = true
		} else {
			for _, row := range rows { result.devices[row.DeviceId] = struct{}{} }
		}
	}
	if db.Migrator().HasTable(&model.DevicePolicyUserAssignment{}) {
		var rows []struct { UserId uint }
		if err := db.Model(&model.DevicePolicyUserAssignment{}).Select("user_id").
			Limit(sparsePolicyAssignmentLimit+1).Find(&rows).Error; err != nil { return result, true, err }
		assigned = assigned || len(rows) > 0
		if len(rows) > sparsePolicyAssignmentLimit {
			result.largeUsers = true
		} else {
			for _, row := range rows { result.users[row.UserId] = struct{}{} }
		}
	}
	if db.Migrator().HasTable(&model.DevicePolicyGroupAssignment{}) {
		var rows []struct { GroupId uint }
		if err := db.Model(&model.DevicePolicyGroupAssignment{}).Select("group_id").
			Limit(sparsePolicyAssignmentLimit+1).Find(&rows).Error; err != nil { return result, true, err }
		assigned = assigned || len(rows) > 0
		if len(rows) > sparsePolicyAssignmentLimit {
			result.largeGroups = true
		} else {
			for _, row := range rows { result.groups[row.GroupId] = struct{}{} }
		}
	}
	return result, assigned, nil
}

func policyMayApplyToPeer(peer *model.Peer) bool {
	policyActivity.Lock()
	defer policyActivity.Unlock()
	if policyActivity.db != DB || policyActivity.checked.IsZero() ||
		time.Since(policyActivity.checked) >= optionalFeatureRefresh {
		return true
	}
	return policyActivity.snapshot.mayApply(peer)
}

func devicePolicyAssignmentsPresent() (bool, error) {
	if DB == nil { return false, nil }
	for {
		db := DB
		policyActivity.Lock()
		if policyActivity.db == db && !policyActivity.checked.IsZero() &&
			time.Since(policyActivity.checked) < optionalFeatureRefresh {
			value := policyActivity.assigned
			policyActivity.Unlock()
			return value, nil
		}
		policyActivity.Unlock()
		policyActivity.refresh.Lock()
		policyActivity.Lock()
		if policyActivity.db == db && !policyActivity.checked.IsZero() &&
			time.Since(policyActivity.checked) < optionalFeatureRefresh {
			value := policyActivity.assigned
			policyActivity.Unlock()
			policyActivity.refresh.Unlock()
			return value, nil
		}
		generation := policyActivity.generation
		policyActivity.Unlock()
		snapshot, assigned, readErr := readPolicyAssignmentSnapshot(db)
		policyActivity.Lock()
		changed := policyActivity.generation != generation || DB != db
		if !changed && readErr == nil {
			policyActivity.db = db
			policyActivity.assigned = assigned
			policyActivity.snapshot = snapshot
			policyActivity.checked = time.Now()
		}
		policyActivity.Unlock()
		policyActivity.refresh.Unlock()
		if changed { continue }
		if readErr != nil { return true, readErr }
		return assigned, nil
	}
}

var pushActivity = struct {
	sync.Mutex
	refresh sync.Mutex
	db *gorm.DB
	generation uint64
	checked time.Time
	events map[string]bool
}{}

func invalidatePushActivity() {
	pushActivity.Lock()
	pushActivity.checked = time.Time{}
	pushActivity.generation++
	pushActivity.Unlock()
}

func hasPushSubscription(code string) bool {
	if DB == nil {
		return false
	}
	for {
		db := DB
		pushActivity.Lock()
		if pushActivity.db == db && !pushActivity.checked.IsZero() &&
			time.Since(pushActivity.checked) < optionalFeatureRefresh {
			value := pushActivity.events[code]
			pushActivity.Unlock()
			return value
		}
		pushActivity.Unlock()
		pushActivity.refresh.Lock()
		pushActivity.Lock()
		if pushActivity.db == db && !pushActivity.checked.IsZero() &&
			time.Since(pushActivity.checked) < optionalFeatureRefresh {
			value := pushActivity.events[code]
			pushActivity.Unlock()
			pushActivity.refresh.Unlock()
			return value
		}
		generation := pushActivity.generation
		pushActivity.Unlock()

		events := make(map[string]bool)
		var readErr error
		if db.Migrator().HasTable(&model.PushChannel{}) {
			var channels []struct{ Events string }
			readErr = db.Model(&model.PushChannel{}).Select("events").
				Where("enabled = ?", true).Find(&channels).Error
			if readErr == nil {
				for _, channel := range channels {
					for _, event := range strings.Split(channel.Events, ",") {
						if event != "" {
							events[event] = true
						}
					}
				}
			}
		}
		pushActivity.Lock()
		changed := pushActivity.generation != generation || DB != db
		if !changed && readErr == nil {
			pushActivity.db = db
			pushActivity.events = events
			pushActivity.checked = time.Now()
		}
		pushActivity.Unlock()
		pushActivity.refresh.Unlock()
		if changed {
			continue
		}
		if readErr != nil {
			// Retry event persistence when a failed database read might
			// otherwise silence an active subscription.
			return true
		}
		return events[code]
	}
}

var administrativeNotificationActivity = struct {
	sync.Mutex
	refresh sync.Mutex
	db *gorm.DB
	generation uint64
	checked time.Time
	events map[string]bool
	settings model.NotificationSetting
}{}

func invalidateAdministrativeNotificationActivity() {
	administrativeNotificationActivity.Lock()
	administrativeNotificationActivity.checked = time.Time{}
	administrativeNotificationActivity.generation++
	administrativeNotificationActivity.Unlock()
}

// notificationSettingsForEvent shares the event-interest snapshot with the
// delivery writer, avoiding another settings query for each accepted event.
func notificationSettingsForEvent(event string) (*model.NotificationSetting, error) {
	if DB == nil {
		return nil, nil
	}
	for {
		db := DB
		administrativeNotificationActivity.Lock()
		if administrativeNotificationActivity.db == db &&
			!administrativeNotificationActivity.checked.IsZero() &&
			time.Since(administrativeNotificationActivity.checked) < optionalFeatureRefresh {
			enabled := administrativeNotificationActivity.events[event]
			settings := administrativeNotificationActivity.settings
			administrativeNotificationActivity.Unlock()
			if !enabled {
				return nil, nil
			}
			return &settings, nil
		}
		administrativeNotificationActivity.Unlock()

		administrativeNotificationActivity.refresh.Lock()
		administrativeNotificationActivity.Lock()
		if administrativeNotificationActivity.db == db &&
			!administrativeNotificationActivity.checked.IsZero() &&
			time.Since(administrativeNotificationActivity.checked) < optionalFeatureRefresh {
			enabled := administrativeNotificationActivity.events[event]
			settings := administrativeNotificationActivity.settings
			administrativeNotificationActivity.Unlock()
			administrativeNotificationActivity.refresh.Unlock()
			if !enabled {
				return nil, nil
			}
			return &settings, nil
		}
		generation := administrativeNotificationActivity.generation
		administrativeNotificationActivity.Unlock()
		settings, err := CurrentNotificationSettings()
		administrativeNotificationActivity.Lock()
		changed := administrativeNotificationActivity.generation != generation || DB != db
		if !changed && err == nil {
			events := make(map[string]bool)
			if settings.Enabled {
				for _, name := range strings.Split(settings.Events, ",") {
					if name != "" {
						events[name] = true
					}
				}
			}
			administrativeNotificationActivity.events = events
			administrativeNotificationActivity.settings = *settings
			administrativeNotificationActivity.db = db
			administrativeNotificationActivity.checked = time.Now()
		}
		administrativeNotificationActivity.Unlock()
		administrativeNotificationActivity.refresh.Unlock()
		if changed {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !settings.Enabled {
			return nil, nil
		}
		for _, name := range strings.Split(settings.Events, ",") {
			if name == event {
				return settings, nil
			}
		}
		return nil, nil
	}
}

func hasAdministrativeNotification(event string) bool {
	settings, err := notificationSettingsForEvent(event)
	return err != nil || settings != nil
}

type heartbeatPolicyEntry struct {
	userID uint
	groupID uint
	expires time.Time
	value *ResolvedPolicy
}
var heartbeatPolicyCache = struct {
	sync.Mutex
	db *gorm.DB
	generation uint64
	entries map[uint]heartbeatPolicyEntry
}{entries: make(map[uint]heartbeatPolicyEntry)}

// A bounded, incremental trim avoids invalidating every device when the cache
// reaches its high-water mark.
func trimHeartbeatPolicyEntries(now time.Time) {
	if len(heartbeatPolicyCache.entries) < 100000 {
		return
	}
	for id, entry := range heartbeatPolicyCache.entries {
		if !now.Before(entry.expires) {
			delete(heartbeatPolicyCache.entries, id)
		}
	}
	for id := range heartbeatPolicyCache.entries {
		if len(heartbeatPolicyCache.entries) <= 90000 {
			break
		}
		delete(heartbeatPolicyCache.entries, id)
	}
}

func resolveHeartbeatPolicy(peer *model.Peer) (*ResolvedPolicy, error) {
	if !policyMayApplyToPeer(peer) { return nil, nil }
	now := time.Now()
	heartbeatPolicyCache.Lock()
	if heartbeatPolicyCache.db != DB {
		heartbeatPolicyCache.db = DB
		heartbeatPolicyCache.entries = make(map[uint]heartbeatPolicyEntry)
	}
	entry, exists := heartbeatPolicyCache.entries[peer.RowId]
	if exists && entry.userID == peer.UserId && entry.groupID == peer.GroupId &&
		now.Before(entry.expires) {
		heartbeatPolicyCache.Unlock()
		return entry.value, nil
	}
	generation := heartbeatPolicyCache.generation
	heartbeatPolicyCache.Unlock()
	value, err := ResolveDevicePolicy(peer)
	if err != nil {
		return nil, err
	}
	heartbeatPolicyCache.Lock()
	if heartbeatPolicyCache.db == DB && heartbeatPolicyCache.generation == generation {
		trimHeartbeatPolicyEntries(time.Now())
		heartbeatPolicyCache.entries[peer.RowId] = heartbeatPolicyEntry{
			userID: peer.UserId, groupID: peer.GroupId,
			expires: time.Now().Add(optionalFeatureRefresh), value: value,
		}
	}
	heartbeatPolicyCache.Unlock()
	return value, nil
}
