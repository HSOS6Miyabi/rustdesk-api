package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestDevicePresenceBatchOwnershipSharingAndExpiry(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	owner := &model.User{Username:"presence-owner-batch",Status:1}
	guest := &model.User{Username:"presence-guest-batch",Status:1}
	if err := DB.Create(owner).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(guest).Error; err != nil { t.Fatal(err) }
	ownerDevice := &model.Peer{Id:"batch-online-1001",UserId:owner.Id,Hostname:"Office"}
	guestDevice := &model.Peer{Id:"batch-offline-1002",UserId:guest.Id,Hostname:"Guest"}
	if err := DB.Create(ownerDevice).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(guestDevice).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(&model.PushPresence{
		PeerRowId:ownerDevice.RowId,DeviceId:ownerDevice.Id,
		Online:true,LastSeenAt:time.Now().Unix(),
	}).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(&model.PushPresence{
		PeerRowId:guestDevice.RowId,DeviceId:guestDevice.Id,
		Online:true,LastSeenAt:time.Now().Add(-121*time.Second).Unix(),
	}).Error; err != nil { t.Fatal(err) }

	ids := []string{ownerDevice.Id, guestDevice.Id, "unknown-device", ownerDevice.Id}
	guestStatus, err := DevicePresenceForUser(guest,ids,false)
	if err != nil { t.Fatal(err) }
	if guestStatus[ownerDevice.Id] || guestStatus[guestDevice.Id] || guestStatus["unknown-device"] {
		t.Fatalf("unauthorized or expired device status: %+v",guestStatus)
	}
	if err := DB.Create(&model.Contact{
		OwnerUserId:owner.Id,RecipientUserId:guest.Id,
	}).Error; err != nil { t.Fatal(err) }
	guestStatus, err = DevicePresenceForUser(guest,ids,false)
	if err != nil { t.Fatal(err) }
	if !guestStatus[ownerDevice.Id] || guestStatus[guestDevice.Id] {
		t.Fatalf("shared device or expired status incorrect: %+v",guestStatus)
	}
	ownerStatus, err := DevicePresenceForUser(owner,ids,false)
	if err != nil || !ownerStatus[ownerDevice.Id] {
		t.Fatalf("owner status unavailable: %+v %v",ownerStatus,err)
	}
	if err := DB.Where("owner_user_id = ?",owner.Id).Delete(&model.Contact{}).Error;err!=nil{t.Fatal(err)}
	if err := DB.Create(&model.AddressBook{
		UserId:guest.Id,Id:ownerDevice.Id,
	}).Error;err!=nil{t.Fatal(err)}
	guestStatus, err = DevicePresenceForUser(guest,ids,false)
	if err != nil || !guestStatus[ownerDevice.Id] {
		t.Fatalf("saved address-book device status unavailable: %+v %v",guestStatus,err)
	}
	adminStatus, err := DevicePresenceForUser(guest,ids,true)
	if err != nil || !adminStatus[ownerDevice.Id] || adminStatus[guestDevice.Id] {
		t.Fatalf("administrator presence result incorrect: %+v %v",adminStatus,err)
	}
	big := make([]string,257)
	for i := range big { big[i] = fmt.Sprintf("device-%d",i) }
	if _,err:=DevicePresenceForUser(owner,big,true);err==nil{
		t.Fatal("unbounded presence query was accepted")
	}
}

func TestAuthenticatedHeartbeatCoalescing(t *testing.T) {
	const id uint = 891327
	base := time.Date(2026,11,10,12,0,0,0,time.UTC)
	if !shouldPublishPushHeartbeat(id,"192.168.1.15",base){
		t.Fatal("first verified heartbeat was dropped")
	}
	if shouldPublishPushHeartbeat(id,"192.168.1.15",base.Add(5*time.Second)){
		t.Fatal("repeated heartbeat generated a database task")
	}
	if !shouldPublishPushHeartbeat(id,"192.168.1.16",base.Add(6*time.Second)){
		t.Fatal("device IP change was not captured")
	}
	if shouldPublishPushHeartbeat(id,"192.168.1.16",base.Add(10*time.Second)){
		t.Fatal("repeated new-IP heartbeat generated a task")
	}
	if !shouldPublishPushHeartbeat(id,"192.168.1.16",base.Add(22*time.Second)){
		t.Fatal("heartbeat was not persisted after the throttle interval")
	}
}
