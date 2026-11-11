package config

import (
    "os"
    "path/filepath"
    "testing"
)

func TestEnvironmentOnlyConfigurationIsDecoded(t *testing.T){
    directory:=t.TempDir()
    file:=filepath.Join(directory,"config.yaml")
    if err:=os.WriteFile(file,[]byte("app:\n  register: true\n"),0600);err!=nil{t.Fatal(err)}
    t.Setenv("RUSTDESK_API_CACHE_TYPE","memory")
    t.Setenv("RUSTDESK_API_APP_REGISTER","false")
    t.Setenv("RUSTDESK_API_GIN_API_ADDR","127.0.0.1:27114")
    var cfg Config
    Init(&cfg,file)
    if cfg.Cache.Type!="memory" {t.Fatalf("environment-only cache type was not loaded: %q",cfg.Cache.Type)}
    if cfg.App.Register {t.Fatal("environment variable did not override YAML boolean")}
    if cfg.Gin.ApiAddr!="127.0.0.1:27114" {
        t.Fatalf("HTTP listener environment setting was not decoded: %q",cfg.Gin.ApiAddr)
    }
}
