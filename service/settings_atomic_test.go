package service

import (
    "errors"
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/gorm"
)

func TestSettingsDatabaseOverrideAndAuditHistoryAreAtomic(t *testing.T) {
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.SystemSettingOverride{},
        &model.SystemSettingRevision{},&model.SystemSettingHistory{});err!=nil{t.Fatal(err)}
    if err:=DB.Create(&model.SystemSettingRevision{ID:1,Version:1}).Error;err!=nil{t.Fatal(err)}
    value:="false"
    input:=SettingsRevisionInput{
        DatabaseVersion:1,
        Changes:[]SettingsChange{{Key:"app.register",Enabled:true,Value:&value}},
    }

    const name="test:simulate_history_failure"
    if err:=DB.Callback().Create().Before("gorm:create").Register(name,func(tx *gorm.DB){
        if tx.Statement.Schema!=nil&&tx.Statement.Schema.Table=="system_setting_histories"{
            tx.AddError(errors.New("audit database unavailable"))
        }
    });err!=nil{t.Fatal(err)}
    err:=SaveSettingChanges(1,input)
    if removeErr:=DB.Callback().Create().Remove(name);removeErr!=nil{t.Fatal(removeErr)}
    if err==nil{t.Fatal("failed history insert was accepted")}
    var overrides int64
    if err:=DB.Model(&model.SystemSettingOverride{}).Count(&overrides).Error;err!=nil{t.Fatal(err)}
    if overrides!=0{t.Fatal("setting override was stored without audit history")}
    var revision model.SystemSettingRevision
    if err:=DB.First(&revision,1).Error;err!=nil{t.Fatal(err)}
    if revision.Version!=1{t.Fatalf("failed setting changed revision %d",revision.Version)}
    if err:=SaveSettingChanges(1,input);err!=nil{t.Fatal(err)}
    var histories int64
    if err:=DB.Model(&model.SystemSettingHistory{}).Count(&histories).Error;err!=nil{t.Fatal(err)}
    if histories!=1{t.Fatalf("expected one committed audit history, got %d",histories)}
}
