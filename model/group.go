package model

const (
	GroupTypeDefault = 1 // 默认
	GroupTypeShare   = 2 // 共享
	GroupTypeAnchor  = 3 // 公共锚点
)

type Group struct {
	IdModel
	Name string `json:"name" gorm:"default:'';not null;"`
	Type int    `json:"type" gorm:"default:1;not null;"`
	OwnerUserId uint `json:"owner_user_id" gorm:"default:0;not null;index"`
	TimeModel
	MemberCount int64 `json:"member_count" gorm:"-"`
	DeviceCount int64 `json:"device_count" gorm:"-"`
	AnchorCount int64 `json:"anchor_count" gorm:"-"`
}

type GroupList struct {
	Groups []*Group `json:"list"`
	Pagination
}

type DeviceGroup struct {
	IdModel
	Name string `json:"name" gorm:"default:'';not null;"`
	TimeModel
	DeviceCount int64 `json:"device_count" gorm:"-"`
	PolicyName string `json:"policy_name" gorm:"-"`
	ControlRoleName string `json:"control_role_name" gorm:"-"`
}

type DeviceGroupList struct {
	DeviceGroups []*DeviceGroup `json:"list"`
	Pagination
}
