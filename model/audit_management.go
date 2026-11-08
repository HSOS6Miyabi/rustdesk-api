package model

// AuditOperation records changes made by authenticated users without storing request bodies or secrets.
type AuditOperation struct {
	IdModel
	UserId uint `json:"user_id" gorm:"index;not null"`
	Account string `json:"account" gorm:"not null;default:''"`
	Method string `json:"method" gorm:"not null"`
	Path string `json:"path" gorm:"not null;index"`
	ClientIp string `json:"client_ip" gorm:"not null;default:''"`
	UserAgent string `json:"user_agent" gorm:"not null;default:''"`
	HttpStatus int `json:"http_status" gorm:"not null"`
	Success bool `json:"success" gorm:"not null"`
	TimeModel
}

// AuditRetentionPolicy specifies how many days each log category is preserved.
// A value of zero retains the category without automatic deletion.
type AuditRetentionPolicy struct {
	IdModel
	ConnectionDays int `json:"connection_days" gorm:"not null;default:0"`
	FileDays int `json:"file_days" gorm:"not null;default:0"`
	AlarmDays int `json:"alarm_days" gorm:"not null;default:0"`
	LoginDays int `json:"login_days" gorm:"not null;default:0"`
	OperationDays int `json:"operation_days" gorm:"not null;default:0"`
	TimeModel
}
