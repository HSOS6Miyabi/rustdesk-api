package service

import (
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestConnectionAuditNotes(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AuditConn{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	row:=&model.AuditConn{PeerId:"remote-desk",FromPeer:"support-desk",ConnId:112}
	if err:=DB.Create(row).Error;err!=nil{t.Fatal(err)}
	if err:=UpdateConnectionNote(row.Id,users["admin"].Id,"  应用安装、远程协助  ");err!=nil{t.Fatal(err)}
	var updated model.AuditConn
	if err:=DB.First(&updated,row.Id).Error;err!=nil{t.Fatal(err)}
	if updated.Note!="应用安装、远程协助"||updated.NoteUpdatedBy!=users["admin"].Id{
		t.Fatalf("session note was not persisted: %+v",updated)
	}
	if updated.ConnId!=112||updated.PeerId!="remote-desk"{
		t.Fatalf("original audit data was modified: %+v",updated)
	}
	if err:=UpdateConnectionNote(row.Id,users["admin"].Id,strings.Repeat("中",2001));err==nil{
		t.Fatal("oversized note accepted")
	}
	if err:=UpdateConnectionNote(row.Id,users["admin"].Id,"");err!=nil{t.Fatal(err)}
	if err:=DB.First(&updated,row.Id).Error;err!=nil{t.Fatal(err)}
	if updated.Note!=""{t.Fatal("empty note did not clear the previous value")}
	if err:=UpdateConnectionNote(999999,users["admin"].Id,"missing");err==nil{
		t.Fatal("unknown audit record accepted")
	}
	if err:=UpdateConnectionNote(row.Id,0,"no author");err==nil{
		t.Fatal("unknown administrator ID accepted")
	}
}
