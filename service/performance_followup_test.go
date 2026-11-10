package service

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func TestInactiveSessionRowsAreRemovedAndDoNotActivateHeartbeats(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceActiveConnection{}, &model.DeviceDisconnectRequest{}); err != nil { t.Fatal(err) }
	now := time.Now()
	old := &model.DeviceActiveConnection{DeviceId: "stale-device", ConnId: 10, LastSeenAt: now.Add(-2*time.Minute).Unix()}
	if err := DB.Create(old).Error; err != nil { t.Fatal(err) }
	expired := &model.DeviceDisconnectRequest{DeviceId: "stale-device", ConnId: 10,
		RequestedBy: 1, Status: "pending", ExpiresAt: now.Add(-time.Minute).Unix()}
	if err := DB.Create(expired).Error; err != nil { t.Fatal(err) }
	if empty, err := persistedSessionsEmpty(); err != nil || !empty {
		t.Fatalf("expired rows activated idle sessions: %v %v", empty, err)
	}
	changed, err := PruneExpiredSessions(now)
	if err != nil || changed != 2 { t.Fatalf("session cleanup: %d %v", changed, err) }
	var count int64
	if err := DB.Model(&model.DeviceActiveConnection{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("old connection was retained: %d %v", count, err)
	}
	var saved model.DeviceDisconnectRequest
	if err := DB.First(&saved, expired.Id).Error; err != nil || saved.Status != "expired" {
		t.Fatalf("expired request was not updated: %+v %v", saved, err)
	}
	if delay, exists, err := nextSessionMaintenanceDelay(); err != nil || exists || delay != 0 {
		t.Fatalf("idle session worker has an unnecessary deadline: %v %v %v", delay, exists, err)
	}
}

func TestUnchangedActiveConnectionsAvoidRepeatedWrites(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceActiveConnection{}, &model.DeviceDisconnectRequest{}); err != nil { t.Fatal(err) }
	peer := &model.Peer{Id: "repeat-session-device"}
	if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
	creates := 0
	key := "perf:repeat_connections"
	if err := DB.Callback().Create().Before("gorm:create").Register(key, func(*gorm.DB) { creates++ }); err != nil { t.Fatal(err) }
	defer DB.Callback().Create().Remove(key)
	if _, err := SynchronizeDeviceSessions(peer, []int32{21, 22}); err != nil { t.Fatal(err) }
	first := creates
	for i := 0; i < 8; i++ {
		if _, err := SynchronizeDeviceSessions(peer, []int32{22, 21}); err != nil { t.Fatal(err) }
	}
	if first == 0 || creates != first {
		t.Fatalf("repeated connections caused unnecessary inserts: first=%d total=%d", first, creates)
	}
	if _, err := SynchronizeDeviceSessions(peer, []int32{21}); err != nil { t.Fatal(err) }
	if creates <= first { t.Fatal("changed connection set was not saved") }
}

func TestAccessTokenJoinPreservesImmediateRevocation(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.UserToken{}); err != nil { t.Fatal(err) }
	user := &model.User{Username: "token-join-user", Status: model.COMMON_STATUS_ENABLE}
	if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
	token := &model.UserToken{UserId: user.Id, Token: "token-join-example",
		DeviceId: "device-1", DeviceUuid: "uuid-1", ExpiredAt: time.Now().Add(time.Hour).Unix()}
	if err := DB.Create(token).Error; err != nil { t.Fatal(err) }
	queries := 0
	key := "perf:auth_query"
	if err := DB.Callback().Query().Before("gorm:query").Register(key, func(*gorm.DB) { queries++ }); err != nil { t.Fatal(err) }
	actual, access := (&UserService{}).InfoByAccessToken(token.Token)
	DB.Callback().Query().Remove(key)
	if actual.Id != user.Id || access.Id != token.Id ||
		access.DeviceId != token.DeviceId || access.DeviceUuid != token.DeviceUuid {
		t.Fatalf("joined session differs: user=%+v token=%+v", actual, access)
	}
	if queries != 1 { t.Fatalf("access token used %d SQL reads", queries) }
	if err := DB.Delete(token).Error; err != nil { t.Fatal(err) }
	actual, _ = (&UserService{}).InfoByAccessToken(token.Token)
	if actual.Id != 0 { t.Fatal("revoked token remained usable") }
}

func TestSharedCollectionRulesMatchIndividualEvaluation(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AddressBookCollection{}, &model.AddressBookCollectionRule{}); err != nil {
		t.Fatal(err)
	}
	users := seedGroups(t)
	member, owner := users["member"], users["other"]
	var collections []*model.AddressBookCollection
	for i := 0; i < 24; i++ {
		c := &model.AddressBookCollection{UserId: owner.Id, Name: fmt.Sprintf("shared-%d", i)}
		if err := DB.Create(c).Error; err != nil { t.Fatal(err) }
		collections = append(collections, c)
		switch i % 4 {
		case 0:
			if err := DB.Create(&model.AddressBookCollectionRule{
				UserId: owner.Id, CollectionId: c.Id, Type: model.ShareAddressBookRuleTypePersonal,
				ToId: member.Id, Rule: model.ShareAddressBookRuleRuleReadWrite,
			}).Error; err != nil { t.Fatal(err) }
		case 1:
			if err := DB.Create(&model.AddressBookCollectionRule{
				UserId: owner.Id, CollectionId: c.Id, Type: model.ShareAddressBookRuleTypeGroup,
				ToId: member.GroupId, Rule: model.ShareAddressBookRuleRuleRead,
			}).Error; err != nil { t.Fatal(err) }
		case 2:
			if err := DB.Create(&model.AddressBookCollectionRule{
				UserId: owner.Id, CollectionId: c.Id, Type: model.ShareAddressBookRuleTypeAll,
				ToId: 0, Rule: model.ShareAddressBookRuleRuleRead,
			}).Error; err != nil { t.Fatal(err) }
		}
	}
	queries := 0
	key := "perf:shared_rules"
	if err := DB.Callback().Query().Before("gorm:query").Register(key, func(*gorm.DB) { queries++ }); err != nil { t.Fatal(err) }
	batch, err := (&AddressBookService{}).BatchUserMaxRules(member, collections)
	DB.Callback().Query().Remove(key)
	if err != nil || queries > 3 { t.Fatalf("batch permissions: queries=%d err=%v", queries, err) }
	for _, collection := range collections {
		individual := (&AddressBookService{}).UserMaxRule(member, owner.Id, collection.Id)
		if batch[collection.Id] != individual {
			t.Fatalf("collection %d differs: batch=%d individual=%d", collection.Id, batch[collection.Id], individual)
		}
	}
}

func TestPolicyInvalidationDoesNotWaitForSlowDatabaseRead(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DevicePolicyAssignment{},
		&model.DevicePolicyUserAssignment{}, &model.DevicePolicyGroupAssignment{}); err != nil {
		t.Fatal(err)
	}
	invalidatePolicyActivity()
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	key := "perf:slow_policy_refresh"
	if err := DB.Callback().Query().Before("gorm:query").Register(key, func(*gorm.DB) {
		once.Do(func() { close(started); <-release })
	}); err != nil { t.Fatal(err) }
	defer DB.Callback().Query().Remove(key)
	readDone := make(chan struct{})
	go func() {
		_, _ = devicePolicyAssignmentsPresent()
		close(readDone)
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		close(release)
		t.Fatal("policy refresh did not perform a database read")
	}
	invalidated := make(chan struct{})
	go func() { invalidatePolicyActivity(); close(invalidated) }()
	select {
	case <-invalidated:
	case <-time.After(300 * time.Millisecond):
		close(release)
		t.Fatal("cache invalidation was blocked by a slow SQL query")
	}
	close(release)
	select {
	case <-readDone:
	case <-time.After(3 * time.Second):
		t.Fatal("policy refresh did not complete")
	}
}

func TestAcceptedPushEventSurvivesWorkerRestart(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	user := &model.User{Username: "persisted-push-user", Status: model.COMMON_STATUS_ENABLE}
	if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
	peer := &model.Peer{Id: "persisted-push-device", UserId: user.Id}
	if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
	if _, err := SavePushChannel(user.Id, PushChannelInput{
		Kind: "messagepusher", Name: "persistent", Enabled: true,
		APIKey: "mpk_persisted_test", BaseURL: "https://push.example.test",
		Events: []string{"device.registered"},
	}); err != nil { t.Fatal(err) }
	ev := PushEvent{Code: "device.registered", Reference: "persistent-event-42",
		DeviceID: peer.Id, PeerRowID: peer.RowId, UserID: user.Id}
	PublishPushEvent(ev)
	var count int64
	if err := DB.Model(&model.PushEventOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("accepted event was not stored: %d %v", count, err)
	}
	if processed, err := ProcessPushEventOutbox(20); err != nil || processed != 1 {
		t.Fatalf("outbox recovery failed: %d %v", processed, err)
	}
	if err := DB.Model(&model.PushEventOutbox{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("completed outbox entry remains: %d %v", count, err)
	}
	if err := persistPushEvent(ev); err != nil { t.Fatal(err) }
	if _, err := ProcessPushEventOutbox(20); err != nil { t.Fatal(err) }
	if err := DB.Model(&model.PushDelivery{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("replayed event duplicated delivery: %d %v", count, err)
	}
}

func TestAcceptedAdministrativeNotificationSurvivesRestart(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.NotificationSetting{}, &model.NotificationDelivery{},
		&model.NotificationEventOutbox{}); err != nil { t.Fatal(err) }
	if err := SaveNotificationSettings(&model.NotificationSetting{
		Enabled: true, Recipients: "admin@example.test", Events: "deployment.changed",
	}); err != nil { t.Fatal(err) }
	NotifyEvent("deployment.changed", "durable-incident", "deployment changed")
	NotifyEvent("deployment.changed", "durable-incident", "same event")
	var count int64
	if err := DB.Model(&model.NotificationEventOutbox{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("accepted alerts were not saved: %d %v", count, err)
	}
	if n, err := ProcessNotificationEventOutbox(20); err != nil || n != 2 {
		t.Fatalf("notification recovery: %d %v", n, err)
	}
	if err := DB.Model(&model.NotificationDelivery{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("notification replay was not deduplicated: %d %v", count, err)
	}
}

// TestDevicePresenceAuditQueryPlan checks the existing audit indexes on a
// representative history size before adding indexes that also slow inserts.
func TestDevicePresenceAuditQueryPlan(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	records := make([]model.AuditConn, 4000)
	for i := range records {
		records[i] = model.AuditConn{
			FromPeer: fmt.Sprintf("source-%d", i%100),
			PeerId: fmt.Sprintf("device-%d", i%100),
			FromUserId: uint(1+i%100),
			ToUserId: uint(101+i%100),
		}
	}
	if err := DB.CreateInBatches(&records, 100).Error; err != nil { t.Fatal(err) }
	if err := DB.Exec("ANALYZE").Error; err != nil { t.Fatal(err) }
	var plan []struct{ Detail string `gorm:"column:detail"` }
	err := DB.Raw(
		"EXPLAIN QUERY PLAN SELECT DISTINCT from_peer AS id FROM audit_conns "+
			"WHERE (from_user_id = ? OR to_user_id = ?) AND from_peer IN ? "+
			"UNION SELECT DISTINCT peer_id AS id FROM audit_conns "+
			"WHERE (from_user_id = ? OR to_user_id = ?) AND peer_id IN ?",
		uint(2), uint(2), []string{"source-1"}, uint(2), uint(2), []string{"device-1"},
	).Scan(&plan).Error
	if err != nil { t.Fatal(err) }
	usesIndex := false
	for _, row := range plan {
		t.Logf("audit query plan: %s", row.Detail)
		if strings.Contains(strings.ToUpper(row.Detail), "USING INDEX") ||
			strings.Contains(strings.ToUpper(row.Detail), "USING COVERING INDEX") {
			usesIndex = true
		}
	}
	if len(plan) == 0 || !usesIndex {
		t.Fatalf("audit lookup did not use any existing index: %+v", plan)
	}
}

// The number of channel lookup queries remains constant as recipients grow.
func TestPushChannelLookupIsBatched(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	users := make([]model.User, 24)
	for i := range users {
		users[i] = model.User{Username: fmt.Sprintf("bulk-recipient-%02d", i),
			Status: model.COMMON_STATUS_ENABLE}
	}
	if err := DB.CreateInBatches(&users, 100).Error; err != nil { t.Fatal(err) }
	channels := make([]model.PushChannel, 0, len(users))
	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.Id)
		channels = append(channels, model.PushChannel{
			UserId: user.Id, Kind: "messagepusher", Name: "batch",
			Enabled: true, Events: "device.registered",
		})
	}
	if err := DB.CreateInBatches(&channels, 100).Error; err != nil { t.Fatal(err) }
	device := &model.Peer{Id: "batched-device", UserId: users[0].Id}
	if err := DB.Create(device).Error; err != nil { t.Fatal(err) }
	lookups := 0
	key := "perf:bulk_push_channels"
	if err := DB.Callback().Query().Before("gorm:query").Register(key, func(tx *gorm.DB) {
		if tx.Statement.Table == "push_channels" { lookups++ }
	}); err != nil { t.Fatal(err) }
	err := QueuePushForUsers(PushEvent{
		Code: "device.registered", Reference: "bulk-recipient-event",
		DeviceID: device.Id, PeerRowID: device.RowId, When: time.Now(),
	}, userIDs)
	DB.Callback().Query().Remove(key)
	if err != nil || lookups != 1 {
		t.Fatalf("push channel lookup count %d with %d users: %v", lookups, len(users), err)
	}
	var deliveries int64
	if err := DB.Model(&model.PushDelivery{}).Count(&deliveries).Error; err != nil ||
		deliveries != int64(len(users)) {
		t.Fatalf("push recipients were not queued: %d %v", deliveries, err)
	}
}

// A presence transition and its alert are committed together. A later
// delivery-table failure leaves an event for recovery after restart.
func TestPresenceTransitionRetainsEventUntilDeliveryIsRecorded(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	user := &model.User{Username: "durable-presence-user", Status: model.COMMON_STATUS_ENABLE}
	if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
	peer := &model.Peer{Id: "durable-presence-device", UserId: user.Id}
	if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
	if _, err := SavePushChannel(user.Id, PushChannelInput{
		Kind: "messagepusher", Name: "online alert", Enabled: true,
		APIKey: "mpk_durable_presence", BaseURL: "https://push.example.test",
		Events: []string{"device.online"},
	}); err != nil { t.Fatal(err) }
	if err := DB.Migrator().DropTable(&model.PushDelivery{}); err != nil { t.Fatal(err) }
	event := PushEvent{
		Code: "device.heartbeat", DeviceID: peer.Id,
		PeerRowID: peer.RowId, When: time.Now(),
	}
	if err := processPushEvent(event); err != nil { t.Fatal(err) }
	if _, err := ProcessPushEventOutbox(20); err == nil {
		t.Fatal("missing delivery table should interrupt notification processing")
	}
	var count int64
	if err := DB.Model(&model.PushEventOutbox{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("accepted online event was lost: %d %v", count, err)
	}
	if err := DB.AutoMigrate(&model.PushDelivery{}); err != nil { t.Fatal(err) }
	if n, err := ProcessPushEventOutbox(20); err != nil || n != 1 {
		t.Fatalf("online event recovery failed: %d %v", n, err)
	}
	if err := DB.Model(&model.PushDelivery{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("online event was not recovered: %d %v", count, err)
	}
}

func TestActiveSessionsWithoutPendingDisconnectSkipDatabaseOperations(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceActiveConnection{}, &model.DeviceDisconnectRequest{}); err != nil { t.Fatal(err) }
	peer := &model.Peer{Id: "active-session-no-disconnect"}
	if err := DB.Create(peer).Error; err != nil { t.Fatal(err) }
	if _, err := SynchronizeDeviceSessions(peer, []int32{52}); err != nil { t.Fatal(err) }
	if _, err := deviceHasPendingDisconnectRequest(peer.Id); err != nil { t.Fatal(err) }
	reads, updates, creates := 0, 0, 0
	if err := DB.Callback().Query().Before("gorm:query").Register("perf:pending_query", func(*gorm.DB) { reads++ }); err != nil { t.Fatal(err) }
	if err := DB.Callback().Update().Before("gorm:update").Register("perf:pending_update", func(*gorm.DB) { updates++ }); err != nil { t.Fatal(err) }
	if err := DB.Callback().Create().Before("gorm:create").Register("perf:pending_create", func(*gorm.DB) { creates++ }); err != nil { t.Fatal(err) }
	for i := 0; i < 20; i++ {
		if commands, err := SynchronizeDeviceSessions(peer, []int32{52}); err != nil || len(commands) > 0 {
			t.Fatalf("unchanged active session: %v %v", commands, err)
		}
	}
	DB.Callback().Query().Remove("perf:pending_query")
	DB.Callback().Update().Remove("perf:pending_update")
	DB.Callback().Create().Remove("perf:pending_create")
	if reads != 0 || updates != 0 || creates != 0 {
		t.Fatalf("no pending request caused SQL: read=%d update=%d create=%d", reads, updates, creates)
	}
	if err := RequestDeviceDisconnect(peer.Id, 52, 1); err != nil { t.Fatal(err) }
	commands, err := SynchronizeDeviceSessions(peer, []int32{52})
	if err != nil || len(commands) != 1 || commands[0] != 52 {
		t.Fatalf("pending disconnect was delayed: %v %v", commands, err)
	}
}
