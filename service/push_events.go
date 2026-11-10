package service

import (
	"errors"
	"hash/fnv"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type PushEvent struct {
	Code      string
	Reference string
	DeviceID  string
	PeerRowID uint
	FromPeer  string
	SourceIP  string
	ConnID    int64
	Detail    string
	UserID    uint
	AccountUsername string
	When      time.Time
}

const pushEventShardCount = 4
const pushEventShardCapacity = 1024

// Route events for the same device to the same worker, preserving order
// without serializing unrelated devices behind a single database worker.
var pushEventShards = func() [pushEventShardCount]chan PushEvent {
	var shards [pushEventShardCount]chan PushEvent
	for i := range shards {
		shards[i] = make(chan PushEvent, pushEventShardCapacity)
	}
	return shards
}()

func pushEventShard(event PushEvent) int {
	if event.PeerRowID != 0 { return int(event.PeerRowID % pushEventShardCount) }
	key := event.DeviceID
	if key == "" { key = event.FromPeer }
	if key == "" { key = event.Reference }
	if key == "" { key = event.Code }
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return int(hash.Sum32() % pushEventShardCount)
}

// Coalesce frequent heartbeats in memory before entering the database worker.
// Presence remains accurate because the stored deadline is much longer than
// this interval. No timer or periodic map scan is needed.
const pushHeartbeatWriteInterval = 15 * time.Second
const maxHeartbeatCoalesceEntries = 100000

type recentPushHeartbeat struct {
	Last time.Time
	IP   string
}

var pushHeartbeatGate = struct {
	sync.Mutex
	Entries map[uint]recentPushHeartbeat
}{Entries: make(map[uint]recentPushHeartbeat)}

func shouldPublishPushHeartbeat(rowID uint, ip string, now time.Time) bool {
	pushHeartbeatGate.Lock()
	defer pushHeartbeatGate.Unlock()
	if previous, ok := pushHeartbeatGate.Entries[rowID]; ok &&
		now.Sub(previous.Last) < pushHeartbeatWriteInterval &&
		previous.IP == ip {
		return false
	}
	if len(pushHeartbeatGate.Entries) >= maxHeartbeatCoalesceEntries {
		pushHeartbeatGate.Entries = make(map[uint]recentPushHeartbeat)
	}
	pushHeartbeatGate.Entries[rowID] = recentPushHeartbeat{Last: now, IP: ip}
	return true
}

var pushPresenceWake = make(chan struct{}, 1)

func wakePushPresence() {
	select { case pushPresenceWake <- struct{}{}: default: }
}

// Event handlers never wait for outbound messages or database queue writes.
func PublishPushEvent(event PushEvent) {
	if event.When.IsZero() { event.When = time.Now() }
	worker := pushEventShards[pushEventShard(event)]
	select {
	case worker <- event:
	default:
		if Logger != nil { Logger.Warn("push event buffer is full") }
	}
}

func ReportPushHeartbeat(peer *model.Peer, ip string) {
	if peer == nil || peer.RowId == 0 { return }
	now := time.Now()
	if !shouldPublishPushHeartbeat(peer.RowId, ip, now) { return }
	PublishPushEvent(PushEvent{
		Code: "device.heartbeat", PeerRowID: peer.RowId,
		DeviceID: peer.Id, SourceIP: ip, When: now,
	})
}

func pushEventName(code string) string {
	for _, option := range pushEventCatalog {
		if option.Code == code { return option.Name }
	}
	return ""
}

func startPushEventWorker() {
	for i := range pushEventShards {
		events := pushEventShards[i]
		go func() {
			for event := range events {
				if err := processPushEvent(event); err != nil && Logger != nil {
					Logger.WithError(err).Warn("push event processing error")
				}
			}
		}()
	}
	go runPushPresenceScheduler()
}

// The first authenticated heartbeat and every offline-to-online change
// generate an event. Subsequent heartbeats only refresh the last-seen time.
func processPushHeartbeat(event PushEvent) error {
	if event.PeerRowID == 0 { return nil }
	now := event.When.Unix()
	if now == 0 { now = time.Now().Unix() }
	online := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var old model.PushPresence
		err := tx.Where("peer_row_id = ?", event.PeerRowID).Take(&old).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			online = true
			return tx.Create(&model.PushPresence{
				PeerRowId: event.PeerRowID, DeviceId: event.DeviceID,
				Online: true, LastSeenAt: now, LastIP: event.SourceIP,
			}).Error
		}
		if err != nil { return err }
		if old.LastSeenAt > now { now = old.LastSeenAt }
		online = !old.Online
		return tx.Model(&model.PushPresence{}).Where("peer_row_id = ?", event.PeerRowID).
			Updates(map[string]interface{}{
				"device_id": event.DeviceID, "online": true,
				"last_seen_at": now, "last_ip": event.SourceIP,
			}).Error
	})
	if err != nil { return err }
	if online {
		wakePushPresence()
		event.Code = "device.online"
		event.Reference = fmt.Sprintf("online:%d:%d", event.PeerRowID, now)
		return processPushEvent(event)
	}
	return nil
}

// nextPushPresenceDeadline is evaluated when a heartbeat is accepted,
// when a deadline expires, and once on startup.
func nextPushPresenceDeadline() (time.Duration, bool, error) {
	var row struct {
		Seen *int64 `gorm:"column:seen"`
	}
	if err := DB.Model(&model.PushPresence{}).
		Select("MIN(last_seen_at) AS seen").
		Where("online = ?", true).
		Scan(&row).Error; err != nil {
		return 0, false, err
	}
	if row.Seen == nil { return 0, false, nil }
	// The strict expiry check uses <, so the deadline includes one extra second.
	delay := time.Until(time.Unix(*row.Seen+121, 0))
	if delay < 0 { delay = 0 }
	return delay, true, nil
}

func runPushPresenceScheduler() {
	for {
		delay, hasDeadline, err := nextPushPresenceDeadline()
		if err != nil {
			if Logger != nil { Logger.WithError(err).Warn("presence timer error") }
			waitForWork(pushPresenceWake, 10*time.Second, true)
			continue
		}
		if hasDeadline && delay <= 0 {
			if err := ProcessPushPresenceTimeouts(100); err != nil {
				if Logger != nil { Logger.WithError(err).Warn("presence expiration error") }
				waitForWork(pushPresenceWake, 10*time.Second, true)
			}
			continue
		}
		waitForWork(pushPresenceWake, delay, hasDeadline)
	}
}

// Devices are declared offline after 120 seconds without an authenticated
// heartbeat. The conditional update prevents duplicate offline events.
func ProcessPushPresenceTimeouts(limit int) error {
	if DB == nil { return nil }
	if limit < 1 || limit > 500 { limit = 100 }
	cutoff := time.Now().Add(-120 * time.Second).Unix()
	var rows []model.PushPresence
	if err := DB.Where("online = ? AND last_seen_at < ?", true, cutoff).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	for _, presence := range rows {
		change := DB.Model(&model.PushPresence{}).Where(
			"peer_row_id = ? AND online = ? AND last_seen_at = ?",
			presence.PeerRowId, true, presence.LastSeenAt,
		).Update("online", false)
		if change.Error != nil { return change.Error }
		if change.RowsAffected != 1 { continue }
		event := PushEvent{
			Code: "device.offline", PeerRowID: presence.PeerRowId,
			DeviceID: presence.DeviceId, SourceIP: presence.LastIP,
			Reference: fmt.Sprintf("offline:%d:%d", presence.PeerRowId, presence.LastSeenAt),
			Detail: "最后在线：" + time.Unix(presence.LastSeenAt, 0).Local().Format(time.RFC3339),
			When: time.Now(),
		}
		if err := processPushEvent(event); err != nil { return err }
	}
	return nil
}

func processPushEvent(event PushEvent) error {
	if event.Code == "device.heartbeat" { return processPushHeartbeat(event) }
	if pushEventName(event.Code) == "" || DB == nil { return nil }
	if event.When.IsZero() { event.When = time.Now() }
	participants := make([]uint, 0, 3)
	if event.UserID == 0 && event.AccountUsername != "" {
		var attempted model.User
		if DB.Select("id").Where("username = ?", event.AccountUsername).Take(&attempted).Error == nil {
			event.UserID = attempted.Id
		}
	}
	if event.UserID != 0 { participants = append(participants, event.UserID) }
	var peer model.Peer
	if event.PeerRowID != 0 {
		_ = DB.Where("row_id = ?", event.PeerRowID).Take(&peer).Error
	} else if event.DeviceID != "" {
		_ = DB.Where("id = ?", event.DeviceID).Order("last_online_time DESC, row_id DESC").Take(&peer).Error
	}
	if peer.UserId != 0 { participants = append(participants, peer.UserId) }
	if event.FromPeer != "" {
		var source model.Peer
		_ = DB.Select("user_id").Where("id = ?", event.FromPeer).Order("last_online_time DESC, row_id DESC").Take(&source).Error
		if source.UserId != 0 { participants = append(participants, source.UserId) }
	}
	if strings.HasPrefix(event.Code, "connection.") && event.ConnID != 0 {
		var audit model.AuditConn
		if DB.Where("peer_id = ? AND conn_id = ?", event.DeviceID, event.ConnID).
			Order("id DESC").Take(&audit).Error == nil {
			if audit.FromPeer != "" { event.FromPeer = audit.FromPeer }
			if audit.Ip != "" { event.SourceIP = audit.Ip }
			if event.Reference == "" { event.Reference = fmt.Sprintf("connection:%d:%s", audit.Id, event.Code) }
			if audit.FromUserId != 0 { participants = append(participants, audit.FromUserId) }
			if audit.ToUserId != 0 { participants = append(participants, audit.ToUserId) }
		}
	}
	if len(participants) == 0 { return nil }
	return QueuePushForUsers(event, participants)
}

func pushText(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '\u2028' || r == '\u2029' { return ' ' }
		return r
	}, s))
}

func pushShort(s string, n int) string {
	r := []rune(pushText(s))
	if len(r) <= n { return string(r) }
	return string(r[:n-1]) + "…"
}

func pushLabel(peer model.Peer, id string) string {
	name := peer.Alias
	if name == "" { name = peer.Hostname }
	if name == "" { name = id }
	name = pushShort(name, 55)
	if id == "" || name == id { return name }
	return name + " (" + pushShort(id, 70) + ")"
}

func formatPushMessage(event PushEvent) (string, string) {
	name := pushEventName(event.Code)
	if name == "" { return "", "" }
	var target, source model.Peer
	if event.DeviceID != "" {
		_ = DB.Where("id = ?", event.DeviceID).Order("last_online_time DESC, row_id DESC").Take(&target).Error
	}
	if event.FromPeer != "" {
		_ = DB.Where("id = ?", event.FromPeer).Order("last_online_time DESC, row_id DESC").Take(&source).Error
	}
	targetLabel := pushLabel(target, event.DeviceID)
	sourceLabel := pushLabel(source, event.FromPeer)
	title := "[RustDesk · " + name + "]"
	if event.AccountUsername != "" { title += " " + pushShort(event.AccountUsername, 65) }
	if targetLabel != "" { title += " " + targetLabel }
	if sourceLabel != "" && strings.HasPrefix(event.Code, "connection.") {
		title += " ← " + sourceLabel
	}
	if event.SourceIP != "" && len([]rune(title)) < 160 {
		title += " · " + pushShort(event.SourceIP, 42) + "（" + GeoIPDescription(event.SourceIP) + "）"
	}
	lines := make([]string, 0, 20)
	appendField := func(key, value string) {
		value = pushText(value)
		if value != "" { lines = append(lines, key+"："+pushShort(value, 2000)) }
	}
	appendField("事件", name)
	appendField("设备", targetLabel)
	if event.FromPeer != "" { appendField("连接方设备", sourceLabel) }
	ip := pushText(event.SourceIP)
	if ip != "" { appendField("来源 IP", ip+"（"+GeoIPDescription(ip)+"）") }
	if event.UserID != 0 {
		var user model.User
		if DB.Select("id,username").Where("id = ?", event.UserID).Take(&user).Error == nil {
			appendField("关联账户", fmt.Sprintf("%s (ID %d)", user.Username, user.Id))
		}
	}
	if target.UserId != 0 {
		var user model.User
		if DB.Select("id,username").Where("id = ?", target.UserId).Take(&user).Error == nil {
			appendField("设备所有者", fmt.Sprintf("%s (ID %d)", user.Username, user.Id))
		}
	}
	if source.UserId != 0 {
		var user model.User
		if DB.Select("id,username").Where("id = ?", source.UserId).Take(&user).Error == nil {
			appendField("连接方账户", fmt.Sprintf("%s (ID %d)", user.Username, user.Id))
		}
	}
	appendField("发生时间", event.When.Local().Format("2006-01-02 15:04:05 -07:00"))
	appendField("设备主机名", target.Hostname)
	appendField("设备登录用户", target.Username)
	appendField("操作系统", target.Os)
	appendField("客户端版本", target.Version)
	if target.LastOnlineIp != "" && target.LastOnlineIp != ip {
		appendField("设备最近 IP", target.LastOnlineIp+"（"+GeoIPDescription(target.LastOnlineIp)+"）")
	}
	appendField("连接方主机名", source.Hostname)
	appendField("连接方系统用户", source.Username)
	if event.ConnID != 0 { appendField("连接编号", strconv.FormatInt(event.ConnID, 10)) }
	if strings.HasPrefix(event.Code, "connection.") && event.ConnID != 0 {
		var audit model.AuditConn
		if DB.Where("peer_id = ? AND conn_id = ?", event.DeviceID, event.ConnID).
			Order("id DESC").Take(&audit).Error == nil {
			appendField("连接方登录名称", audit.FromName)
			appendField("会话编号", audit.SessionId)
			appendField("连接关联编号", audit.ConnAuditRef)
			if audit.PrimaryAuth != 0 { appendField("认证方式编号", strconv.Itoa(audit.PrimaryAuth)) }
			if audit.TwoFactor != 0 { appendField("双重认证编号", strconv.Itoa(audit.TwoFactor)) }
			if audit.CloseTime != 0 && event.Code == "connection.ended" {
				appendField("连接结束时间", time.Unix(audit.CloseTime, 0).Local().Format(time.RFC3339))
			}
		}
	}
	appendField("事件详情", event.Detail)
	body := strings.Join(lines, "\n")
	if len(body) > 12000 {
		r := []rune(body)
		if len(r) > 2500 { r = r[:2500] }
		body = string(r)
	}
	return pushShort(title, 245), body
}
