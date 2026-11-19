package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestGroupDirectoryCountsMembersAndOwnedDevices(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	group := model.Group{Name: "Team", Type: model.GroupTypeAnchor}
	if err := DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	user1 := model.User{Username: "group-one", GroupId: group.Id, Status: 1}
	user2 := model.User{Username: "group-two", GroupId: 1, Status: 1}
	if err := DB.Create(&user1).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&user2).Error; err != nil {
		t.Fatal(err)
	}
	for _, membership := range []model.GroupMembership{
		{GroupId: group.Id, UserId: user1.Id},
		{GroupId: group.Id, UserId: user2.Id},
	} {
		if err := DB.Create(&membership).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := DB.Create(&model.GroupAnchor{GroupId: group.Id, UserId: user2.Id}).Error; err != nil {
		t.Fatal(err)
	}
	for _, device := range []model.Peer{
		{Id: "desk-1", UserId: user1.Id}, {Id: "desk-2", UserId: user1.Id},
		{Id: "desk-3", UserId: user2.Id}, {Id: "unassigned", UserId: 0},
	} {
		if err := DB.Create(&device).Error; err != nil {
			t.Fatal(err)
		}
	}
	groups := []*model.Group{&group}
	if err := EnrichGroupDirectory(groups); err != nil {
		t.Fatal(err)
	}
	if group.MemberCount != 2 || group.DeviceCount != 3 || group.AnchorCount != 1 {
		t.Fatalf("incorrect directory counts: members=%d devices=%d anchors=%d", group.MemberCount, group.DeviceCount, group.AnchorCount)
	}
	users := []*model.User{&user1, &user2}
	if err := EnrichUserDeviceCounts(users); err != nil {
		t.Fatal(err)
	}
	if user1.DeviceCount != 2 || user2.DeviceCount != 1 {
		t.Fatalf("incorrect account device counts: %d, %d", user1.DeviceCount, user2.DeviceCount)
	}
}
