package service

import (
	"errors"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// RevokeDeviceSessions expires every session tied to a device whose owner changes.
// An empty UUID must never be used as a database filter for other devices.
func RevokeDeviceSessions(tx *gorm.DB, deviceID, deviceUUID string) error {
	if tx == nil {
		return errors.New("database connection is required")
	}
	deviceID = strings.TrimSpace(deviceID)
	deviceUUID = strings.TrimSpace(deviceUUID)
	if deviceID == "" && deviceUUID == "" {
		return nil
	}
	if !tx.Migrator().HasTable(&model.UserToken{}) {
		return nil
	}
	query := tx.Model(&model.UserToken{})
	if deviceID == "" {
		query = query.Where("device_uuid = ?", deviceUUID)
	} else {
		query = query.Where("device_id = ?", deviceID)
		if deviceUUID != "" {
			query = query.Or("device_uuid = ?", deviceUUID)
		}
	}
	return query.Delete(&model.UserToken{}).Error
}
