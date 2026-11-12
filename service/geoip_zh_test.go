package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func chineseGeoIPRecordForTest(t *testing.T, provinceCode, provinceName, cityName string) geoIPRecord {
	t.Helper()
	data, err := json.Marshal(map[string]interface{}{
		"Country": map[string]interface{}{
			"ISOCode": "CN",
			"Names":   map[string]string{"en": "China", "zh-CN": "中国"},
		},
		"Subdivisions": []map[string]interface{}{
			{"ISOCode": provinceCode, "Names": map[string]string{"en": provinceName}},
		},
		"City": map[string]interface{}{
			"Names": map[string]string{"en": cityName},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var record geoIPRecord
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestGeoIPChineseAdministrativeDivisions(t *testing.T) {
	tests := []struct {
		name         string
		provinceISO  string
		provinceName string
		city         string
		want         string
	}{
		{"Shaanxi district", "SN", "Shaanxi", "Xi'an (Yanta Qu)", "中国 · 陕西 · 西安 · 雁塔区"},
		{"Shaanxi district by English suffix", "", "Shaanxi", "Xi'an (Yanta District)", "中国 · 陕西 · 西安 · 雁塔区"},
		{"Shaanxi district without city", "SN", "Shaanxi", "Yanta Qu", "中国 · 陕西 · 西安 · 雁塔区"},
		{"Shanxi is distinct from Shaanxi", "SX", "Shanxi", "Taiyuan (Xiaodian Qu)", "中国 · 山西 · 太原 · 小店区"},
		{"Zhejiang district", "ZJ", "Zhejiang", "Hangzhou (Xihu Qu)", "中国 · 浙江 · 杭州 · 西湖区"},
		{"Municipality district", "BJ", "Beijing", "Beijing (Chaoyang Qu)", "中国 · 北京 · 朝阳区"},
		{"English place remains when unrecognized", "SN", "Shaanxi", "Unknown locality", "中国 · 陕西 · Unknown locality"},
		{"Province without city", "SN", "Shaanxi", "", "中国 · 陕西"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := chineseGeoIPRecordForTest(t, tt.provinceISO, tt.provinceName, tt.city)
			if got := chinaGeoIPDescription(record); got != tt.want {
				t.Fatalf("translated address = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeoIPChineseTranslationOnlyAppliesToChina(t *testing.T) {
	var record geoIPRecord
	record.Country.ISOCode = "US"
	record.Country.Names = map[string]string{"en": "United States", "zh-CN": "美国"}
	if got := chinaGeoIPDescription(record); got != "" {
		t.Fatalf("foreign location modified: %q", got)
	}
}

func TestGeoIPChineseDivisionCoverage(t *testing.T) {
	index := chineseLocationIndex()
	if len(index.byCode) < 3500 {
		t.Fatalf("Chinese province, city and district data incomplete: %d entries", len(index.byCode))
	}
	for _, key := range []string{"61", "6101", "610113"} {
		if _, ok := index.byCode[key]; !ok {
			t.Fatalf("missing geographical division %s", key)
		}
	}
	record := chineseGeoIPRecordForTest(t, "SN", "Shaanxi", "Xi'an (Yanta Qu)")
	if got := chinaGeoIPDescription(record); !strings.Contains(got, "雁塔区") {
		t.Fatalf("district not translated: %q", got)
	}
}
