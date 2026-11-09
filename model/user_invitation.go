package model

// UserInvitation contains an email invitation before the recipient registers.
type UserInvitation struct {
	IdModel
	Email string `json:"email" gorm:"not null;size:253;index"`
	GroupId uint `json:"group_id" gorm:"not null;index"`
	CreatedBy uint `json:"created_by" gorm:"not null;index"`
	TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;index"`
	AcceptedAt int64 `json:"accepted_at" gorm:"not null;default:0"`
	RevokedAt int64 `json:"revoked_at" gorm:"not null;default:0"`
	TimeModel
}
