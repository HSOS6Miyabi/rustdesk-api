package model

// APIToken represents a revocable token used for automated administration.
type APIToken struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;index"`
	Name string `json:"name" gorm:"not null;size:100"`
	Prefix string `json:"prefix" gorm:"not null;size:24"`
	TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	Permissions string `json:"permissions" gorm:"type:text;not null"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;default:0"`
	RevokedAt int64 `json:"revoked_at" gorm:"not null;default:0"`
	LastUsedAt int64 `json:"last_used_at" gorm:"not null;default:0"`
	TimeModel
}
