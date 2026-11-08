package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestOwnedGroupPermissionsAndMembership(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	users := seedGroups(t)
	owner := users["member"]
	other := users["other"]
	administrator := users["admin"]
	groups := AllService.GroupService

	created, err := groups.CreateOwnedGroup(owner.Id, "我的小组", model.GroupTypeAnchor)
	if err != nil { t.Fatal(err) }
	if created.OwnerUserId != owner.Id { t.Fatalf("incorrect owner: %d", created.OwnerUserId) }
	entries, err := groups.OwnedGroupMembers(owner.Id, created.Id)
	if err != nil || !includeID(entries, owner.Id) { t.Fatalf("owner membership: %v %v", entries, err) }

	if _, err := groups.OwnedGroup(other.Id, created.Id); err == nil { t.Fatal("another user accessed the group") }
	if err := groups.UpdateOwnedGroup(other.Id, created.Id, "changed", model.GroupTypeShare); err == nil { t.Fatal("unauthorized update") }
	if err := groups.DeleteOwnedGroup(other.Id, created.Id); err == nil { t.Fatal("unauthorized deletion") }
	if err := groups.ChangeOwnedMembers(other.Id, created.Id, []uint{other.Id}, "add"); err == nil { t.Fatal("unauthorized member change") }
	if err := groups.ChangeOwnedAnchors(other.Id, created.Id, []uint{other.Id}); err == nil { t.Fatal("unauthorized public member change") }
	if err := groups.ChangeOwnedMembers(owner.Id, created.Id, []uint{administrator.Id}, "add"); err == nil { t.Fatal("group owner enrolled an administrator") }

	if err := groups.ChangeOwnedMembers(owner.Id, created.Id, []uint{other.Id}, "add"); err != nil { t.Fatal(err) }
	members, err := groups.OwnedGroupMembers(owner.Id, created.Id)
	if err != nil || !includeID(members, other.Id) { t.Fatalf("member addition: %v %v", members, err) }

	if err := groups.ChangeOwnedAnchors(owner.Id, created.Id, []uint{administrator.Id}); err != nil { t.Fatal(err) }
	anchors, err := groups.OwnedGroupAnchors(owner.Id, created.Id)
	if err != nil || !includeID(anchors, administrator.Id) { t.Fatalf("anchor addition: %v %v", anchors, err) }

	if err := groups.ChangeOwnedMembers(owner.Id, created.Id, []uint{}, "replace"); err != nil { t.Fatal(err) }
	members, err = groups.OwnedGroupMembers(owner.Id, created.Id)
	if err != nil || len(members) != 1 || members[0] != owner.Id { t.Fatalf("owner lost membership: %v %v", members, err) }
	if err := groups.DeleteOwnedGroup(owner.Id, created.Id); err != nil { t.Fatal(err) }
	if _, err := groups.OwnedGroup(owner.Id, created.Id); err == nil { t.Fatal("deleted group was returned") }
}

func TestAuditParticipationAndOutOfOrderConnectionUpdates(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AuditConn{}, &model.AuditFile{}); err != nil { t.Fatal(err) }
	users := seedGroups(t)
	sourceUser, targetUser, stranger := users["member"], users["other"], users["contact"]
	AllService.AuditService = &AuditService{}
	source := &model.Peer{Id: "source-device", Hostname: "Source PC", UserId: sourceUser.Id}
	target := &model.Peer{Id: "target-device", Hostname: "Target PC", UserId: targetUser.Id}
	if err := DB.Create(source).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(target).Error; err != nil { t.Fatal(err) }
	audit := AllService.AuditService

	// The destination reports a session start before the source peer ID arrives.
	if err := audit.SaveAuditConnection(&model.AuditConn{Action: "new", PeerId: "target-device", ConnId: 78}); err != nil { t.Fatal(err) }
	if err := audit.SaveAuditConnection(&model.AuditConn{PeerId: "target-device", ConnId: 78,
		FromPeer: "source-device", FromName: "Remote Operator", SessionId: "session-78"}); err != nil { t.Fatal(err) }
	if err := audit.SaveAuditConnection(&model.AuditConn{Action: "close", PeerId: "target-device", ConnId: 78}); err != nil { t.Fatal(err) }

	var saved []model.AuditConn
	if err := DB.Find(&saved).Error; err != nil { t.Fatal(err) }
	if len(saved) != 1 { t.Fatalf("expected one merged connection, got %d", len(saved)) }
	row := saved[0]
	if row.FromPeer != source.Id || row.PeerId != target.Id || row.FromUserId != sourceUser.Id || row.ToUserId != targetUser.Id {
		t.Fatalf("both participants were not saved: %+v", row)
	}
	if row.FromAccount != sourceUser.Username || row.ToAccount != targetUser.Username ||
		row.FromDeviceName != source.Hostname || row.ToDeviceName != target.Hostname {
		t.Fatalf("account or device names were not saved: %+v", row)
	}
	if row.FromName != "Remote Operator" || row.CloseTime == 0 { t.Fatalf("follow-up information lost: %+v", row) }

	outgoing, err := audit.OwnConnections(sourceUser.Id, 1, 20)
	if err != nil || outgoing.Total != 1 || outgoing.AuditConns[0].Direction != "outgoing" {
		t.Fatalf("outgoing records: %+v %v", outgoing, err)
	}
	incoming, err := audit.OwnConnections(targetUser.Id, 1, 20)
	if err != nil || incoming.Total != 1 || incoming.AuditConns[0].Direction != "incoming" {
		t.Fatalf("incoming records: %+v %v", incoming, err)
	}
	private, err := audit.OwnConnections(stranger.Id, 1, 20)
	if err != nil || private.Total != 0 { t.Fatalf("unrelated user could read connection logs: %+v %v", private, err) }

	file := &model.AuditFile{PeerId: "target-device", FromPeer: "source-device", Path: "/shared/document.pdf"}
	if err := audit.SaveAuditFile(file); err != nil { t.Fatal(err) }
	for _, test := range []struct{ user uint; expected string }{
		{sourceUser.Id, "outgoing"}, {targetUser.Id, "incoming"},
	} {
		list, err := audit.OwnFiles(test.user, 1, 20)
		if err != nil || list.Total != 1 || list.AuditFiles[0].Direction != test.expected {
			t.Fatalf("file audit (%d): %+v %v", test.user, list, err)
		}
	}
	privateFiles, err := audit.OwnFiles(stranger.Id, 1, 20)
	if err != nil || privateFiles.Total != 0 { t.Fatalf("unrelated user could read file logs: %+v %v", privateFiles, err) }

	// Device ownership changes do not erase the original session participants.
	if err := DB.Model(target).Update("user_id", stranger.Id).Error; err != nil { t.Fatal(err) }
	previous, err := audit.OwnConnections(targetUser.Id, 1, 20)
	if err != nil || previous.Total != 1 || previous.AuditConns[0].ToAccount != targetUser.Username {
		t.Fatalf("historical ownership was lost: %+v %v", previous, err)
	}
	// The device's new owner must not see records from an earlier ownership period.
	newOwnerHistory, err := audit.OwnConnections(stranger.Id, 1, 20)
	if err != nil || newOwnerHistory.Total != 0 {
		t.Fatalf("device transfer exposed the former owner's connection history: %+v %v", newOwnerHistory, err)
	}
	newOwnerFiles, err := audit.OwnFiles(stranger.Id, 1, 20)
	if err != nil || newOwnerFiles.Total != 0 {
		t.Fatalf("device transfer exposed the former owner's file history: %+v %v", newOwnerFiles, err)
	}
}

func TestAuditRecordWithoutStoredOwnerUsesRegisteredDevice(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AuditConn{}, &model.AuditFile{}); err != nil { t.Fatal(err) }
	users := seedGroups(t)
	AllService.AuditService = &AuditService{}
	device := &model.Peer{Id: "historical-target", Hostname: "Old target", UserId: users["member"].Id}
	if err := DB.Create(device).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(&model.AuditConn{PeerId: device.Id, ConnId: 1, FromPeer: "unregistered-external"}).Error; err != nil {
		t.Fatal(err)
	}
	history, err := AllService.AuditService.OwnConnections(users["member"].Id, 1, 20)
	if err != nil || history.Total != 1 || history.AuditConns[0].ToAccount != users["member"].Username {
		t.Fatalf("legacy audit record should be visible with recovered name: %+v %v", history, err)
	}
}
