package service

import (
	"strings"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const optionalFeatureRefresh = 30*time.Second

var policyActivity = struct {
	sync.Mutex
	db *gorm.DB
	checked time.Time
	assigned bool
}{}

func invalidatePolicyActivity(){
	policyActivity.Lock()
	policyActivity.checked=time.Time{}
	policyActivity.Unlock()
	heartbeatPolicyCache.Lock()
	heartbeatPolicyCache.generation++
	heartbeatPolicyCache.entries=make(map[uint]heartbeatPolicyEntry)
	heartbeatPolicyCache.Unlock()
}

func devicePolicyAssignmentsPresent()(bool,error){
	if DB==nil{return false,nil}
	policyActivity.Lock()
	defer policyActivity.Unlock()
	if policyActivity.db==DB && !policyActivity.checked.IsZero() &&
		time.Since(policyActivity.checked)<optionalFeatureRefresh {
		return policyActivity.assigned,nil
	}
	policyActivity.db=DB
	policyActivity.checked=time.Time{}
	for _,table:=range []interface{}{
		&model.DevicePolicyAssignment{},&model.DevicePolicyUserAssignment{},
		&model.DevicePolicyGroupAssignment{},
	}{
		if !DB.Migrator().HasTable(table){continue}
		var ids []uint
		if err:=DB.Model(table).Limit(1).Pluck("id",&ids).Error;err!=nil{
			return true,err
		}
		if len(ids)>0{
			policyActivity.assigned=true
			policyActivity.checked=time.Now()
			return true,nil
		}
	}
	policyActivity.assigned=false
	policyActivity.checked=time.Now()
	return false,nil
}

var pushActivity = struct {
	sync.Mutex
	db *gorm.DB
	checked time.Time
	events map[string]bool
}{}

func invalidatePushActivity(){
	pushActivity.Lock()
	pushActivity.checked=time.Time{}
	pushActivity.Unlock()
}

// Skip database lookups for notifications without any subscribed recipients.
// Heartbeats still maintain the device presence state.
func hasPushSubscription(code string) bool {
	if DB==nil{return false}
	pushActivity.Lock()
	defer pushActivity.Unlock()
	if pushActivity.db!=DB || pushActivity.checked.IsZero() ||
		time.Since(pushActivity.checked)>=optionalFeatureRefresh {
		events:=make(map[string]bool)
		if DB.Migrator().HasTable(&model.PushChannel{}) {
			var channels []struct{Events string}
			if err:=DB.Model(&model.PushChannel{}).Select("events").
				Where("enabled = ?",true).Find(&channels).Error;err!=nil{
				return true // Preserve notifications if the database is unavailable.
			}
			for _,channel:=range channels {
				for _,event:=range strings.Split(channel.Events,","){
					if event!=""{events[event]=true}
				}
			}
		}
		pushActivity.db=DB
		pushActivity.events=events
		pushActivity.checked=time.Now()
	}
	return pushActivity.events[code]
}

var administrativeNotificationActivity = struct {
	sync.Mutex
	db *gorm.DB
	checked time.Time
	events map[string]bool
}{}

func invalidateAdministrativeNotificationActivity(){
	administrativeNotificationActivity.Lock()
	administrativeNotificationActivity.checked=time.Time{}
	administrativeNotificationActivity.Unlock()
}

func hasAdministrativeNotification(event string) bool {
	if DB==nil{return false}
	administrativeNotificationActivity.Lock()
	defer administrativeNotificationActivity.Unlock()
	if administrativeNotificationActivity.db!=DB ||
		administrativeNotificationActivity.checked.IsZero() ||
		time.Since(administrativeNotificationActivity.checked)>=optionalFeatureRefresh {
		settings,err:=CurrentNotificationSettings()
		if err!=nil{return true} // Do not silently discard events on lookup failure.
		events:=make(map[string]bool)
		if settings.Enabled{
			for _,name:=range strings.Split(settings.Events,","){
				if name!=""{events[name]=true}
			}
		}
		administrativeNotificationActivity.events=events
		administrativeNotificationActivity.db=DB
		administrativeNotificationActivity.checked=time.Now()
	}
	return administrativeNotificationActivity.events[event]
}

// A device-specific snapshot avoids repeatedly resolving three assignment
// levels when only a small number of devices use advanced policies.
type heartbeatPolicyEntry struct {
	userID uint
	groupID uint
	expires time.Time
	value *ResolvedPolicy
}
var heartbeatPolicyCache=struct{
	sync.Mutex
	db *gorm.DB
	generation uint64
	entries map[uint]heartbeatPolicyEntry
}{entries:make(map[uint]heartbeatPolicyEntry)}

func resolveHeartbeatPolicy(peer *model.Peer)(*ResolvedPolicy,error){
	now:=time.Now()
	heartbeatPolicyCache.Lock()
	if heartbeatPolicyCache.db!=DB {
		heartbeatPolicyCache.db=DB
		heartbeatPolicyCache.entries=make(map[uint]heartbeatPolicyEntry)
	}
	entry,exists:=heartbeatPolicyCache.entries[peer.RowId]
	if exists && entry.userID==peer.UserId && entry.groupID==peer.GroupId &&
		now.Before(entry.expires) {
		heartbeatPolicyCache.Unlock()
		return entry.value,nil
	}
	generation:=heartbeatPolicyCache.generation
	heartbeatPolicyCache.Unlock()
	value,err:=ResolveDevicePolicy(peer)
	if err!=nil{return nil,err}
	heartbeatPolicyCache.Lock()
	if heartbeatPolicyCache.db==DB && heartbeatPolicyCache.generation==generation {
		if len(heartbeatPolicyCache.entries)>=100000 {
			heartbeatPolicyCache.entries=make(map[uint]heartbeatPolicyEntry)
		}
		heartbeatPolicyCache.entries[peer.RowId]=heartbeatPolicyEntry{
			userID:peer.UserId,groupID:peer.GroupId,
			expires:time.Now().Add(optionalFeatureRefresh),value:value,
		}
	}
	heartbeatPolicyCache.Unlock()
	return value,nil
}
