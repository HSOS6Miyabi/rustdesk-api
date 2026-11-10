package model

// PushEventOutbox stores accepted non-heartbeat events until the corresponding
// delivery records are committed. Recovery resumes processing after restart.
type PushEventOutbox struct {
	IdModel
	Payload string `gorm:"type:text;not null"`
	LeaseUntil int64 `gorm:"not null;default:0"`
	TimeModel
}

// NotificationEventOutbox stores accepted administrative alerts before the
// delivery table is populated.
type NotificationEventOutbox struct {
	IdModel
	Event string `gorm:"size:64;not null"`
	Reference string `gorm:"size:512;not null"`
	Description string `gorm:"type:text;not null"`
	LeaseUntil int64 `gorm:"not null;default:0"`
	TimeModel
}
