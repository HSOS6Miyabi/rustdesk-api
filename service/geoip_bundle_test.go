package service

import (
    "net"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/oschwald/maxminddb-golang"
)

func TestBundledGeoIPCityDatabase(t *testing.T) {
    path := filepath.Join("..", "resources", "geoip", "dbip-city-lite.mmdb")
    if _, err := os.Stat(path); os.IsNotExist(err) {
        t.Skip("database is generated when preparing release packages")
    } else if err != nil {
        t.Fatal(err)
    }
    reader, err := maxminddb.Open(path)
    if err != nil { t.Fatalf("open packaged GeoIP database: %v", err) }
    defer reader.Close()
    if reader.Metadata.NodeCount < 100000 {
        t.Fatalf("incomplete MMDB database: only %d nodes", reader.Metadata.NodeCount)
    }
    var city geoIPRecord
    if err := reader.Lookup(net.ParseIP("8.8.8.8"), &city); err != nil {
        t.Fatalf("lookup a public IPv4 address: %v", err)
    }
    if strings.TrimSpace(city.Country.Names["en"]) == "" {
        t.Fatal("city database did not return a country for 8.8.8.8")
    }
    if err := reader.Lookup(net.ParseIP("2001:4860:4860::8888"), &city); err != nil {
        t.Fatalf("lookup a public IPv6 address: %v", err)
    }
}
