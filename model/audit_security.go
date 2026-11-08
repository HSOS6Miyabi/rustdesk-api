package model

// AuditEventNonce prevents retries from generating duplicate audit records.
type AuditEventNonce struct {
	Nonce string `json:"nonce" gorm:"primaryKey;size:128"`
	CreatedAt int64 `json:"created_at" gorm:"not null;index"`
}

// AuditAlarm represents a security event emitted by a RustDesk client.
type AuditAlarm struct {
	IdModel
	PeerId string `json:"id" gorm:"not null;index"`
	Uuid string `json:"uuid" gorm:"default:'';not null"`
	Typ int `json:"typ" gorm:"not null;index"`
	Info string `json:"info" gorm:"type:text;not null"`
	ConnId int64 `json:"conn_id" gorm:"not null;default:0;index"`
	ConnAuditRef string `json:"conn_audit_ref,omitempty" gorm:"default:'';not null;size:256"`
	FromUserId uint `json:"from_user_id" gorm:"default:0;not null;index"`
	ToUserId uint `json:"to_user_id" gorm:"default:0;not null;index"`
	SourceIp string `json:"source_ip" gorm:"default:'';not null"`
	TimeModel
}
