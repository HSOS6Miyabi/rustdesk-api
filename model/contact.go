package model

// Contact is a unilateral grant. OwnerUserId exposes their devices to
// RecipientUserId without requiring approval.
type Contact struct {
	OwnerUserId uint `json:"owner_user_id" gorm:"primaryKey;not null"`
	RecipientUserId uint `json:"recipient_user_id" gorm:"primaryKey;not null;index:idx_contact_recipient"`
}
