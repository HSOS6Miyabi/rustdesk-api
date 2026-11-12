package model

// UserPasskey contains an authenticated public key credential. Never expose the
// serialized credential record to clients.
type UserPasskey struct {
    IdModel
    UserId uint `json:"user_id" gorm:"not null;index"`
    CredentialID string `json:"-" gorm:"not null;uniqueIndex;size:1024"`
    CredentialJSON string `json:"-" gorm:"type:text;not null"`
    RPID string `json:"-" gorm:"not null;size:253"`
    Name string `json:"name" gorm:"not null;size:80"`
    LastUsedAt int64 `json:"last_used_at" gorm:"not null;default:0"`
    TimeModel
}

// UserPasskeyHandle is an opaque, stable user handle scoped to the account.
// This prevents exposing account numbers to authenticators.
type UserPasskeyHandle struct {
    IdModel
    UserId uint `json:"-" gorm:"not null;uniqueIndex"`
    Handle string `json:"-" gorm:"not null;uniqueIndex;size:64"`
    TimeModel
}

// UserPasskeyChallenge persists a short-lived, single-use ceremony session.
// SessionJSON is never sent back to the client.
type UserPasskeyChallenge struct {
    IdModel
    UserId uint `json:"-" gorm:"not null;index"`
    Purpose string `json:"-" gorm:"not null;size:24"`
    TokenHash string `json:"-" gorm:"not null;uniqueIndex;size:64"`
    SessionJSON string `json:"-" gorm:"not null;type:text"`
    Origin string `json:"-" gorm:"not null;size:255"`
    IPHash string `json:"-" gorm:"not null;index;size:64"`
    ExpiresAt int64 `json:"-" gorm:"not null;index"`
    UsedAt int64 `json:"-" gorm:"not null;default:0"`
    TimeModel
}
