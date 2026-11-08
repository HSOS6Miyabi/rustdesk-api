package model

// DevicePolicyUserAssignment applies a strategy to all devices owned by a user.
// A device-specific assignment takes precedence.
type DevicePolicyUserAssignment struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;uniqueIndex"`
	PolicyId uint `json:"policy_id" gorm:"not null;index"`
	Revision int64 `json:"revision" gorm:"not null;default:0"`
	TimeModel
}

// DevicePolicyGroupAssignment applies a strategy to all devices in a device group.
// User-specific and device-specific assignments take precedence.
type DevicePolicyGroupAssignment struct {
	IdModel
	GroupId uint `json:"group_id" gorm:"not null;uniqueIndex"`
	PolicyId uint `json:"policy_id" gorm:"not null;index"`
	Revision int64 `json:"revision" gorm:"not null;default:0"`
	TimeModel
}
