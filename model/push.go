package model

// PushChannel contains one user's notification destination. Secrets are always
// stored encrypted and excluded from serialized API responses.
type PushChannel struct {
	IdModel
	UserId             uint   `json:"user_id" gorm:"not null;index"`
	Kind               string `json:"kind" gorm:"size:24;not null"`
	Name               string `json:"name" gorm:"size:100;not null"`
	Enabled            bool   `json:"enabled" gorm:"not null;default:false"`
	Events             string `json:"events" gorm:"type:text;not null"`
	GeoIPEnabled       bool   `json:"geoip_enabled" gorm:"not null;default:false"`
	BaseURL            string `json:"base_url" gorm:"size:512;not null;default:''"`
	KeyCiphertext      string `json:"-" gorm:"type:text;not null"`
	SMTPHost           string `json:"smtp_host" gorm:"size:253;not null;default:''"`
	SMTPPort           int    `json:"smtp_port" gorm:"not null;default:587"`
	SMTPSecurity       string `json:"smtp_security" gorm:"size:16;not null;default:'starttls'"`
	SMTPUsername       string `json:"smtp_username" gorm:"size:253;not null;default:''"`
	SMTPPasswordCipher string `json:"-" gorm:"type:text;not null"`
	SMTPFrom           string `json:"smtp_from" gorm:"size:253;not null;default:''"`
	SMTPTo             string `json:"smtp_to" gorm:"size:253;not null;default:''"`
	TimeModel
}

// PushDelivery stores one accepted event per destination, with retry state.
// Delivery rows never contain sending keys or passwords.
type PushDelivery struct {
	IdModel
	ChannelId     uint   `json:"channel_id" gorm:"not null;uniqueIndex:idx_push_event,priority:1;index"`
	UserId        uint   `json:"user_id" gorm:"not null;index"`
	EventType     string `json:"event_type" gorm:"size:64;not null;uniqueIndex:idx_push_event,priority:2"`
	EventKey      string `json:"-" gorm:"size:64;not null;uniqueIndex:idx_push_event,priority:3"`
	Title         string `json:"title" gorm:"size:250;not null"`
	Body          string `json:"body" gorm:"type:text;not null"`
	Status        string `json:"status" gorm:"size:24;not null;index"`
	Attempts      int    `json:"attempts" gorm:"not null;default:0"`
	NextAttemptAt int64  `json:"next_attempt_at" gorm:"not null;index"`
	LockedAt      int64  `json:"-" gorm:"not null;default:0"`
	SentAt        int64  `json:"sent_at" gorm:"not null;default:0"`
	LastError     string `json:"last_error" gorm:"type:text;not null"`
	TimeModel
}

// PushPresence records online state only after an authenticated heartbeat.
type PushPresence struct {
	PeerRowId  uint   `json:"-" gorm:"primaryKey"`
	DeviceId   string `json:"-" gorm:"size:253;not null"`
	Online     bool   `json:"-" gorm:"not null;default:false;index"`
	LastSeenAt int64  `json:"-" gorm:"not null;index"`
	LastIP     string `json:"-" gorm:"size:64;not null;default:''"`
	TimeModel
}
