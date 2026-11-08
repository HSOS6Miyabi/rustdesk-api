package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestNewAuditFieldsAndDuplicateEventHandling(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AuditConn{},&model.AuditFile{},&model.AuditAlarm{},&model.AuditEventNonce{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	receiver,sender:=users["member"],users["other"]
	if err:=DB.Create(&model.Peer{Id:"target",UserId:receiver.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.Peer{Id:"source",UserId:sender.Id}).Error;err!=nil{t.Fatal(err)}
	svc:=&AuditService{}

	start:=&model.AuditConn{PeerId:"target",ConnId:45,Action:"new",SessionId:"9007199254740993123",ConnAuditRef:"opaque-connection-ref",EventNonce:"a1"}
	if err:=svc.SaveAuditConnection(start);err!=nil{t.Fatal(err)}
	if err:=svc.SaveAuditConnection(start);err!=nil{t.Fatal(err)}
	detail:=&model.AuditConn{PeerId:"target",ConnId:45,FromPeer:"source",PrimaryAuth:3,TwoFactor:2,EventNonce:"a2"}
	if err:=svc.SaveAuditConnection(detail);err!=nil{t.Fatal(err)}
	if err:=svc.SaveAuditConnection(detail);err!=nil{t.Fatal(err)}
	var conns []model.AuditConn
	if err:=DB.Find(&conns).Error;err!=nil{t.Fatal(err)}
	if len(conns)!=1||conns[0].PrimaryAuth!=3||conns[0].TwoFactor!=2||
		conns[0].SessionId!="9007199254740993123"||conns[0].FromUserId!=sender.Id{
		t.Fatalf("connection audit not merged correctly: %+v",conns)
	}
	file:=&model.AuditFile{PeerId:"target",FromPeer:"source",ConnId:45,Path:"/report.txt",EventNonce:"file1"}
	for i:=0;i<2;i++{if err:=svc.SaveAuditFile(file);err!=nil{t.Fatal(err)}}
	var files []model.AuditFile
	if err:=DB.Find(&files).Error;err!=nil{t.Fatal(err)}
	if len(files)!=1||files[0].ConnId!=45 {t.Fatalf("duplicate file audit: %+v",files)}
	alarm:=&model.AuditAlarm{PeerId:"target",ConnId:45,Typ:0,Info:`{"ip":"192.0.2.1"}`}
	for i:=0;i<2;i++{if err:=svc.SaveAuditAlarm(alarm,"alarm1","192.0.2.1");err!=nil{t.Fatal(err)}}
	rows,total,err:=svc.OwnAlarms(receiver.Id,1,20)
	if err!=nil||total!=1||len(rows)!=1||rows[0].FromUserId!=sender.Id||rows[0].ToUserId!=receiver.Id {
		t.Fatalf("security alarm association failed: %+v %d %v",rows,total,err)
	}
	_,total,err=svc.OwnAlarms(users["contact"].Id,1,20)
	if err!=nil||total!=0 {t.Fatalf("unrelated user could see alarm: %d %v",total,err)}
	var nonceCount int64
	if err:=DB.Model(&model.AuditEventNonce{}).Count(&nonceCount).Error;err!=nil{t.Fatal(err)}
	if nonceCount!=4 {t.Fatalf("unexpected nonce count: %d",nonceCount)}
}

func TestAuditNoncePruning(t *testing.T){
	cleanup:=setupGroupDB(t);defer cleanup()
	if err:=DB.AutoMigrate(&model.AuditEventNonce{});err!=nil{t.Fatal(err)}
	now:=time.Now()
	for _,row:=range []model.AuditEventNonce{
		{Nonce:"old",CreatedAt:now.Add(-48*time.Hour).Unix()},
		{Nonce:"recent",CreatedAt:now.Unix()},
	}{if err:=DB.Create(&row).Error;err!=nil{t.Fatal(err)}}
	if err:=PruneAuditNonces(now);err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.AuditEventNonce{}).Count(&count).Error;err!=nil{t.Fatal(err)}
	if count!=1 {t.Fatalf("nonce cleanup removed incorrect rows: %d",count)}
}
