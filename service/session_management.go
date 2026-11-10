package service

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sessionAliveSeconds int64 = 45
const disconnectExpiresSeconds int64 = 120
const maxConnectionsPerHeartbeat = 256


var idleSessionHeartbeats = struct {
	sync.Mutex
	db *gorm.DB
	until map[string]time.Time
}{until: make(map[string]time.Time)}

// A server with no persisted sessions checks two small indexed queries once
// per refresh interval and skips the per-device transaction entirely.
var persistedSessionActivity=struct{
	sync.Mutex
	refresh sync.Mutex
	db *gorm.DB
	generation uint64
	expires time.Time
	empty bool
}{}

func persistedSessionsEmpty() (bool, error) {
	for {
		db := DB
		if db == nil { return true, nil }
		now := time.Now()
		persistedSessionActivity.Lock()
		if persistedSessionActivity.db == db && now.Before(persistedSessionActivity.expires) {
			value := persistedSessionActivity.empty
			persistedSessionActivity.Unlock()
			return value, nil
		}
		persistedSessionActivity.Unlock()
		persistedSessionActivity.refresh.Lock()
		persistedSessionActivity.Lock()
		if persistedSessionActivity.db == db && time.Now().Before(persistedSessionActivity.expires) {
			value := persistedSessionActivity.empty
			persistedSessionActivity.Unlock()
			persistedSessionActivity.refresh.Unlock()
			return value, nil
		}
		generation := persistedSessionActivity.generation
		persistedSessionActivity.Unlock()
		var ids []uint
		err := db.Model(&model.DeviceActiveConnection{}).
			Where("last_seen_at >= ?", now.Unix()-sessionAliveSeconds).
			Limit(1).Pluck("id", &ids).Error
		empty := len(ids) == 0
		if err == nil && empty {
			ids = nil
			err = db.Model(&model.DeviceDisconnectRequest{}).
				Where("status IN ? AND expires_at > ?", []string{"pending", "delivered"}, now.Unix()).
				Limit(1).Pluck("id", &ids).Error
			empty = len(ids) == 0
		}
		persistedSessionActivity.Lock()
		changed := persistedSessionActivity.generation != generation || DB != db
		if !changed && err == nil {
			persistedSessionActivity.db = db
			persistedSessionActivity.empty = empty
			persistedSessionActivity.expires = time.Now().Add(optionalFeatureRefresh)
		}
		persistedSessionActivity.Unlock()
		persistedSessionActivity.refresh.Unlock()
		if changed { continue }
		return empty, err
	}
}

func invalidatePersistedSessions() {
	persistedSessionActivity.Lock()
	persistedSessionActivity.expires = time.Time{}
	persistedSessionActivity.generation++
	persistedSessionActivity.Unlock()
}

func markPersistedSessionActive() {
	persistedSessionActivity.Lock()
	wasEmpty := persistedSessionActivity.db != DB || persistedSessionActivity.empty ||
		persistedSessionActivity.expires.IsZero()
	persistedSessionActivity.db = DB
	persistedSessionActivity.empty = false
	persistedSessionActivity.generation++
	persistedSessionActivity.expires = time.Now().Add(optionalFeatureRefresh)
	persistedSessionActivity.Unlock()
	if wasEmpty { wakeSessionMaintenance() }
}

func knownIdleSession(id string, now time.Time) bool {
	idleSessionHeartbeats.Lock()
	defer idleSessionHeartbeats.Unlock()
	if idleSessionHeartbeats.db != DB {
		idleSessionHeartbeats.db = DB
		idleSessionHeartbeats.until = make(map[string]time.Time)
	}
	return now.Before(idleSessionHeartbeats.until[id])
}

func recordIdleSession(id string) {
	idleSessionHeartbeats.Lock()
	defer idleSessionHeartbeats.Unlock()
	if idleSessionHeartbeats.db != DB {
		idleSessionHeartbeats.db = DB
		idleSessionHeartbeats.until = make(map[string]time.Time)
	}
	if len(idleSessionHeartbeats.until) >= 100000 {
		now := time.Now()
		for id, until := range idleSessionHeartbeats.until {
			if !now.Before(until) { delete(idleSessionHeartbeats.until, id) }
		}
		for id := range idleSessionHeartbeats.until {
			if len(idleSessionHeartbeats.until) <= 90000 { break }
			delete(idleSessionHeartbeats.until, id)
		}
	}
	idleSessionHeartbeats.until[id] = time.Now().Add(optionalFeatureRefresh)
}

func invalidateIdleSession(id string) {
	idleSessionHeartbeats.Lock()
	defer idleSessionHeartbeats.Unlock()
	if idleSessionHeartbeats.db == DB {
		delete(idleSessionHeartbeats.until, id)
	}
}

// Only refresh unchanged active-session timestamps periodically. Disconnect
// requests are still checked on every active-session heartbeat.
const activeSessionWriteInterval = 12 * time.Second

type activeSessionSnapshot struct {
	keys []int32
	written time.Time
}
var activeSessionSnapshots = struct {
	sync.Mutex
	db *gorm.DB
	values map[string]activeSessionSnapshot
}{values: make(map[string]activeSessionSnapshot)}

func sessionSnapshotMatches(deviceID string, keys []int32, now time.Time) bool {
	activeSessionSnapshots.Lock()
	defer activeSessionSnapshots.Unlock()
	if activeSessionSnapshots.db != DB {
		activeSessionSnapshots.db = DB
		activeSessionSnapshots.values = make(map[string]activeSessionSnapshot)
	}
	old, ok := activeSessionSnapshots.values[deviceID]
	if !ok || now.Sub(old.written) >= activeSessionWriteInterval || len(old.keys) != len(keys) {
		return false
	}
	for i := range keys {
		if old.keys[i] != keys[i] { return false }
	}
	return true
}

func rememberSessionSnapshot(deviceID string, keys []int32) {
	activeSessionSnapshots.Lock()
	defer activeSessionSnapshots.Unlock()
	if activeSessionSnapshots.db != DB {
		activeSessionSnapshots.db = DB
		activeSessionSnapshots.values = make(map[string]activeSessionSnapshot)
	}
	if len(activeSessionSnapshots.values) >= 100000 {
		for id, entry := range activeSessionSnapshots.values {
			if time.Since(entry.written) > 2 * activeSessionWriteInterval {
				delete(activeSessionSnapshots.values, id)
			}
		}
		for id := range activeSessionSnapshots.values {
			if len(activeSessionSnapshots.values) <= 90000 { break }
			delete(activeSessionSnapshots.values, id)
		}
	}
	activeSessionSnapshots.values[deviceID] = activeSessionSnapshot{
		keys: append([]int32(nil), keys...), written: time.Now(),
	}
}

func forgetSessionSnapshot(deviceID string) {
	activeSessionSnapshots.Lock()
	delete(activeSessionSnapshots.values, deviceID)
	activeSessionSnapshots.Unlock()
}

// SynchronizeDeviceSessions follows the existing RustDesk heartbeat protocol.
func SynchronizeDeviceSessions(peer *model.Peer, connections []int32) ([]int32,error) {
	if peer==nil||peer.RowId==0{return nil,errors.New("device is not registered")}
	if len(connections)>maxConnectionsPerHeartbeat{
		return nil,errors.New("too many connections in heartbeat")
	}
	now:=time.Now().Unix()
	active:=make(map[int32]bool,len(connections))
	for _,id:=range connections {
		if id!=0 {active[id]=true}
	}
	keys:=make([]int32,0,len(active))
	records:=make([]model.DeviceActiveConnection,0,len(active))
	for id:=range active {
		keys=append(keys,id)
		records=append(records,model.DeviceActiveConnection{DeviceId:peer.Id,ConnId:id,LastSeenAt:now})
	}
	sort.Slice(keys,func(i,j int)bool{return keys[i]<keys[j]})
	if len(keys)==0 {
		empty,err:=persistedSessionsEmpty()
		if err==nil && empty {return nil,nil}
		if err==nil && knownIdleSession(peer.Id,time.Now()) {return nil,nil}
	}
	if len(keys)>0 {
		invalidateIdleSession(peer.Id)
		markPersistedSessionActive()
	}
	writeConnections := len(keys) > 0 && !sessionSnapshotMatches(peer.Id, keys, time.Now())
	instructions:=make([]int32,0)
	completed:=make([]model.DeviceDisconnectRequest,0)
	err:=DB.Transaction(func(tx *gorm.DB) error {
		if len(records)>0 && writeConnections {
			if err:=tx.Clauses(clause.OnConflict{
				Columns:[]clause.Column{{Name:"device_id"},{Name:"conn_id"}},
				DoUpdates:clause.AssignmentColumns([]string{"last_seen_at","updated_at"}),
			}).CreateInBatches(records,100).Error;err!=nil{return err}
			if err:=tx.Where("device_id = ? AND conn_id NOT IN ?",peer.Id,keys).
				Delete(&model.DeviceActiveConnection{}).Error;err!=nil{return err}
		}else if len(records) == 0 {
			if err:=tx.Where("device_id = ?",peer.Id).
				Delete(&model.DeviceActiveConnection{}).Error;err!=nil{return err}
		}
		var requests []model.DeviceDisconnectRequest
		if err:=tx.Where("device_id = ? AND status IN ?",peer.Id,[]string{"pending","delivered"}).
			Order("id ASC").Find(&requests).Error;err!=nil{return err}
		for _,request:=range requests{
			switch {
			case request.ExpiresAt<=now:
				if err:=tx.Model(&request).Update("status","expired").Error;err!=nil{return err}
			case !active[request.ConnId]:
				fields:=map[string]interface{}{"status":"expired"}
				if request.Status=="delivered" {
					fields["status"]="completed"
					fields["completed_at"]=now
					completed=append(completed,request)
				}
				if err:=tx.Model(&request).Updates(fields).Error;err!=nil{return err}
			default:
				instructions=append(instructions,request.ConnId)
				if err:=tx.Model(&request).Updates(map[string]interface{}{
					"status":"delivered","delivered_at":now,
				}).Error;err!=nil{return err}
			}
		}
		return nil
	})
	if err!=nil{return nil,err}
	if len(keys)==0 {
		recordIdleSession(peer.Id)
		forgetSessionSnapshot(peer.Id)
	} else if writeConnections {
		rememberSessionSnapshot(peer.Id, keys)
	}
	for _,finished:=range completed {
		reference:=fmt.Sprintf("%d:%s:%d",finished.Id,finished.DeviceId,finished.ConnId)
		NotifyEvent("session.ended",reference,"远程会话已完成结束请求："+reference)
		PublishPushEvent(PushEvent{
			Code:"session.ended",Reference:reference,DeviceID:finished.DeviceId,
			ConnID:int64(finished.ConnId),UserID:finished.RequestedBy,
			Detail:"远程会话已按管理请求结束",
		})
	}
	return instructions,nil
}

// RequestDeviceDisconnect only queues a request for a recently reported session.
func RequestDeviceDisconnect(deviceID string,connID int32,requester uint) error {
	if deviceID==""||len(deviceID)>253||connID==0||requester==0{
		return errors.New("valid device, connection and requester are required")
	}
	now:=time.Now().Unix()
	err := DB.Transaction(func(tx *gorm.DB) error {
		var connection model.DeviceActiveConnection
		err:=tx.Where("device_id = ? AND conn_id = ? AND last_seen_at >= ?",
			deviceID,connID,now-sessionAliveSeconds).First(&connection).Error
		if errors.Is(err,gorm.ErrRecordNotFound){return errors.New("connection is not active")}
		if err!=nil{return err}
		var existing model.DeviceDisconnectRequest
		err=tx.Where("device_id = ? AND conn_id = ? AND status IN ? AND expires_at > ?",
			deviceID,connID,[]string{"pending","delivered"},now).First(&existing).Error
		if err==nil{return nil}
		if !errors.Is(err,gorm.ErrRecordNotFound){return err}
		return tx.Create(&model.DeviceDisconnectRequest{
			DeviceId:deviceID,ConnId:connID,RequestedBy:requester,
			Status:"pending",ExpiresAt:now+disconnectExpiresSeconds,
		}).Error
	})
	if err==nil {
		invalidateIdleSession(deviceID)
		markPersistedSessionActive()
		wakeSessionMaintenance()
		NotifyEvent("session.disconnect_requested",fmt.Sprintf("%s:%d:%d",deviceID,connID,now/120),fmt.Sprintf("设备 %s 的连接 %d 已请求结束",deviceID,connID))
		PublishPushEvent(PushEvent{
			Code:"session.disconnect_requested",Reference:fmt.Sprintf("%s:%d:%d",deviceID,connID,now/120),
			DeviceID:deviceID,ConnID:int64(connID),UserID:requester,Detail:"管理员请求结束远程连接",
		})
	}
	return err
}

func ListActiveDeviceSessions(page,pageSize uint) ([]model.DeviceActiveConnection,int64,error) {
	var total int64
	var rows []model.DeviceActiveConnection
	query:=DB.Model(&model.DeviceActiveConnection{}).
		Where("last_seen_at >= ?",time.Now().Unix()-sessionAliveSeconds)
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	if err:=query.Order("last_seen_at DESC").Scopes(Paginate(page,pageSize)).
		Find(&rows).Error;err!=nil{return nil,0,err}
	return rows,total,nil
}

func ListDisconnectRequests(page,pageSize uint) ([]model.DeviceDisconnectRequest,int64,error) {
	var total int64
	var rows []model.DeviceDisconnectRequest
	query:=DB.Model(&model.DeviceDisconnectRequest{})
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	if err:=query.Order("id DESC").Scopes(Paginate(page,pageSize)).
		Find(&rows).Error;err!=nil{return nil,0,err}
	return rows,total,nil
}
