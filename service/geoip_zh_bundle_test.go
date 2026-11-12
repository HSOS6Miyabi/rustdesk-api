package service

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oschwald/maxminddb-golang"
)

func TestBundledGeoIPChineseXiAn(t *testing.T) {
	path := filepath.Join("..", "resources", "geoip", "dbip-city-lite.mmdb")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("city MMDB is included when preparing the release")
	} else if err != nil {
		t.Fatal(err)
	}
	database, err := maxminddb.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	var location geoIPRecord
	if err := database.Lookup(net.ParseIP("221.11.67.246"), &location); err != nil {
		t.Fatal(err)
	}
	got := chinaGeoIPDescription(location)
	for _, name := range []string{"中国", "陕西", "西安", "雁塔区"} {
		if !strings.Contains(got, name) {
			t.Fatalf("bundled GeoIP location %q is missing %q", got, name)
		}
	}
}
