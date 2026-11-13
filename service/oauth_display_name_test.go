package service

import (
    "strings"
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type legacyOAuthProvider struct {
    Id           uint `gorm:"primaryKey"`
    Op           string
    OauthType    string
    ClientId     string
    ClientSecret string
}

func (legacyOAuthProvider) TableName() string { return "oauths" }

func TestOAuthDisplayNameSurvivesUpgradeAndCanBeRenamed(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file:oauth_display_name_upgrade?mode=memory&cache=shared"), &gorm.Config{})
    if err != nil { t.Fatal(err) }
    sql, err := db.DB()
    if err != nil { t.Fatal(err) }
    defer sql.Close()
    sql.SetMaxOpenConns(1)

    if err := db.AutoMigrate(&legacyOAuthProvider{}); err != nil { t.Fatal(err) }
    legacy := &legacyOAuthProvider{Op: "github", OauthType: model.OauthTypeGithub,
        ClientId: "test-client", ClientSecret: "test-secret"}
    if err := db.Create(legacy).Error; err != nil { t.Fatal(err) }
    if err := db.AutoMigrate(&model.Oauth{}); err != nil { t.Fatal(err) }

    oldDB := DB
    DB = db
    defer func() { DB = oldDB }()
    service := &OauthService{}
    before := service.InfoByOp("github")
    if before.Id != legacy.Id || before.DisplayName != "" {
        t.Fatalf("previous GitHub provider was not preserved: %+v", before)
    }
    before.DisplayName = "  我的 GitHub 登录  "
    before.Op = "a-different-name"
    if err := service.Update(before); err != nil { t.Fatal(err) }
    after := service.InfoByOp("github")
    if after.Id != legacy.Id || after.DisplayName != "我的 GitHub 登录" {
        t.Fatalf("renamed GitHub provider not saved: %+v", after)
    }
    if service.InfoByOp("a-different-name").Id != 0 {
        t.Fatal("display name changed the stable OAuth authentication identifier")
    }
    if name := service.GetOauthProviderDisplayNames()["github"]; name != "我的 GitHub 登录" {
        t.Fatalf("login option is missing the stored display name: %q", name)
    }
    after.DisplayName = ""
    if err := service.Update(after); err != nil { t.Fatal(err) }
    if name := service.GetOauthProviderDisplayNames()["github"]; name != "github" {
        t.Fatalf("blank display name did not restore the provider's default label: %q", name)
    }
    after.DisplayName = strings.Repeat("字", 81)
    if err := service.Update(after); err == nil { t.Fatal("oversized OAuth display name was accepted") }
}
