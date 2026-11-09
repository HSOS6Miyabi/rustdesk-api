package model

// RelayNode describes a remote hbbr available to this RustDesk installation.
type RelayNode struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	Address string `json:"address" gorm:"not null;uniqueIndex;size:253"`
	Subnets string `json:"subnets" gorm:"type:text;not null"`
	Countries string `json:"countries" gorm:"type:text;not null;default:''"`
	Priority int `json:"priority" gorm:"not null;default:100"`
	Enabled bool `json:"enabled" gorm:"not null;default:false"`
	TimeModel
}
