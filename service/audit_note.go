package service

import (
	"errors"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

// UpdateConnectionNote saves an administrator's explanation of an audit entry.
// The original session data remains unchanged.
func UpdateConnectionNote(connectionID uint,authorID uint,value string) error {
	if connectionID==0||authorID==0{return errors.New("valid connection and administrator IDs are required")}
	note:=strings.TrimSpace(value)
	if len([]rune(note))>2000{return errors.New("connection note must not exceed 2000 characters")}
	if strings.ContainsRune(note,0){return errors.New("connection note contains invalid characters")}
	result:=DB.Model(&model.AuditConn{}).Where("id = ?",connectionID).
		Updates(map[string]interface{}{"note":note,"note_updated_by":authorID})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}
