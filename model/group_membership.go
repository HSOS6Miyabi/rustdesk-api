package model

// GroupMembership represents additional group memberships beyond the user's primary group.
type GroupMembership struct {
	GroupId uint `json:"group_id" gorm:"primaryKey;not null"`
	UserId uint `json:"user_id" gorm:"primaryKey;not null"`
}

// GroupAnchor identifies public users visible to members of an anchor group.
type GroupAnchor struct {
	GroupId uint `json:"group_id" gorm:"primaryKey;not null"`
	UserId uint `json:"user_id" gorm:"primaryKey;not null"`
}
