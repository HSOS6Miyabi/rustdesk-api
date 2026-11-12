package service

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
	"unicode"
)

//go:embed geoip_zh_divisions.json
var chineseGeoIPDivisionData []byte

type chineseGeoIPDivision struct {
	Code       string `json:"c"`
	ParentCode string `json:"p"`
	Level      int    `json:"l"`
	Name       string `json:"n"`
	Pinyin     string `json:"py"`
	FullName   string `json:"f"`
}

type chineseGeoIPIndex struct {
	byCode    map[string]chineseGeoIPDivision
	children  map[string]map[string][]chineseGeoIPDivision
	provinces map[string]chineseGeoIPDivision
}

var chineseGeoIPOnce sync.Once
var chineseGeoIPIndexData chineseGeoIPIndex

func chineseLocationIndex() *chineseGeoIPIndex {
	chineseGeoIPOnce.Do(func() {
		var divisions []chineseGeoIPDivision
		if err := json.Unmarshal(chineseGeoIPDivisionData, &divisions); err != nil {
			if Logger != nil {
				Logger.WithError(err).Error("cannot parse bundled Chinese geographic names")
			}
			return
		}
		index := chineseGeoIPIndex{
			byCode:    make(map[string]chineseGeoIPDivision, len(divisions)),
			children:  make(map[string]map[string][]chineseGeoIPDivision),
			provinces: make(map[string]chineseGeoIPDivision),
		}
		for _, division := range divisions {
			index.byCode[division.Code] = division
			if index.children[division.ParentCode] == nil {
				index.children[division.ParentCode] = make(map[string][]chineseGeoIPDivision)
			}
			for _, alias := range []string{division.Pinyin, division.Name, division.FullName} {
				index.add(division.ParentCode, alias, division)
			}
		}
		for alias, code := range chineseProvinceAliases {
			if province, found := index.byCode[code]; found {
				index.provinces[geoIPChineseKey(alias)] = province
			}
		}
		for _, division := range divisions {
			if division.Level != 0 {
				continue
			}
			for _, alias := range []string{division.Name, division.FullName} {
				index.provinces[geoIPChineseKey(alias)] = division
			}
		}
		chineseGeoIPIndexData = index
	})
	return &chineseGeoIPIndexData
}

func (index *chineseGeoIPIndex) add(parent, alias string, division chineseGeoIPDivision) {
	key := geoIPChineseKey(alias)
	if key == "" {
		return
	}
	for _, existing := range index.children[parent][key] {
		if existing.Code == division.Code {
			return
		}
	}
	index.children[parent][key] = append(index.children[parent][key], division)
}

func geoIPChineseKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(
		"ü", "u", "ǘ", "u", "ǚ", "u", "ǜ", "u",
		"ā", "a", "á", "a", "ǎ", "a", "à", "a",
		"ē", "e", "é", "e", "ě", "e", "è", "e",
		"ī", "i", "í", "i", "ǐ", "i", "ì", "i",
		"ō", "o", "ó", "o", "ǒ", "o", "ò", "o",
		"ū", "u", "ú", "u", "ǔ", "u", "ù", "u",
	)
	name = replacer.Replace(name)
	var result strings.Builder
	for _, character := range name {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			result.WriteRune(character)
		}
	}
	return result.String()
}

func geoIPChineseCandidates(name string) []string {
	key := geoIPChineseKey(name)
	if key == "" {
		return nil
	}
	candidates := []string{key}
	for _, suffix := range []string{"district", "county", "city", "prefecture", "qu", "xian", "shi"} {
		if strings.HasSuffix(key, suffix) && len(key) > len(suffix) {
			candidates = append(candidates, strings.TrimSuffix(key, suffix))
		}
	}
	return candidates
}

func (index *chineseGeoIPIndex) child(parent, name string) (chineseGeoIPDivision, bool) {
	for _, candidate := range geoIPChineseCandidates(name) {
		matches := index.children[parent][candidate]
		if len(matches) == 1 {
			return matches[0], true
		}
	}
	return chineseGeoIPDivision{}, false
}

func (index *chineseGeoIPIndex) province(name, isoCode string) (chineseGeoIPDivision, bool) {
	if code, found := chineseProvinceISOCodes[strings.ToUpper(strings.TrimSpace(isoCode))]; found {
		if province, found := index.byCode[code]; found {
			return province, true
		}
	}
	for _, candidate := range geoIPChineseCandidates(name) {
		if province, found := index.provinces[candidate]; found {
			return province, true
		}
	}
	return chineseGeoIPDivision{}, false
}

func splitChineseGeoIPCity(name string) (string, string) {
	name = strings.TrimSpace(name)
	open := strings.LastIndex(name, "(")
	close := strings.LastIndex(name, ")")
	if open > 0 && close == len(name)-1 && close > open+1 {
		return strings.TrimSpace(name[:open]), strings.TrimSpace(name[open+1 : close])
	}
	return name, ""
}

func (index *chineseGeoIPIndex) districtInProvince(provinceCode, name string) (chineseGeoIPDivision, chineseGeoIPDivision, bool) {
	var foundDistrict, foundCity chineseGeoIPDivision
	found := false
	for _, division := range index.byCode {
		if division.Level != 1 || division.ParentCode != provinceCode {
			continue
		}
		district, ok := index.child(division.Code, name)
		if !ok {
			continue
		}
		if found && foundDistrict.Code != district.Code {
			return chineseGeoIPDivision{}, chineseGeoIPDivision{}, false
		}
		foundDistrict, foundCity, found = district, division, true
	}
	return foundDistrict, foundCity, found
}

var chineseProvinceAliases = map[string]string{
	"Beijing": "11", "Tianjin": "12", "Hebei": "13", "Shanxi": "14",
	"Inner Mongolia": "15", "Nei Mongol": "15", "Liaoning": "21",
	"Jilin": "22", "Heilongjiang": "23", "Shanghai": "31",
	"Jiangsu": "32", "Zhejiang": "33", "Anhui": "34", "Fujian": "35",
	"Jiangxi": "36", "Shandong": "37", "Henan": "41", "Hubei": "42",
	"Hunan": "43", "Guangdong": "44", "Guangxi": "45", "Hainan": "46",
	"Chongqing": "50", "Sichuan": "51", "Guizhou": "52", "Yunnan": "53",
	"Tibet": "54", "Xizang": "54", "Shaanxi": "61", "Gansu": "62",
	"Qinghai": "63", "Ningxia": "64", "Xinjiang": "65",
	"Taiwan": "71", "Hong Kong": "81", "Macau": "82", "Macao": "82",
}

var chineseProvinceISOCodes = map[string]string{
	"BJ": "11", "TJ": "12", "HE": "13", "SX": "14", "NM": "15",
	"LN": "21", "JL": "22", "HL": "23", "SH": "31", "JS": "32",
	"ZJ": "33", "AH": "34", "FJ": "35", "JX": "36", "SD": "37",
	"HA": "41", "HB": "42", "HN": "43", "GD": "44", "GX": "45",
	"HI": "46", "CQ": "50", "SC": "51", "GZ": "52", "YN": "53",
	"XZ": "54", "SN": "61", "GS": "62", "QH": "63", "NX": "64",
	"XJ": "65", "TW": "71", "HK": "81", "MO": "82",
}

// chinaGeoIPDescription preserves the hierarchy of the original city lookup.
// A district is used only when it belongs to the matched city or has a unique
// match within the reported province.
func chinaGeoIPDescription(record geoIPRecord) string {
	if record.Country.ISOCode != "CN" && geoIPChineseKey(record.Country.Names["en"]) != "china" {
		return ""
	}
	parts := []string{"中国"}
	if len(record.Subdivisions) == 0 {
		return appendGeoIPName(parts, geoIPPreferredName(record.City.Names))
	}
	subdivision := record.Subdivisions[0]
	province, found := chineseLocationIndex().province(
		geoIPPreferredName(subdivision.Names), subdivision.ISOCode)
	if !found {
		parts = appendGeoIPName(parts, geoIPPreferredName(subdivision.Names))
		return appendGeoIPName(parts, geoIPPreferredName(record.City.Names))
	}
	parts = append(parts, province.Name)
	cityName := geoIPPreferredName(record.City.Names)
	if cityName == "" {
		return strings.Join(parts, " · ")
	}
	cityText, districtText := splitChineseGeoIPCity(cityName)
	index := chineseLocationIndex()
	city, found := index.child(province.Code, cityText)
	if !found {
		if district, parent, ok := index.districtInProvince(province.Code, cityText); ok {
			parts = appendGeoIPName(parts, parent.Name)
			return strings.Join(appendGeoIPName(parts, district.FullName), " · ")
		}
		return strings.Join(appendGeoIPName(parts, cityName), " · ")
	}
	parts = appendGeoIPName(parts, city.Name)
	if districtText != "" {
		if district, ok := index.child(city.Code, districtText); ok {
			parts = appendGeoIPName(parts, district.FullName)
		} else {
			parts = appendGeoIPName(parts, districtText)
		}
	}
	return strings.Join(parts, " · ")
}

func appendGeoIPName(parts []string, name string) []string {
	name = strings.TrimSpace(name)
	if name == "" || (len(parts) > 0 && parts[len(parts)-1] == name) {
		return parts
	}
	return append(parts, name)
}

func geoIPPreferredName(names map[string]string) string {
	for _, language := range []string{"zh-CN", "zh-Hans", "zh", "en"} {
		if name := strings.TrimSpace(names[language]); name != "" {
			return pushShort(name, 60)
		}
	}
	return ""
}
