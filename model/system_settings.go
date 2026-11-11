package model

import "time"

type SystemSettingOverride struct {
    Key string `gorm:"primaryKey;size:128" json:"key"`
    Enabled bool `gorm:"not null;default:false" json:"enabled"`
    Value string `gorm:"type:text;not null" json:"-"`
    UpdatedAt time.Time `json:"updated_at"`
}

type SystemSettingRevision struct {
    ID uint `gorm:"primaryKey"`
    Version int64 `gorm:"not null"`
}

type SystemSettingHistory struct {
    ID uint `gorm:"primaryKey" json:"id"`
    ActorID uint `gorm:"not null;index" json:"actor_id"`
    SettingKey string `gorm:"not null;size:128;index" json:"setting_key"`
    Enabled bool `gorm:"not null" json:"enabled"`
    Storage string `gorm:"size:16;not null" json:"storage"`
    ChangedAt time.Time `gorm:"not null;index" json:"changed_at"`
}
