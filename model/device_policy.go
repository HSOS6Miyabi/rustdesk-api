package model

// DevicePolicy stores client settings that can be applied by managed RustDesk installations.
type DevicePolicy struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	Status int `json:"status" gorm:"not null;default:1"`
	IdWhitelist string `json:"id_whitelist" gorm:"type:text;not null"`
	IpWhitelist string `json:"ip_whitelist" gorm:"type:text;not null"`
	EnableWebRTC bool `json:"enable_webrtc" gorm:"not null;default:false"`
	TrustDurationDays int `json:"trust_duration_days" gorm:"not null;default:0"`
	AccessMode string `json:"access_mode" gorm:"not null;size:20;default:''"`
	EnableKeyboard string `json:"enable_keyboard" gorm:"not null;size:4;default:''"`
	EnableClipboard string `json:"enable_clipboard" gorm:"not null;size:4;default:''"`
	EnableFileTransfer string `json:"enable_file_transfer" gorm:"not null;size:4;default:''"`
	ApprovalMode string `json:"approval_mode" gorm:"not null;size:24;default:''"`
	ManagedClientKeys string `json:"-" gorm:"size:255;not null;default:''"`
	Revision int64 `json:"revision" gorm:"not null;default:0"`
	TimeModel
}

// DevicePolicyAssignment identifies the policy selected for one registered device.
type DevicePolicyAssignment struct {
	IdModel
	DeviceId string `json:"device_id" gorm:"not null;uniqueIndex;size:128"`
	PolicyId uint `json:"policy_id" gorm:"not null;index"`
	Revision int64 `json:"revision" gorm:"not null;default:0"`
	TimeModel
}
