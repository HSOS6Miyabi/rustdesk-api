package model

// PasswordResetToken is a single-use email recovery request.
type PasswordResetToken struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;index"`
	TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	RequestedAt int64 `json:"requested_at" gorm:"not null;index"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;index"`
	UsedAt int64 `json:"used_at" gorm:"not null;default:0"`
	TimeModel
}
