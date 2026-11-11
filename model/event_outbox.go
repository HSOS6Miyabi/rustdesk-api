package model

// PushEventOutbox stores accepted non-heartbeat events until the corresponding
// delivery records are committed. Recovery resumes processing after restart.
type PushEventOutbox struct {
	IdModel
	Payload string `gorm:"type:text;not null"`
	LeaseUntil int64 `gorm:"not null;default:0;index"`
	LeaseOwner string `gorm:"size:40;not null;default:''"`
	Status string `gorm:"not null;size:16;default:pending;index"`
	Attempts int `gorm:"not null;default:0"`
	NextAttemptAt int64 `gorm:"not null;default:0;index"`
	LastError string `gorm:"not null;size:240;default:''"`
	TimeModel
}

// NotificationEventOutbox stores accepted administrative alerts before the
// delivery table is populated.
type NotificationEventOutbox struct {
	IdModel
	Event string `gorm:"size:64;not null"`
	Reference string `gorm:"size:512;not null"`
	Description string `gorm:"type:text;not null"`
	LeaseUntil int64 `gorm:"not null;default:0;index"`
	LeaseOwner string `gorm:"size:40;not null;default:''"`
	Status string `gorm:"not null;size:16;default:pending;index"`
	Attempts int `gorm:"not null;default:0"`
	NextAttemptAt int64 `gorm:"not null;default:0;index"`
	LastError string `gorm:"not null;size:240;default:''"`
	TimeModel
}
