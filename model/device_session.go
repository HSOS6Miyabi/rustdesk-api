package model

// DeviceActiveConnection represents a session reported by a registered device.
type DeviceActiveConnection struct {
	IdModel
	DeviceId string `json:"device_id" gorm:"not null;size:253;uniqueIndex:idx_device_active_connection,priority:1"`
	ConnId int32 `json:"conn_id" gorm:"not null;uniqueIndex:idx_device_active_connection,priority:2"`
	LastSeenAt int64 `json:"last_seen_at" gorm:"not null;index"`
	TimeModel
}

// DeviceDisconnectRequest records the outcome of a remote session termination.
type DeviceDisconnectRequest struct {
	IdModel
	DeviceId string `json:"device_id" gorm:"not null;size:253;index"`
	ConnId int32 `json:"conn_id" gorm:"not null"`
	RequestedBy uint `json:"requested_by" gorm:"not null;index"`
	Status string `json:"status" gorm:"not null;size:24;index"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;index"`
	DeliveredAt int64 `json:"delivered_at" gorm:"not null;default:0"`
	CompletedAt int64 `json:"completed_at" gorm:"not null;default:0"`
	TimeModel
}
