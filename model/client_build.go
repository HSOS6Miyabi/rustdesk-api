package model

// ClientBuildProfile stores settings for a verified official client installer.
// Windows uses an embedded filename; Linux and macOS use a configuration script.
type ClientBuildProfile struct {
	IdModel
	Name string `json:"name" gorm:"size:100;not null"`
	Platform string `json:"platform" gorm:"size:30;not null"`
	Version string `json:"version" gorm:"size:30;not null"`
	Host string `json:"host" gorm:"size:180;not null"`
	PublicKey string `json:"public_key" gorm:"size:100;not null"`
	APIServer string `json:"api_server" gorm:"size:200;not null"`
	RelayServer string `json:"relay_server" gorm:"size:180;not null"`
	TimeModel
}

// ClientBuildTask freezes the profile and official artifact version when
// requested, so later profile edits cannot change an already queued build.
type ClientBuildTask struct {
	IdModel
	ProfileID uint `json:"profile_id" gorm:"index;not null"`
	RequestedBy uint `json:"requested_by" gorm:"index;not null"`
	Platform string `json:"platform" gorm:"size:30;not null"`
	Version string `json:"version" gorm:"size:30;not null"`
	ConfigSnapshot string `json:"-" gorm:"type:text;not null"`
	Status string `json:"status" gorm:"size:24;not null;index"`
	Filename string `json:"filename" gorm:"size:255;not null"`
	Sha256 string `json:"sha256" gorm:"size:64;not null"`
	Bytes int64 `json:"bytes" gorm:"not null;default:0"`
	Error string `json:"error" gorm:"size:512;not null"`
	StartedAt int64 `json:"started_at" gorm:"not null;default:0"`
	FinishedAt int64 `json:"finished_at" gorm:"not null;default:0"`
	TimeModel
}
