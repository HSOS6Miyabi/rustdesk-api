package api

import (
	"encoding/json"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"strconv"
	"strings"

)

type AuditConnForm struct {
	Action    string   `json:"action"`
	ConnId    int64    `json:"conn_id"`
	Id        string   `json:"id"`
	Peer      []string `json:"peer"`
	Ip        string   `json:"ip"`
	SessionId json.RawMessage `json:"session_id"`
	PrimaryAuth *int `json:"primary_auth"`
	TwoFactor *int `json:"two_factor"`
	ConnAuditRef string `json:"conn_audit_ref"`
	Nonce string `json:"nonce"`
	Type      int      `json:"type"`
	Uuid      string   `json:"uuid"`
}

func (a *AuditConnForm) ToAuditConn() *model.AuditConn {
	fp := ""
	fn := ""
	if len(a.Peer) >= 1 {
		fp = a.Peer[0]
		if len(a.Peer) >= 2 {
			fn = a.Peer[1]
		}
	}
	ssid := ExactSessionID(a.SessionId)
	row := &model.AuditConn{
		Action:    a.Action,
		ConnId:    a.ConnId,
		PeerId:    a.Id,
		FromPeer:  fp,
		FromName:  fn,
		Ip:        a.Ip,
		SessionId: ssid,
		ConnAuditRef: strings.TrimSpace(a.ConnAuditRef),
		EventNonce: strings.TrimSpace(a.Nonce),
		Type:      a.Type,
		Uuid:      a.Uuid,
	}
	if a.PrimaryAuth != nil { row.PrimaryAuth = *a.PrimaryAuth }
	if a.TwoFactor != nil { row.TwoFactor = *a.TwoFactor }
	return row
}

// ExactSessionID preserves numeric identifiers beyond JavaScript floating-point precision.
func ExactSessionID(raw json.RawMessage) string {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" { return "" }
	if strings.HasPrefix(value, "\"") {
		var quoted string
		if err := json.Unmarshal(raw, &quoted); err == nil { return quoted }
		return ""
	}
	if _, err := strconv.ParseInt(value, 10, 64); err == nil { return value }
	if _, err := strconv.ParseUint(value, 10, 64); err == nil { return value }
	return ""
}

type AuditFileForm struct {
	Id     string `json:"id"`
	Info   string `json:"info"`
	IsFile bool   `json:"is_file"`
	Path   string `json:"path"`
	PeerId string `json:"peer_id"`
	ConnId int64 `json:"conn_id"`
	Nonce string `json:"nonce"`
	Type   int    `json:"type"`
	Uuid   string `json:"uuid"`
}
type AuditFileInfo struct {
	Ip   string `json:"ip"`
	Name string `json:"name"`
	Num  int    `json:"num"`
}

func (a *AuditFileForm) ToAuditFile() *model.AuditFile {
	fi := &AuditFileInfo{}
	err := json.Unmarshal([]byte(a.Info), fi)
	if err != nil {
		global.Logger.Warn("ToAuditFile", err)
	}

	return &model.AuditFile{
		PeerId:   a.Id,
		ConnId: a.ConnId,
		EventNonce: strings.TrimSpace(a.Nonce),
		Info:     a.Info,
		IsFile:   a.IsFile,
		FromPeer: a.PeerId,
		Path:     a.Path,
		Type:     a.Type,
		Uuid:     a.Uuid,
		FromName: fi.Name,
		Ip:       fi.Ip,
		Num:      fi.Num,
	}
}
