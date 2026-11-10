package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestPushSchedulerOnlySchedulesPersistedDeadlines(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()

	delay, hasWork, err := scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60)
	if err != nil || hasWork {
		t.Fatalf("empty queue has a deadline: %v %v %v", delay, hasWork, err)
	}
	at := time.Now().Add(45*time.Second).Unix()
	row := &model.PushDelivery{
		ChannelId: 1, UserId: 1, EventType: "device.online",
		EventKey: eventHash("deadline"), Status: "pending", NextAttemptAt: at,
		Title:"device online", Body:"test",
	}
	if err := DB.Create(row).Error; err != nil { t.Fatal(err) }
	delay, hasWork, err = scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60)
	if err != nil || !hasWork || delay < 42*time.Second || delay > 46*time.Second {
		t.Fatalf("incorrect pending deadline: %v %v %v", delay, hasWork, err)
	}
	if err := DB.Model(row).Updates(map[string]interface{}{
		"status": "paused", "next_attempt_at": 0,
	}).Error; err != nil { t.Fatal(err) }
	if _, hasWork, err = scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60); err != nil || hasWork {
		t.Fatalf("paused task activated a deadline: %v %v", hasWork, err)
	}
	if err := DB.Model(row).Updates(map[string]interface{}{
		"status":"retry","next_attempt_at":time.Now().Add(-time.Second).Unix(),
	}).Error; err != nil { t.Fatal(err) }
	if delay, hasWork, err = scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60); err != nil || !hasWork || delay > time.Second {
		t.Fatalf("due retry did not wake immediately: %v %v %v", delay, hasWork, err)
	}
	if err := DB.Model(row).Updates(map[string]interface{}{
		"status":"sending","locked_at":time.Now().Add(-90*time.Second).Unix(),
	}).Error; err != nil { t.Fatal(err) }
	if delay, hasWork, err = scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60); err != nil || !hasWork || delay > time.Second {
		t.Fatalf("abandoned in-flight task did not become eligible: %v %v %v", delay, hasWork, err)
	}
	if err := DB.Model(row).Updates(map[string]interface{}{
		"status":"sent", "locked_at":int64(0),
	}).Error; err != nil { t.Fatal(err) }
	if _, hasWork, err = scheduledDeliveryDelay(&model.PushDelivery{}, 5, 60); err != nil || hasWork {
		t.Fatalf("completed task still produced a deadline: %v %v", hasWork, err)
	}
}

func TestPresenceSchedulerUsesEarliestOnlineExpiry(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	if _, hasWork, err := nextPushPresenceDeadline(); err != nil || hasWork {
		t.Fatalf("empty presence table activated a deadline: %v %v", hasWork, err)
	}
	presence := &model.PushPresence{
		PeerRowId:9911,DeviceId:"device-9911",Online:true,LastSeenAt:time.Now().Unix(),
	}
	if err := DB.Create(presence).Error; err != nil { t.Fatal(err) }
	delay, hasWork, err := nextPushPresenceDeadline()
	if err != nil || !hasWork || delay < 119*time.Second || delay > 122*time.Second {
		t.Fatalf("active device expiry was not scheduled: %v %v %v",delay,hasWork,err)
	}
	if err := DB.Model(presence).Update("last_seen_at",time.Now().Add(-3*time.Minute).Unix()).Error; err != nil {
		t.Fatal(err)
	}
	delay, hasWork, err = nextPushPresenceDeadline()
	if err != nil || !hasWork || delay > time.Second {
		t.Fatalf("overdue device was not eligible: %v %v %v",delay,hasWork,err)
	}
	if err := DB.Model(presence).Update("online",false).Error; err != nil {t.Fatal(err)}
	if _, hasWork, err = nextPushPresenceDeadline(); err != nil || hasWork {
		t.Fatalf("offline device still has a deadline: %v %v",hasWork,err)
	}
}

func TestDeadlineWaitWakesImmediatelyOnNewEvent(t *testing.T) {
	wake := make(chan struct{},1)
	done := make(chan struct{})
	go func() {
		waitForWork(wake,time.Hour,true)
		close(done)
	}()
	select {
	case wake<-struct{}{}:
	case <-time.After(time.Second):t.Fatal("wake channel could not accept a message")
	}
	select {
	case <-done:
	case <-time.After(time.Second):t.Fatal("new notification was delayed by a timer")
	}
}
