package api

import (
	"encoding/json"
	"testing"
)

func TestAuditConnectionPreservesLongSessionID(t *testing.T) {
	var form AuditConnForm
	if err:=json.Unmarshal([]byte(`{"id":"remote","conn_id":10,"session_id":9007199254740993123,"primary_auth":3,"two_factor":2,"nonce":"first"}`),&form);err!=nil{t.Fatal(err)}
	row:=form.ToAuditConn()
	if row.SessionId!="9007199254740993123" {
		t.Fatalf("long session ID lost precision: %s",row.SessionId)
	}
	if row.PrimaryAuth!=3 || row.TwoFactor!=2 || row.EventNonce!="first" {
		t.Fatalf("authentication fields missing: %+v",row)
	}
	if err:=json.Unmarshal([]byte(`{"session_id":"9999999999999999999999"}`),&form);err!=nil{t.Fatal(err)}
	if got:=form.ToAuditConn().SessionId;got!="9999999999999999999999" {
		t.Fatalf("string session ID changed: %s",got)
	}
}

func TestAuditFileConnectionAndNonce(t *testing.T) {
	var input AuditFileForm
	if err:=json.Unmarshal([]byte(`{"id":"target","peer_id":"source","conn_id":88,"nonce":"unique-file-88","info":"{\"ip\":\"127.0.0.1\",\"name\":\"PC\",\"num\":2}"}`),&input);err!=nil{t.Fatal(err)}
	row:=input.ToAuditFile()
	if row.ConnId!=88 || row.EventNonce!="unique-file-88" || row.Num!=2 || row.FromPeer!="source" {
		t.Fatalf("file auditing fields missing: %+v",row)
	}
}

func TestAuditAlarmAcceptsLegacyStringAndModernObject(t *testing.T) {
	for _,payload:=range []string{
		`{"id":"target","typ":0,"info":"{\"ip\":\"127.0.0.1\"}","conn_id":88,"nonce":"alarm1"}`,
		`{"id":"target","typ":0,"info":{"ip":"127.0.0.1"},"conn_id":88,"nonce":"alarm2"}`,
	} {
		var form AuditAlarmForm
		if err:=json.Unmarshal([]byte(payload),&form);err!=nil{t.Fatal(err)}
		row,err:=form.ToAuditAlarm()
		if err!=nil{t.Fatal(err)}
		if row.PeerId!="target"||row.ConnId!=88||!json.Valid([]byte(row.Info)){
			t.Fatalf("invalid alarm: %+v",row)
		}
	}
}
