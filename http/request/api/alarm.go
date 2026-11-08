package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"github.com/lejianwen/rustdesk-api/v2/model"
)

type AuditAlarmForm struct {
	Id string `json:"id"`
	Uuid string `json:"uuid"`
	Typ int `json:"typ"`
	Info json.RawMessage `json:"info"`
	ConnId int64 `json:"conn_id"`
	ConnAuditRef string `json:"conn_audit_ref"`
	Nonce string `json:"nonce"`
}

func (f *AuditAlarmForm) ToAuditAlarm() (*model.AuditAlarm, error) {
	id := strings.TrimSpace(f.Id)
	if len(id)==0 || len(id)>128 || f.Typ<0 || f.Typ>127 {
		return nil,errors.New("invalid alarm device ID or type")
	}
	if len(f.Nonce)>128 || len(f.ConnAuditRef)>256 || len(f.Uuid)>256 || len(f.Info)>8192 {
		return nil, errors.New("alarm parameters exceed their size limits")
	}
	info:=strings.TrimSpace(string(f.Info))
	if info=="" || info=="null" { info="{}" }
	if strings.HasPrefix(info,"\"") {
		var data string
		if err:=json.Unmarshal(f.Info,&data);err!=nil{return nil,err}
		if !json.Valid([]byte(data)) { return nil,errors.New("alarm information is not valid JSON") }
		info=data
	} else if !json.Valid(f.Info) {
		return nil,fmt.Errorf("invalid alarm information")
	}
	return &model.AuditAlarm{
		PeerId:id, Uuid:f.Uuid, Typ:f.Typ,Info:info,
		ConnId:f.ConnId,ConnAuditRef:f.ConnAuditRef,
	},nil
}
