package model

// EmailLoginSetting enables optional email verification for password logins.
type EmailLoginSetting struct {
	IdModel
	UserId uint `json:"user_id" gorm:"uniqueIndex;not null"`
	VerifiedEmail string `json:"verified_email" gorm:"not null;size:253"`
	Enabled bool `json:"enabled" gorm:"not null;default:false"`
	TimeModel
}

// EmailCodeChallenge stores a short-lived code sent to the account's email.
type EmailCodeChallenge struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;index"`
	Email string `json:"-" gorm:"not null;size:253"`
	Purpose string `json:"-" gorm:"not null;size:16;index"`
	TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
	CodeHash string `json:"-" gorm:"not null;size:64"`
	IssuedAt int64 `json:"issued_at" gorm:"not null;index"`
	ExpiresAt int64 `json:"expires_at" gorm:"not null;index"`
	ConsumedAt int64 `json:"consumed_at" gorm:"not null;default:0"`
	Attempts int `json:"attempts" gorm:"not null;default:0"`
	TimeModel
}
