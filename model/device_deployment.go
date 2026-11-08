package model

// DeviceDeployment stores approved device identity information for optional managed registration.
type DeviceDeployment struct {
	IdModel
	DeviceId    string `json:"device_id" gorm:"uniqueIndex;size:253;not null"`
	Uuid        string `json:"uuid" gorm:"size:128;not null"`
	PublicKey   string `json:"-" gorm:"size:128;not null"`
	OwnerUserId uint   `json:"owner_user_id" gorm:"index;not null"`
	Status      string `json:"status" gorm:"size:24;not null;default:approved"`
	TimeModel
}
