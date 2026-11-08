package model

// DevicePolicy stores client settings that can be applied by managed RustDesk installations.
type DevicePolicy struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	IdWhitelist string `json:"id_whitelist" gorm:"type:text;not null"`
	IpWhitelist string `json:"ip_whitelist" gorm:"type:text;not null"`
	EnableWebRTC bool `json:"enable_webrtc" gorm:"not null;default:false"`
	TrustDurationDays int `json:"trust_duration_days" gorm:"not null;default:0"`
	TimeModel
}

// DevicePolicyAssignment identifies the policy selected for one registered device.
type DevicePolicyAssignment struct {
	IdModel
	DeviceId string `json:"device_id" gorm:"not null;uniqueIndex;size:128"`
	PolicyId uint `json:"policy_id" gorm:"not null;index"`
	TimeModel
}
