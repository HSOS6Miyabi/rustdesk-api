package main

import (
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/sirupsen/logrus"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type oauthBeforeDisplayName struct {
    Id           uint   `gorm:"primaryKey"`
    Op           string
    OauthType    string
    ClientId     string
    ClientSecret string
}
func (oauthBeforeDisplayName) TableName() string { return "oauths" }

func TestStartupRepairsSchemaWithoutDuplicateVersions(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file:startup_schema_repair?mode=memory&cache=shared"), &gorm.Config{})
    if err != nil { t.Fatal(err) }
    conn, err := db.DB()
    if err != nil { t.Fatal(err) }
    defer conn.Close()
    conn.SetMaxOpenConns(1)

    if err := db.AutoMigrate(&model.Version{}, &oauthBeforeDisplayName{}); err != nil { t.Fatal(err) }
    prior := &oauthBeforeDisplayName{Op:"github", OauthType:"github", ClientId:"historical-client"}
    if err := db.Create(prior).Error; err != nil { t.Fatal(err) }
    if err := db.Create(&model.Version{Version: DatabaseVersion}).Error; err != nil { t.Fatal(err) }

    previousDB, previousConfig, previousLogger := global.DB, global.Config, global.Logger
    global.DB, global.Config, global.Logger = db, config.Config{Gorm:config.Gorm{Type:config.TypeSqlite}}, logrus.New()
    defer func(){ global.DB, global.Config, global.Logger = previousDB, previousConfig, previousLogger }()

    needs, err := databaseSchemaNeedsMigration(db)
    if err != nil || !needs { t.Fatalf("missing OAuth field was not detected: %v",err) }
    if err := DatabaseAutoUpdate(); err != nil { t.Fatalf("startup migration: %v",err) }
    if !db.Migrator().HasColumn(&model.Oauth{}, "display_name") {
        t.Fatal("OAuth display name column was not added")
    }
    var saved model.Oauth
    if err := db.Where("id = ?", prior.Id).First(&saved).Error; err != nil { t.Fatal(err) }
    if saved.Op != "github" || saved.ClientId != "historical-client" {
        t.Fatalf("existing OAuth provider was changed: %+v", saved)
    }
    if err := db.Model(&saved).Update("display_name", "GitHub 工作账户").Error; err != nil { t.Fatal(err) }

    needs, err = databaseSchemaNeedsMigration(db)
    if err != nil || needs { t.Fatalf("newly migrated schema is still incomplete: %v %v",needs,err) }
    if err := db.Migrator().DropTable(&model.UserInvitation{}); err != nil { t.Fatal(err) }
    needs, err = databaseSchemaNeedsMigration(db)
    if err != nil || !needs { t.Fatalf("missing table was not detected: %v",err) }
    if err := DatabaseAutoUpdate(); err != nil { t.Fatalf("repeat startup migration: %v",err) }
    if err := DatabaseAutoUpdate(); err != nil { t.Fatalf("idempotent startup migration: %v",err) }
    var count int64
    if err := db.Model(&model.Version{}).Count(&count).Error; err != nil || count != 1 {
        t.Fatalf("schema repair wrote duplicate version records: %d %v",count,err)
    }
    if err := db.Where("id = ?", prior.Id).First(&saved).Error; err != nil || saved.DisplayName != "GitHub 工作账户" {
        t.Fatalf("schema repair did not preserve OAuth name: %+v %v",saved,err)
    }
}
