package model

import "errors"

// ControlRole contains native RustDesk connection permissions.
// 0 inherits the device setting, 1 prohibits the operation, and 2 allows it.
// Each field occupies two bits, matching RustDesk's ControlPermissions protocol.
type ControlRole struct {
	IdModel
	Name string `json:"name" gorm:"not null;size:100"`
	Keyboard int `json:"keyboard" gorm:"not null;default:0"`
	RemotePrinter int `json:"remote_printer" gorm:"not null;default:0"`
	Clipboard int `json:"clipboard" gorm:"not null;default:0"`
	File int `json:"file" gorm:"not null;default:0"`
	Audio int `json:"audio" gorm:"not null;default:0"`
	Camera int `json:"camera" gorm:"not null;default:0"`
	Terminal int `json:"terminal" gorm:"not null;default:0"`
	Tunnel int `json:"tunnel" gorm:"not null;default:0"`
	Restart int `json:"restart" gorm:"not null;default:0"`
	Recording int `json:"recording" gorm:"not null;default:0"`
	BlockInput int `json:"block_input" gorm:"not null;default:0"`
	RemoteModify int `json:"remote_modify" gorm:"not null;default:0"`
	PrivacyMode int `json:"privacy_mode" gorm:"not null;default:0"`
	USB int `json:"usb" gorm:"not null;default:0"`
	TimeModel
}

func (role *ControlRole) PermissionBits() (uint64,error) {
	if role==nil{return 0,errors.New("control role is required")}
	states:=[]int{
		role.Keyboard,role.RemotePrinter,role.Clipboard,role.File,
		role.Audio,role.Camera,role.Terminal,role.Tunnel,
		role.Restart,role.Recording,role.BlockInput,role.RemoteModify,
		role.PrivacyMode,role.USB,
	}
	var bits uint64
	for index,state:=range states{
		if state<0||state>2{return 0,errors.New("control permission must be 0, 1, or 2")}
		bits|=uint64(state)<<(2*index)
	}
	return bits,nil
}

// ControlRoleAssignment applies one role to a device, its owner, or its group.
type ControlRoleAssignment struct {
	IdModel
	TargetType string `json:"target_type" gorm:"not null;size:20;uniqueIndex:idx_control_target,priority:1"`
	TargetId string `json:"target_id" gorm:"not null;size:253;uniqueIndex:idx_control_target,priority:2"`
	RoleId uint `json:"role_id" gorm:"not null;index"`
	TimeModel
}
