package model

// SMTPSetting stores the outbound email configuration for this installation.
type SMTPSetting struct {
	IdModel
	Enabled bool `json:"enabled" gorm:"not null;default:false"`
	Host string `json:"host" gorm:"not null;size:253"`
	Port int `json:"port" gorm:"not null;default:587"`
	Security string `json:"security" gorm:"not null;size:20"`
	Username string `json:"username" gorm:"not null;size:253"`
	PasswordCiphertext string `json:"-" gorm:"type:text;not null"`
	SenderEmail string `json:"sender_email" gorm:"not null;size:253"`
	SenderName string `json:"sender_name" gorm:"not null;size:100"`
	BaseURL string `json:"base_url" gorm:"not null;size:512"`
	TimeModel
}
