package model

// NotificationSetting controls optional administrative email alerts.
type NotificationSetting struct {
	IdModel
	Enabled bool `json:"enabled" gorm:"not null;default:false"`
	Recipients string `json:"recipients" gorm:"type:text;not null"`
	Events string `json:"events" gorm:"type:text;not null"`
	TimeModel
}

// NotificationDelivery records one recipient's delivery result. EventKey keeps
// repeated protocol events from producing multiple identical notifications.
type NotificationDelivery struct {
	IdModel
	EventType string `json:"event_type" gorm:"size:50;not null;uniqueIndex:idx_notification_dedup,priority:1;index"`
	EventKey string `json:"event_key" gorm:"size:80;not null;uniqueIndex:idx_notification_dedup,priority:2"`
	Recipient string `json:"recipient" gorm:"size:253;not null;uniqueIndex:idx_notification_dedup,priority:3"`
	Subject string `json:"subject" gorm:"size:250;not null"`
	Body string `json:"-" gorm:"type:text;not null"`
	Status string `json:"status" gorm:"size:24;not null;index;index:idx_notification_due,priority:1;index:idx_notification_lease,priority:1"`
	Attempts int `json:"attempts" gorm:"not null;default:0"`
	NextAttemptAt int64 `json:"next_attempt_at" gorm:"not null;index;index:idx_notification_due,priority:2"`
	LockedAt int64 `json:"-" gorm:"not null;default:0;index:idx_notification_lease,priority:2"`
	SentAt int64 `json:"sent_at" gorm:"not null;default:0"`
	LastError string `json:"last_error" gorm:"type:text;not null"`
	TimeModel
}
