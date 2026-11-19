package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestConsoleGroupSummariesDeduplicateMemberships(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	users := seedGroups(t)
	member := users["member"]
	if err := DB.Create(&model.GroupMembership{GroupId: 3, UserId: member.Id}).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&model.GroupMembership{GroupId: 2, UserId: member.Id}).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&model.GroupAnchor{GroupId: 3, UserId: users["anchor"].Id}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		if err := DB.Create(&model.Peer{Id: id, UserId: member.Id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	groups := []*model.Group{{IdModel: model.IdModel{Id: 3}, Name: "Lab"}, {IdModel: model.IdModel{Id: 2}, Name: "Shared"}, {IdModel: model.IdModel{Id: 90}, Name: "Empty"}}
	rows, err := SummarizeConsoleGroups(groups)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].MemberCount != 3 || rows[0].DeviceCount != 2 || rows[0].AnchorCount != 1 {
		t.Fatalf("incorrect primary/additional counts: %+v", rows[0])
	}
	if rows[1].MemberCount != 1 || rows[1].DeviceCount != 2 || rows[2].MemberCount != 0 {
		t.Fatalf("incorrect additional or empty counts: %+v", rows)
	}
	accounts, err := SummarizeConsoleUsers([]*model.User{member, users["other"]})
	if err != nil || accounts[0].DeviceCount != 2 || accounts[1].DeviceCount != 0 {
		t.Fatalf("account summary: %+v %v", accounts, err)
	}
	member.Password = "hidden-secret"
	encoded, _ := json.Marshal(accounts)
	if string(encoded) == "" || containsSecret(encoded, "hidden-secret") {
		t.Fatal("account summary exposed password")
	}
}

func containsSecret(data []byte, value string) bool {
	for i := 0; i+len(value) <= len(data); i++ {
		if string(data[i:i+len(value)]) == value {
			return true
		}
	}
	return false
}

func TestConsoleDeviceGroupSummariesUseActualAssignments(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.DeviceGroup{}, &model.DevicePolicy{}); err != nil {
		t.Fatal(err)
	}
	groups := []*model.DeviceGroup{{Name: "Servers"}, {Name: "Empty"}}
	for _, g := range groups {
		if err := DB.Create(g).Error; err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().Unix()
	for i, stamp := range []int64{now, now - 600} {
		if err := DB.Create(&model.Peer{Id: []string{"one", "two"}[i], GroupId: groups[0].Id, LastOnlineTime: stamp}).Error; err != nil {
			t.Fatal(err)
		}
	}
	policy := &model.DevicePolicy{Name: "Managed", Status: 2}
	role := &model.ControlRole{Name: "Read only"}
	if err := DB.Create(policy).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(role).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&model.DevicePolicyGroupAssignment{GroupId: groups[0].Id, PolicyId: policy.Id}).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&model.ControlRoleAssignment{TargetType: "device_group", TargetId: "1", RoleId: role.Id}).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := SummarizeConsoleDeviceGroups(groups)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].DeviceCount != 2 || rows[0].ActiveCount != 1 || rows[0].LastActiveAt != now || rows[0].GroupPolicy != "Managed" || rows[0].GroupPolicyStatus != 2 || rows[0].ControlRole != "Read only" {
		t.Fatalf("unexpected summary: %+v", rows[0])
	}
	if rows[1].DeviceCount != 0 || rows[1].GroupPolicy != "" || rows[1].ControlRole != "" {
		t.Fatalf("empty group has unrelated data: %+v", rows[1])
	}
	empty, err := SummarizeConsoleDeviceGroups(nil)
	if err != nil || len(empty) != 0 {
		t.Fatal("empty summaries must succeed")
	}
}
