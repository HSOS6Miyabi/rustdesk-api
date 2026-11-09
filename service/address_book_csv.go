package service

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/model/custom_types"
	"gorm.io/gorm"
)

const AddressBookCSVMaxEntries = 5000

var addressBookCSVColumns = []string{
	"id", "alias", "hostname", "username", "platform",
	"collection_id", "tags", "note", "force_always_relay",
}

type AddressBookImportResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

type addressBookImportRow struct {
	ID, Alias, Hostname, Username, Platform, Note string
	CollectionID uint
	Tags []string
	ForceRelay bool
}

func escapeSpreadsheetValue(value string) string {
	check := strings.TrimLeft(value, " \t\r\n")
	if len(check) > 0 && strings.ContainsRune("=+-@", rune(check[0])) {
		return "'" + value
	}
	return value
}

func decodeSpreadsheetValue(value string) string {
	if strings.HasPrefix(value, "'") && escapeSpreadsheetValue(value[1:]) == value {
		return value[1:]
	}
	return value
}

func ExportAddressBookCSV(ownerID uint, writer io.Writer) error {
	if ownerID == 0 { return errors.New("account ID is required") }
	var entries []model.AddressBook
	if err := DB.Where("user_id = ?", ownerID).
		Order("collection_id ASC, id ASC, row_id ASC").
		Limit(AddressBookCSVMaxEntries + 1).Find(&entries).Error; err != nil {
		return err
	}
	if len(entries) > AddressBookCSVMaxEntries {
		return fmt.Errorf("one CSV export supports at most %d address book entries", AddressBookCSVMaxEntries)
	}
	// UTF-8 BOM enables Microsoft Excel to recognize Chinese text.
	if _, err := io.WriteString(writer, "\ufeff"); err != nil { return err }
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(addressBookCSVColumns); err != nil { return err }
	for _, item := range entries {
		var tags []string
		if len(item.Tags) > 0 {
			if err := json.Unmarshal(item.Tags, &tags); err != nil { return err }
		}
		row := []string{
			item.Id, item.Alias, item.Hostname, item.Username, item.Platform,
			strconv.FormatUint(uint64(item.CollectionId), 10), strings.Join(tags, ";"),
			item.Note, strconv.FormatBool(item.ForceAlwaysRelay),
		}
		for i := range row { row[i] = escapeSpreadsheetValue(row[i]) }
		if err := csvWriter.Write(row); err != nil { return err }
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func validCSVField(value string, limit int) bool {
	return len(value) <= limit && !strings.ContainsAny(value, "\x00")
}

func parseAddressBookCSV(reader io.Reader) ([]addressBookImportRow, error) {
	csvReader := csv.NewReader(io.LimitReader(reader, 4<<20))
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = false
	header, err := csvReader.Read()
	if err != nil { return nil, fmt.Errorf("cannot read CSV header: %w", err) }
	if len(header) > 0 { header[0] = strings.TrimPrefix(header[0], "\ufeff") }
	columns := make(map[string]int)
	allowed := make(map[string]bool)
	for _, key := range addressBookCSVColumns { allowed[key] = true }
	for i, item := range header {
		name := strings.ToLower(strings.TrimSpace(item))
		if !allowed[name] || columns[name] != 0 {
			if !allowed[name] { return nil, fmt.Errorf("unknown CSV column: %s", name) }
			return nil, fmt.Errorf("duplicate CSV column: %s", name)
		}
		columns[name] = i + 1
	}
	if columns["id"] == 0 { return nil, errors.New("CSV must contain an id column") }
	data := make([]addressBookImportRow, 0)
	seen := make(map[string]bool)
	for {
		record, err := csvReader.Read()
		if errors.Is(err, io.EOF) { break }
		if err != nil { return nil, fmt.Errorf("cannot read CSV row %d: %w", len(data)+2, err) }
		if len(data) >= AddressBookCSVMaxEntries {
			return nil, fmt.Errorf("one CSV import supports at most %d entries", AddressBookCSVMaxEntries)
		}
		if len(record) != len(header) { return nil, fmt.Errorf("CSV row %d has an unexpected number of columns", len(data)+2) }
		get := func(name string) string {
			idx := columns[name]
			if idx == 0 { return "" }
			return decodeSpreadsheetValue(record[idx-1])
		}
		row := addressBookImportRow{
			ID: strings.TrimSpace(get("id")),
			Alias: get("alias"), Hostname: get("hostname"),
			Username: get("username"), Platform: get("platform"), Note: get("note"),
		}
		if row.ID == "" || !validCSVField(row.ID, 253) ||
			strings.ContainsAny(row.ID, "\r\n\t") ||
			!validCSVField(row.Alias, 256) || !validCSVField(row.Hostname, 256) ||
			!validCSVField(row.Username, 256) || !validCSVField(row.Platform, 128) ||
			!validCSVField(row.Note, 2048) {
			return nil, fmt.Errorf("invalid field in CSV row %d", len(data)+2)
		}
		if value := strings.TrimSpace(get("collection_id")); value != "" {
			n, err := strconv.ParseUint(value, 10, 32)
			if err != nil { return nil, fmt.Errorf("invalid collection ID in row %d", len(data)+2) }
			row.CollectionID = uint(n)
		}
		if raw := get("tags"); raw != "" {
			for _, part := range strings.Split(raw, ";") {
				tag := strings.TrimSpace(part)
				if tag != "" { row.Tags = append(row.Tags, tag) }
			}
			if len(row.Tags) > 32 { return nil, fmt.Errorf("too many tags in row %d", len(data)+2) }
			for _, tag := range row.Tags {
				if !validCSVField(tag, 80) { return nil, fmt.Errorf("invalid tag in row %d", len(data)+2) }
			}
		}
		if value := strings.TrimSpace(get("force_always_relay")); value != "" {
			enabled, err := strconv.ParseBool(value)
			if err != nil { return nil, fmt.Errorf("invalid relay setting in row %d", len(data)+2) }
			row.ForceRelay = enabled
		}
		key := row.ID + "\x00" + strconv.FormatUint(uint64(row.CollectionID), 10)
		if seen[key] { return nil, fmt.Errorf("duplicate device ID in CSV row %d", len(data)+2) }
		seen[key] = true
		data = append(data, row)
	}
	if len(data) == 0 { return nil, errors.New("CSV contains no address book entries") }
	return data, nil
}

// ImportAddressBookCSV validates every row before saving. Existing passwords,
// connection hashes and other client settings remain unchanged.
func ImportAddressBookCSV(ownerID uint, reader io.Reader) (AddressBookImportResult, error) {
	var result AddressBookImportResult
	if ownerID == 0 { return result, errors.New("account ID is required") }
	rows, err := parseAddressBookCSV(reader)
	if err != nil { return result, err }
	err = DB.Transaction(func(tx *gorm.DB) error {
		var ownerCount int64
		if err := tx.Model(&model.User{}).Where("id = ?",ownerID).Count(&ownerCount).Error;err!=nil{return err}
		if ownerCount!=1{return errors.New("account does not exist")}
		collections := make(map[uint]bool)
		for _, row := range rows {
			if row.CollectionID == 0 || collections[row.CollectionID] { continue }
			var count int64
			if err := tx.Model(&model.AddressBookCollection{}).
				Where("id = ? AND user_id = ?", row.CollectionID, ownerID).
				Count(&count).Error; err != nil { return err }
			if count != 1 { return fmt.Errorf("address book collection %d does not belong to this account", row.CollectionID) }
			collections[row.CollectionID] = true
		}
		for _, row := range rows {
			encoded, err := json.Marshal(row.Tags)
			if err != nil { return err }
			var existing model.AddressBook
			err = tx.Where("user_id = ? AND collection_id = ? AND id = ?",
				ownerID, row.CollectionID, row.ID).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { return err }
			if errors.Is(err, gorm.ErrRecordNotFound) {
				newEntry := model.AddressBook{
					Id: row.ID, UserId: ownerID, CollectionId: row.CollectionID,
					Alias: row.Alias, Hostname: row.Hostname, Username: row.Username,
					Platform: row.Platform, Note: row.Note,
					Tags: custom_types.AutoJson(encoded), ForceAlwaysRelay: row.ForceRelay,
				}
				if err := tx.Create(&newEntry).Error; err != nil { return err }
				result.Created++
			} else {
				updates := map[string]interface{}{
					"alias":row.Alias, "hostname":row.Hostname,
					"username":row.Username, "platform":row.Platform,
					"note":row.Note, "tags":string(encoded),
					"force_always_relay":row.ForceRelay,
				}
				if err := tx.Model(&existing).Updates(updates).Error; err != nil { return err }
				result.Updated++
			}
		}
		return nil
	})
	if err != nil { return AddressBookImportResult{}, err }
	return result, nil
}
