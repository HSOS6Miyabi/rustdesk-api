package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func groupChange(row, group uint) DeviceGroupMemberChange {
	return DeviceGroupMemberChange{RowId: row, ExpectedGroupId: &group}
}

func TestDeviceGroupMembersAreAtomicAndPreserveDeviceDetails(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}); err != nil {
		t.Fatal(err)
	}
	a, b := &model.DeviceGroup{Name: "A"}, &model.DeviceGroup{Name: "B"}
	DB.Create(a)
	DB.Create(b)
	p := &model.Peer{Id: "desk", Alias: "Workstation", Hostname: "host", Uuid: "uuid", UserId: 42, GroupId: a.Id, LastOnlineTime: 1700000000}
	q := &model.Peer{Id: "other", GroupId: a.Id}
	DB.Create(p)
	DB.Create(q)
	count, err := ChangeDeviceGroupMembers(b.Id, []DeviceGroupMemberChange{groupChange(p.RowId, a.Id), groupChange(q.RowId, 0)}, "add")
	if !errors.Is(err, ErrDeviceGroupChanged) || count != 0 {
		t.Fatalf("stale changes were accepted: %d %v", count, err)
	}
	var read model.Peer
	DB.First(&read, "row_id = ?", p.RowId)
	if read.GroupId != a.Id {
		t.Fatal("partial batch update committed")
	}
	count, err = ChangeDeviceGroupMembers(b.Id, []DeviceGroupMemberChange{groupChange(p.RowId, a.Id), groupChange(q.RowId, a.Id)}, "add")
	if err != nil || count != 2 {
		t.Fatalf("move: %d %v", count, err)
	}
	count, err = ChangeDeviceGroupMembers(b.Id, []DeviceGroupMemberChange{groupChange(p.RowId, b.Id)}, "remove")
	if err != nil || count != 1 {
		t.Fatalf("remove: %d %v", count, err)
	}
	DB.First(&read, "row_id = ?", p.RowId)
	if read.GroupId != 0 || read.UserId != 42 || read.Uuid != "uuid" || read.Alias != "Workstation" || read.LastOnlineTime != 1700000000 {
		t.Fatalf("device details changed: %+v", read)
	}
	for _, changes := range [][]DeviceGroupMemberChange{
		nil, {groupChange(0, 0)}, {{RowId: p.RowId}}, {groupChange(p.RowId, 0), groupChange(p.RowId, 0)},
		{groupChange(p.RowId, 0), groupChange(999999, 0)}, make([]DeviceGroupMemberChange, 201),
	} {
		if _, err := ChangeDeviceGroupMembers(a.Id, changes, "add"); err == nil {
			t.Fatalf("invalid selection accepted: %+v", changes)
		}
	}
	if _, err := ChangeDeviceGroupMembers(a.Id, []DeviceGroupMemberChange{groupChange(q.RowId, b.Id)}, "remove"); !errors.Is(err, ErrDeviceGroupChanged) {
		t.Fatal("removal from unrelated group accepted")
	}
	if _, err := ChangeDeviceGroupMembers(99999, []DeviceGroupMemberChange{groupChange(p.RowId, 0)}, "add"); err == nil {
		t.Fatal("missing group accepted")
	}
}

func TestDeviceGroupDeviceListsSearchAndPaginate(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}); err != nil {
		t.Fatal(err)
	}
	group := &model.DeviceGroup{Name: "研究服务器"}
	DB.Create(group)
	user := &model.User{Username: "researcher"}
	DB.Create(user)
	for i := 0; i < 27; i++ {
		peer := &model.Peer{Id: fmt.Sprintf("server-%02d", i), Hostname: "实验室", UserId: user.Id, GroupId: group.Id}
		if err := DB.Create(peer).Error; err != nil {
			t.Fatal(err)
		}
	}
	DB.Create(&model.Peer{Id: "unassigned", UserId: user.Id})
	rows, total, err := ListDeviceGroupDevices(group.Id, false, "researcher", 2, 20)
	if err != nil || total != 27 || len(rows) != 7 {
		t.Fatalf("second page: %d %d %v", total, len(rows), err)
	}
	if rows[0].OwnerUsername != "researcher" || rows[0].GroupName != "研究服务器" {
		t.Fatalf("missing useful labels: %+v", rows[0])
	}
	rows, total, err = ListDeviceGroupDevices(group.Id, true, "unassigned", 1, 20)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].GroupId != 0 {
		t.Fatalf("candidate search: %d %+v %v", total, rows, err)
	}
	rows, total, err = ListDeviceGroupDevices(group.Id, false, "server-26", 1, 20)
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("global keyword search: %d %v", total, err)
	}
}

func TestDeviceGroupMoveRefreshesHeartbeatPolicy(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}, &model.DevicePolicy{}, &model.DevicePolicyAssignment{}); err != nil {
		t.Fatal(err)
	}
	a, b := &model.DeviceGroup{Name: "First"}, &model.DeviceGroup{Name: "Second"}
	DB.Create(a)
	DB.Create(b)
	for i, g := range []*model.DeviceGroup{a, b} {
		policy := &model.DevicePolicy{Name: g.Name, IdWhitelist: []string{"first-*", "second-*"}[i]}
		if err := SaveDevicePolicy(policy); err != nil {
			t.Fatal(err)
		}
		if err := SaveDevicePolicyGroupAssignment(&model.DevicePolicyGroupAssignment{GroupId: g.Id, PolicyId: policy.Id}); err != nil {
			t.Fatal(err)
		}
	}
	peer := &model.Peer{Id: "managed", Uuid: "registered", GroupId: a.Id}
	DB.Create(peer)
	first, err := HeartbeatDeviceStrategy(peer, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ChangeDeviceGroupMembers(b.Id, []DeviceGroupMemberChange{groupChange(peer.RowId, a.Id)}, "add"); err != nil {
		t.Fatal(err)
	}
	DB.First(peer, "row_id = ?", peer.RowId)
	next, err := HeartbeatDeviceStrategy(peer, first["modified_at"].(int64))
	if err != nil {
		t.Fatal(err)
	}
	options := next["strategy"].(map[string]interface{})["config_options"].(map[string]string)
	if options["id-whitelist"] != "second-*" {
		t.Fatalf("group policy was not refreshed: %+v", next)
	}
}
