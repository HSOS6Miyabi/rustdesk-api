package service

import (
    "strings"
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/config"
)

func TestDatabaseSettingPreflightRejectsUnavailableTarget(t *testing.T) {
    target:=config.Config{
        Gorm: config.Gorm{Type:config.TypeMysql},
        Mysql: config.Mysql{Addr:"127.0.0.1:1",Username:"test",Password:"local_test_secret",Dbname:"test",Tls:"false"},
    }
    err:=verifyDatabaseTarget(target)
    if err==nil {t.Fatal("unavailable target database was accepted")}
    if strings.Contains(err.Error(),"local_test_secret") {
        t.Fatal("database credentials were exposed in error messages")
    }
}

func TestDatabaseSettingPreflightRejectsInvalidPostgresPort(t *testing.T){
    target:=config.Config{
        Gorm:config.Gorm{Type:config.TypePostgresql},
        Postgresql:config.Postgresql{Host:"127.0.0.1",Port:"invalid",Dbname:"rustdesk"},
    }
    if err:=verifyDatabaseTarget(target);err==nil {
        t.Fatal("invalid PostgreSQL port was accepted")
    }
}
