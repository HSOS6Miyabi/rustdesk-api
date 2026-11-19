package service

import (
	"strconv"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestDeviceGroupMembershipChangesOnlySelectedRegisteredRows(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}, &model.DevicePolicy{}, &model.DevicePolicyGroupAssignment{}); err != nil {
		t.Fatal(err)
	}
	first := model.DeviceGroup{Name: "Lab"}
	second := model.DeviceGroup{Name: "Servers"}
	if err := DB.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	a := model.Peer{Id: "desk-1", Alias: "Workstation", GroupId: first.Id}
	b := model.Peer{Id: "desk-2", Alias: "Server", GroupId: second.Id}
	c := model.Peer{Id: "desk-3", Alias: "Spare"}
	for _, p := range []*model.Peer{&a, &b, &c} {
		if err := DB.Create(p).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := AllService.GroupService.ChangeDeviceGroupMembers(first.Id, []uint{b.RowId, c.RowId}, "add"); err != nil {
		t.Fatal(err)
	}
	var changed []model.Peer
	if err := DB.Order("row_id").Find(&changed).Error; err != nil {
		t.Fatal(err)
	}
	for _, peer := range changed {
		if peer.GroupId != first.Id {
			t.Fatalf("device %s remained in group %d", peer.Id, peer.GroupId)
		}
	}
	if err := AllService.GroupService.ChangeDeviceGroupMembers(first.Id, []uint{a.RowId, b.RowId}, "remove"); err != nil {
		t.Fatal(err)
	}
	if err := DB.Where("row_id IN ?", []uint{a.RowId, b.RowId}).Find(&changed).Error; err != nil {
		t.Fatal(err)
	}
	for _, rowID := range []uint{a.RowId, b.RowId} {
		var peer model.Peer
		if err := DB.First(&peer, "row_id = ?", rowID).Error; err != nil {
			t.Fatal(err)
		}
		if peer.GroupId != 0 {
			t.Fatalf("device %d still in group %d", rowID, peer.GroupId)
		}
	}
	var retained model.Peer
	if err := DB.First(&retained, "row_id = ?", c.RowId).Error; err != nil {
		t.Fatal(err)
	}
	if retained.GroupId != first.Id {
		t.Fatalf("unselected device changed group: %d", retained.GroupId)
	}
	for _, invalid := range [][]uint{{}, {0}, {a.RowId, 999999}} {
		if err := AllService.GroupService.ChangeDeviceGroupMembers(first.Id, invalid, "add"); err == nil {
			t.Fatalf("accepted invalid device rows: %v", invalid)
		}
	}
	if err := AllService.GroupService.ChangeDeviceGroupMembers(first.Id, []uint{a.RowId, c.RowId}, "remove"); err == nil {
		t.Fatal("mixed membership removal should reject the entire operation")
	}
	if err := DB.First(&retained, "row_id = ?", c.RowId).Error; err != nil {
		t.Fatal(err)
	}
	if retained.GroupId != first.Id {
		t.Fatal("failed operation modified another device")
	}
	page, err := AllService.GroupService.DeviceGroupDevices(first.Id, 1, 20, "Spare")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Peers) != 1 || page.Peers[0].RowId != c.RowId {
		t.Fatalf("unexpected filtered device list: %+v", page)
	}
}

func TestDeviceGroupDirectoryProvidesAccurateTotalsAndPolicies(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}, &model.DevicePolicy{}, &model.DevicePolicyGroupAssignment{}); err != nil {
		t.Fatal(err)
	}
	group := model.DeviceGroup{Name: "Office"}
	if err := DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	policy := model.DevicePolicy{Name: "Office policy", Status: 1}
	if err := DB.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	assignment := model.DevicePolicyGroupAssignment{GroupId: group.Id, PolicyId: policy.Id}
	if err := DB.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	role := model.ControlRole{Name: "Full control"}
	if err := DB.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	roleAssignment := model.ControlRoleAssignment{TargetType: "device_group", TargetId: strconv.FormatUint(uint64(group.Id), 10), RoleId: role.Id}
	if err := DB.Create(&roleAssignment).Error; err != nil {
		t.Fatal(err)
	}
	devices := []model.Peer{{Id: "desk-1", GroupId: group.Id}, {Id: "desk-2", GroupId: group.Id}, {Id: "desk-3", GroupId: 0}}
	for i := range devices {
		if err := DB.Create(&devices[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	list := AllService.GroupService.DeviceGroupList(1, 20, nil)
	if err := EnrichDeviceGroupDirectory(list.DeviceGroups); err != nil {
		t.Fatal(err)
	}
	if len(list.DeviceGroups) != 1 {
		t.Fatalf("groups: %v", list.DeviceGroups)
	}
	result := list.DeviceGroups[0]
	if result.DeviceCount != 2 || result.PolicyName != "Office policy" || result.ControlRoleName != "Full control" {
		t.Fatalf("unexpected device group summary: %+v", result)
	}
}
