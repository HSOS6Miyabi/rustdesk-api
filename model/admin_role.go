package model

// AdminRole grants selected management operations. Global roles apply to the
// whole installation. Group and individual roles provide bounded directory access.
type AdminRole struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	Permissions string `json:"permissions" gorm:"type:text;not null"`
	ScopeType string `json:"scope_type" gorm:"not null;size:24;default:'global'"`
	ScopeId uint `json:"scope_id" gorm:"not null;default:0"`
	Enabled bool `json:"enabled" gorm:"not null;default:true"`
	TimeModel
}

// AdminRoleAssignment grants a management role to an existing user.
type AdminRoleAssignment struct {
	IdModel
	UserId uint `json:"user_id" gorm:"not null;uniqueIndex:idx_admin_role_grant,priority:1;index"`
	RoleId uint `json:"role_id" gorm:"not null;uniqueIndex:idx_admin_role_grant,priority:2;index"`
	TimeModel
}
