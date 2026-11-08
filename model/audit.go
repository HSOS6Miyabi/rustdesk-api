package model

const (
	AuditActionNew   = "new"
	AuditActionClose = "close"
)

type AuditConn struct {
	IdModel
	Action    string `json:"action" gorm:"default:'';not null;"`
	ConnId    int64  `json:"conn_id" gorm:"default:0;not null;index;index:idx_audit_conn_peer_connection,priority:2"`
	PeerId    string `json:"peer_id" gorm:"default:'';not null;index;index:idx_audit_conn_peer_connection,priority:1"`
	FromPeer  string `json:"from_peer" gorm:"default:'';not null;index"`
	FromName  string `json:"from_name" gorm:"default:'';not null;"`
	FromUserId uint `json:"from_user_id" gorm:"default:0;not null;index"`
	ToUserId uint `json:"to_user_id" gorm:"default:0;not null;index"`
	FromAccount string `json:"from_account" gorm:"default:'';not null;"`
	ToAccount string `json:"to_account" gorm:"default:'';not null;"`
	FromDeviceName string `json:"from_device_name" gorm:"default:'';not null;"`
	ToDeviceName string `json:"to_device_name" gorm:"default:'';not null;"`
	Ip        string `json:"ip" gorm:"default:'';not null;"`
	SessionId string `json:"session_id" gorm:"default:'';not null;"`
	ConnAuditRef string `json:"conn_audit_ref,omitempty" gorm:"default:'';not null;size:256"`
	PrimaryAuth int `json:"primary_auth" gorm:"default:0;not null"`
	TwoFactor int `json:"two_factor" gorm:"default:0;not null"`
	EventNonce string `json:"-" gorm:"-"`
	Type      int    `json:"type" gorm:"default:0;not null;"`
	Uuid      string `json:"uuid" gorm:"default:'';not null;"`
	CloseTime int64  `json:"close_time" gorm:"default:0;not null;"`
	Direction string `json:"direction,omitempty" gorm:"-"`
	TimeModel
}

type AuditConnList struct {
	AuditConns []*AuditConn `json:"list"`
	Pagination
}

type AuditFile struct {
	IdModel
	ConnId int64 `json:"conn_id" gorm:"default:0;not null;index:idx_audit_file_connection"`
	EventNonce string `json:"-" gorm:"-"`
	FromPeer string `json:"from_peer" gorm:"default:'';not null;index"`
	Info     string `json:"info" gorm:"default:'';not null;"`
	IsFile   bool   `json:"is_file" gorm:"default:0;not null;"`
	Path     string `json:"path" gorm:"default:'';not null;"`
	PeerId   string `json:"peer_id" gorm:"default:'';not null;index"`
	Type     int    `json:"type" gorm:"default:0;not null;"`
	Uuid     string `json:"uuid" gorm:"default:'';not null;"`
	Ip       string `json:"ip" gorm:"default:'';not null;"`
	Num      int    `json:"num" gorm:"default:0;not null;"`
	FromName string `json:"from_name" gorm:"default:'';not null;"`
	FromUserId uint `json:"from_user_id" gorm:"default:0;not null;index"`
	ToUserId uint `json:"to_user_id" gorm:"default:0;not null;index"`
	FromAccount string `json:"from_account" gorm:"default:'';not null;"`
	ToAccount string `json:"to_account" gorm:"default:'';not null;"`
	FromDeviceName string `json:"from_device_name" gorm:"default:'';not null;"`
	Direction string `json:"direction,omitempty" gorm:"-"`
	ToDeviceName string `json:"to_device_name" gorm:"default:'';not null;"`
	TimeModel
}

type AuditFileList struct {
	AuditFiles []*AuditFile `json:"list"`
	Pagination
}
