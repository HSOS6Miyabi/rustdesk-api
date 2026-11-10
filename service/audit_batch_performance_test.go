package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAuditCleanupProcessesSeveralBatches(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AuditConn{},&model.AuditRetentionPolicy{},&model.AuditEventNonce{});err!=nil{t.Fatal(err)}
	if err:=SaveAuditRetention(&model.AuditRetentionPolicy{ConnectionDays:30});err!=nil{t.Fatal(err)}
	old:=make([]model.AuditConn,1201)
	for i:=range old {old[i]=model.AuditConn{PeerId:"old-audit-batch",ConnId:int64(i+1)}}
	if err:=DB.CreateInBatches(old,100).Error;err!=nil{t.Fatal(err)}
	age:=time.Now().Add(-60*24*time.Hour)
	if err:=DB.Model(&model.AuditConn{}).Where("peer_id = ?","old-audit-batch").Update("created_at",age).Error;err!=nil{t.Fatal(err)}
	recent:=&model.AuditConn{PeerId:"keep-recent",ConnId:1}
	if err:=DB.Create(recent).Error;err!=nil{t.Fatal(err)}
	if err:=PruneOldAuditRecords(time.Now());err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.AuditConn{}).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("batch cleanup retained %d entries: %v",count,err)
	}
	nonces:=make([]model.AuditEventNonce,1201)
	for i:=range nonces{
		nonces[i]=model.AuditEventNonce{Nonce:fmt.Sprintf("old-%04d",i),CreatedAt:age.Unix()}
	}
	if err:=DB.CreateInBatches(nonces,100).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Create(&model.AuditEventNonce{Nonce:"recent",CreatedAt:time.Now().Unix()}).Error;err!=nil{t.Fatal(err)}
	if err:=PruneAuditNonces(time.Now());err!=nil{t.Fatal(err)}
	if err:=DB.Model(&model.AuditEventNonce{}).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("batch nonce cleanup retained %d entries: %v",count,err)
	}
}
