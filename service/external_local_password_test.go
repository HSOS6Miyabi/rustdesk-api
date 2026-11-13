package service

import (
	"errors"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"github.com/sirupsen/logrus"
)

func TestExternalAccountsCanSetLocalPasswordAndLoginWithoutIdentityProvider(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	oldConfig, oldLogger := Config, Logger
	Config, Logger = &config.Config{}, logrus.New()
	defer func() { Config, Logger = oldConfig, oldLogger }()
	AllService.UserService = &UserService{}
	AllService.LdapService = &LdapService{}
	if err := DB.AutoMigrate(&model.UserThird{}, &model.UserToken{}); err != nil {
		t.Fatal(err)
	}

	sso := &model.User{Username: "sso-local-recovery", Email: "sso@example.test", GroupId: 1, Status: 1}
	if err := DB.Create(sso).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(&model.UserThird{UserId: sso.Id, OauthType: model.OauthTypeOidc, Op: "company-oidc"}).Error; err != nil {
		t.Fatal(err)
	}
	ldapUser, err := AllService.LdapService.mapToLocalUser(&Config.Ldap, &LdapUser{
		Username: "ldap-local-recovery", Email: "ldap@example.test",
		FirstName: "Directory", LastName: "Member", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sso.Password != "" || ldapUser.Password != "" {
		t.Fatal("external identities unexpectedly have local passwords")
	}

	for _, item := range []struct {
		user *model.User
		password string
	}{
		{sso, "SSO-Local-Password2026"},
		{ldapUser, "LDAP-Local-Password2026"},
	} {
		if err := DB.Create(&model.UserToken{
			UserId: item.user.Id, Token: item.user.Username + "-existing-session",
			ExpiredAt: time.Now().Add(time.Hour).Unix(),
		}).Error; err != nil {
			t.Fatal(err)
		}
		if err := AllService.UserService.ChangeOwnPassword(item.user.Id, "", item.password); err != nil {
			t.Fatalf("first local password for %s: %v", item.user.Username, err)
		}
		var stored model.User
		if err := DB.First(&stored, item.user.Id).Error; err != nil {
			t.Fatal(err)
		}
		if stored.Password == "" || stored.Password == item.password {
			t.Fatal("local password was not securely hashed")
		}
		var remaining int64
		if err := DB.Model(&model.UserToken{}).Where("user_id = ?", item.user.Id).Count(&remaining).Error; err != nil || remaining != 0 {
			t.Fatalf("existing sessions were not revoked: %d (%v)", remaining, err)
		}
		Config.Ldap.Enable = false
		if got := AllService.UserService.InfoByUsernamePassword(item.user.Username, item.password); got.Id != item.user.Id {
			t.Fatal("local login failed without LDAP")
		}
		// Check that a valid local password works without contacting LDAP.
		Config.Ldap.Enable = true
		Config.Ldap.Url = "ldap://127.0.0.1:1"
		if got := AllService.UserService.InfoByUsernamePassword(item.user.Username, item.password); got.Id != item.user.Id {
			t.Fatal("local login attempted unreachable LDAP before verifying the local credential")
		}
		Config.Ldap.Enable = false
		if got := AllService.UserService.InfoByUsernamePassword(item.user.Username, "incorrect"); got.Id != 0 {
			t.Fatal("incorrect local password was accepted")
		}
		if err := AllService.UserService.ChangeOwnPassword(item.user.Id, "", "AnotherStrongPassword2026"); !errors.Is(err, ErrCurrentPasswordIncorrect) {
			t.Fatalf("changing an existing password without the old one must fail: %v", err)
		}
		if err := AllService.UserService.ChangeOwnPassword(item.user.Id, "incorrect", "AnotherStrongPassword2026"); !errors.Is(err, ErrCurrentPasswordIncorrect) {
			t.Fatalf("changing an existing password with an incorrect old one must fail: %v", err)
		}
		if err := AllService.UserService.ChangeOwnPassword(item.user.Id, item.password, ""); err == nil {
			t.Fatal("empty replacement password was accepted")
		}
		newPassword := "Updated-Local-Password2026"
		if err := AllService.UserService.ChangeOwnPassword(item.user.Id, item.password, newPassword); err != nil {
			t.Fatal(err)
		}
		after := AllService.UserService.InfoById(item.user.Id)
		ok, _, verifyErr := utils.VerifyPassword(after.Password, newPassword)
		if verifyErr != nil || !ok {
			t.Fatal("replacement password was not saved")
		}
		if got := AllService.UserService.InfoByUsernamePassword(item.user.Username, item.password); got.Id != 0 {
			t.Fatal("previous local password remained valid")
		}
	}
}
