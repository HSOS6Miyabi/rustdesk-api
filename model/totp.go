package model

// UserTOTP contains the encrypted authenticator secret for an account.
type UserTOTP struct {
	IdModel
	UserId uint `json:"user_id" gorm:"uniqueIndex;not null"`
	SecretCiphertext string `json:"-" gorm:"type:text;not null"`
	PendingCiphertext string `json:"-" gorm:"type:text;not null"`
	PendingExpiresAt int64 `json:"-" gorm:"not null;default:0"`
	Enabled bool `json:"enabled" gorm:"not null;default:false"`
	LastUsedStep int64 `json:"-" gorm:"not null;default:0"`
	TimeModel
}

// TOTPRecoveryCode is a one-use fallback code, stored as a hash.
type TOTPRecoveryCode struct {
	IdModel
	UserId uint `json:"user_id" gorm:"index;not null"`
	CodeHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	UsedAt int64 `json:"used_at" gorm:"not null;default:0"`
	TimeModel
}

// TOTPLoginChallenge connects an authenticated password step to the second factor.
type TOTPLoginChallenge struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;index"`
	TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;index"`
	UsedAt int64 `json:"used_at" gorm:"not null;default:0"`
	Attempts int `json:"attempts" gorm:"not null;default:0"`
	TimeModel
}
