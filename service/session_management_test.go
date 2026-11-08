package service

import (
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestSessionManagementRejectsStaleAndInvalidRequests(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.DeviceActiveConnection{},&model.DeviceDisconnectRequest{});err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"session-pc"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	if _,err:=SynchronizeDeviceSessions(peer,make([]int32,maxConnectionsPerHeartbeat+1));err==nil{
		t.Fatal("oversized heartbeat was accepted")
	}
	if err:=RequestDeviceDisconnect(peer.Id,10,1);err==nil{
		t.Fatal("nonexistent active connection was accepted")
	}
	if err:=DB.Create(&model.DeviceActiveConnection{
		DeviceId:peer.Id,ConnId:10,LastSeenAt:time.Now().Unix()-sessionAliveSeconds-10,
	}).Error;err!=nil{t.Fatal(err)}
	if err:=RequestDeviceDisconnect(peer.Id,10,1);err==nil{
		t.Fatal("stale active connection was accepted")
	}
	rows,count,err:=ListActiveDeviceSessions(1,20)
	if err!=nil||count!=0||len(rows)!=0{t.Fatalf("stale session is visible: %v %v",rows,err)}
	for _,test:=range []struct{id string;conn int32;user uint}{
		{"",10,1},{"session-pc",0,1},{"session-pc",10,0},
	}{
		if err:=RequestDeviceDisconnect(test.id,test.conn,test.user);err==nil{
			t.Fatalf("invalid request accepted: %+v",test)
		}
	}
}
