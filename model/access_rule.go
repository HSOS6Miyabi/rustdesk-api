package model

// AccessRule describes which signed-in accounts may request a remote device
// connection when optional server-side access rules are enabled.
type AccessRule struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	SourceType string `json:"source_type" gorm:"not null;size:16;index"`
	SourceID uint `json:"source_id" gorm:"not null;default:0;index"`
	TargetType string `json:"target_type" gorm:"not null;size:24;index"`
	TargetID string `json:"target_id" gorm:"not null;size:253;index"`
	Effect string `json:"effect" gorm:"not null;size:8"`
	Enabled bool `json:"enabled" gorm:"not null;default:true"`
	TimeModel
}
