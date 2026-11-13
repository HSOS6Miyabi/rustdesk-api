package service

import (
    "strings"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/config"
    legacyjwt "github.com/lejianwen/rustdesk-api/v2/lib/jwt"
    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestLoginSessionPersistsAndRefreshesAcrossKeyChanges(t *testing.T) {
    cleanup := setupGroupDB(t)
    defer cleanup()
    if err := DB.AutoMigrate(&model.UserToken{}, &model.LoginLog{}); err != nil { t.Fatal(err) }

    oldConfig, oldJWT := Config, Jwt
    Config = &config.Config{App: config.App{TokenExpire: 3 * time.Hour}}
    Jwt = legacyjwt.NewJwt("original-signature-key", time.Second)
    defer func() { Config, Jwt = oldConfig, oldJWT }()

    user := &model.User{Username: "persistent-session-user", Status: model.COMMON_STATUS_ENABLE}
    if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
    firstInstance := &UserService{}
    login := firstInstance.Login(user, &model.LoginLog{UserId: user.Id})
    if login.Id == 0 || login.Token == "" || strings.Contains(login.Token, ".") {
        t.Fatalf("new logins must use independently generated, persistent opaque tokens: %+v", login)
    }

    // Simulate a new API process using the same database and a different JWT key.
    Jwt = legacyjwt.NewJwt("changed-signature-key", time.Second)
    nextInstance := &UserService{}
    got, active := nextInstance.InfoByAccessToken(login.Token)
    if got.Id != user.Id || active.Id != login.Id {
        t.Fatal("existing database session was rejected after API restart")
    }

    previousExpiry := time.Now().Add(20 * time.Minute).Unix()
    if err := DB.Model(&model.UserToken{}).Where("id = ?", login.Id).
        Update("expired_at", previousExpiry).Error; err != nil { t.Fatal(err) }
    got, active = nextInstance.InfoByAccessToken(login.Token)
    if got.Id != user.Id { t.Fatal("near-expiration login could not be found") }
    nextInstance.AutoRefreshAccessToken(active)
    if active.ExpiredAt <= previousExpiry || active.ExpiredAt < time.Now().Add(2*time.Hour).Unix() {
        t.Fatal("active login was not extended to the configured session duration")
    }

    if err := nextInstance.FlushToken(user); err != nil { t.Fatal(err) }
    got, _ = nextInstance.InfoByAccessToken(login.Token)
    if got.Id != 0 { t.Fatal("revoked session was still accepted") }
}

func TestSessionLifetimeDefaultUsesHoursNotNanoseconds(t *testing.T) {
    oldConfig := Config
    defer func() { Config = oldConfig }()
    Config = &config.Config{}
    now := time.Now().Unix()
    expires := (&UserService{}).UserTokenExpireTimestamp()
    if expires < now+29*24*3600 || expires > now+31*24*3600 {
        t.Fatalf("expected approximately 30 days when expiration is not configured, got %d", expires-now)
    }
}
