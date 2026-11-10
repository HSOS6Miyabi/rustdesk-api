package service

import (
	"net"
	"os"
	"strings"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

var (
	geoIPOnce sync.Once
	geoIPDB *maxminddb.Reader
	geoIPLoadError error
)

// GeoIPReady reports whether the local MaxMind City database is available.
func GeoIPReady() bool {
	initGeoIP()
	return geoIPDB != nil
}

func initGeoIP() {
	geoIPOnce.Do(func() {
		path := strings.TrimSpace(os.Getenv("RUSTDESK_API_GEOIP_DB"))
		if path == "" { path = strings.TrimSpace(os.Getenv("RUSTDESK_API_PUSH_GEOIP_DB")) }
		if path == "" { return }
		db, err := maxminddb.Open(path)
		if err != nil {
			geoIPLoadError = err
			if Logger != nil { Logger.WithError(err).Warn("cannot read GeoIP city database") }
			return
		}
		geoIPDB = db
	})
}

type geoIPRecord struct {
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
}

// parseGeoIPValue accepts an IP address and the host:port syntax found
// in some connection audit records.
func parseGeoIPValue(raw string) net.IP {
	value := strings.TrimSpace(raw)
	if ip := net.ParseIP(value); ip != nil { return ip }
	host, _, err := net.SplitHostPort(value)
	if err == nil { return net.ParseIP(host) }
	return nil
}

// localPushGeoIP returns a country/region/city name when one is known.
func localPushGeoIP(ipValue string) string {
	ip := parseGeoIPValue(ipValue)
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() { return "" }
	initGeoIP()
	if geoIPDB == nil { return "" }
	var location geoIPRecord
	if err := geoIPDB.Lookup(ip, &location); err != nil { return "" }
	names := make([]string, 0, 3)
	add := func(m map[string]string) {
		value := m["zh-CN"]
		if value == "" { value = m["en"] }
		value = pushShort(value, 60)
		if value == "" { return }
		if len(names) == 0 || names[len(names)-1] != value {
			names = append(names, value)
		}
	}
	add(location.Country.Names)
	if len(location.Subdivisions) > 0 { add(location.Subdivisions[0].Names) }
	add(location.City.Names)
	return strings.Join(names, " · ")
}

// GeoIPDescription returns display-only geolocation text. Unknown public
// addresses remain explicit, while private and loopback IPs are identified.
func GeoIPDescription(rawIP string) string {
	ip := parseGeoIPValue(rawIP)
	if ip == nil { return "无效 IP 地址" }
	switch {
	case ip.IsLoopback(): return "本地回环地址"
	case ip.IsPrivate(): return "局域网地址"
	case ip.IsLinkLocalUnicast(): return "链路本地地址"
	case ip.IsUnspecified(): return "未指定地址"
	case ip.IsMulticast(): return "组播地址"
	}
	location := localPushGeoIP(rawIP)
	if location == "" { return "位置未知" }
	return location
}

// LookupGeoIPBatch accepts a bounded set of IP addresses for the management
// console, without sending them to any external geolocation provider.
func LookupGeoIPBatch(input []string) map[string]string {
	locations := make(map[string]string, len(input))
	for _, item := range input {
		if len(locations) >= 256 { break }
		ip := strings.TrimSpace(item)
		if ip == "" || len(ip) > 64 || net.ParseIP(ip) == nil { continue }
		locations[ip] = GeoIPDescription(ip)
	}
	return locations
}
