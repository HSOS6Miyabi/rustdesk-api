package service

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestConnectionAccessChecksManagedDeviceDeployment(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	oldConfig := Config
	Config = &config.Config{}
	Config.App.RequireDeviceDeployment = true
	defer func() { Config = oldConfig }()
	if err := DB.AutoMigrate(
		&model.UserToken{}, &model.DeviceDeployment{}, &model.AccessRule{},
	); err != nil {
		t.Fatal(err)
	}

	users := seedGroups(t)
	owner, viewer := users["anchor"], users["member"]
	id := "managed-access-device"
	uuid := base64.StdEncoding.EncodeToString([]byte("managed-access-test-uuid"))
	peer := &model.Peer{Id: id, Uuid: uuid, UserId: owner.Id}
	if err := DB.Create(peer).Error; err != nil {
		t.Fatal(err)
	}
	for _, user := range []*model.User{owner, viewer} {
		if err := DB.Create(&model.UserToken{
			UserId: user.Id, Token: user.Username + "-managed-access-token",
			ExpiredAt: time.Now().Add(time.Hour).Unix(),
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	allow := &model.AccessRule{
		Name: "Shared managed device", SourceType: "all",
		TargetType: "device", TargetID: id, Effect: "allow", Enabled: true,
	}
	if err := SaveAccessRule(allow); err != nil {
		t.Fatal(err)
	}

	check := func(user *model.User, expectedAllow bool, expectedReason string) {
		t.Helper()
		result, err := AuthorizeConnection(user.Username+"-managed-access-token", id)
		if err != nil || result.Allow != expectedAllow ||
			(expectedReason != "" && result.Reason != expectedReason) {
			t.Fatalf(
				"authorization for %s: %+v, %v; expected allow=%v, reason=%s",
				user.Username, result, err, expectedAllow, expectedReason,
			)
		}
	}
	check(owner, false, "device_not_deployed")
	check(viewer, false, "device_not_deployed")

	deployment := &model.DeviceDeployment{
		DeviceId: id, Uuid: uuid,
		PublicKey: base64.StdEncoding.EncodeToString([]byte("01234567890123456789012345678901")),
		OwnerUserId: owner.Id, Status: "approved",
	}
	if err := DB.Create(deployment).Error; err != nil {
		t.Fatal(err)
	}
	check(owner, true, "device_owner")
	check(viewer, true, "matching_allow")

	set := func(column string, value interface{}) {
		t.Helper()
		if err := DB.Model(&model.DeviceDeployment{}).Where("device_id = ?", id).
			Update(column, value).Error; err != nil {
			t.Fatal(err)
		}
	}
	set("status", "revoked")
	check(owner, false, "device_not_approved")
	check(viewer, false, "device_not_approved")
	set("status", "approved")
	set("uuid", base64.StdEncoding.EncodeToString([]byte("other-device-uuid")))
	check(owner, false, "device_not_approved")
	set("uuid", uuid)
	set("owner_user_id", viewer.Id)
	check(owner, false, "device_not_approved")
	set("owner_user_id", owner.Id)
	check(owner, true, "device_owner")

	Config.App.RequireDeviceDeployment = false
	set("status", "revoked")
	check(owner, true, "device_owner")
	check(viewer, true, "matching_allow")
}
