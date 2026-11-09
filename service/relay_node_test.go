package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestRelayNodeManagementAndSubnetValidation(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.RelayNode{});err!=nil{t.Fatal(err)}
	east:=&model.RelayNode{
		Name:"East relay",Address:"relay-east.example.test:21117",
		Subnets:"203.0.113.0/24\n2001:db8:1::/48",Priority:10,Enabled:true,
	}
	if err:=SaveRelayNode(east);err!=nil{t.Fatal(err)}
	west:=&model.RelayNode{
		Name:"West relay",Address:"[2001:db8::1]:21117",
		Subnets:"198.51.100.0/24",Priority:20,Enabled:true,
	}
	if err:=SaveRelayNode(west);err!=nil{t.Fatal(err)}
	active,err:=EnabledRelayNodes()
	if err!=nil||len(active)!=2{t.Fatalf("enabled nodes: %v %v",active,err)}
	if active[0].Address!=east.Address||len(active[0].Subnets)!=2{
		t.Fatalf("active nodes were not normalized: %v",active)
	}
	if active[1].Address!=west.Address||active[1].Priority!=20{
		t.Fatalf("relay preference order changed: %v",active)
	}
	nodes,total,err:=ListRelayNodes(1,20)
	if err!=nil||total!=2||len(nodes)!=2{t.Fatalf("cannot list relay nodes: %d %v %v",total,nodes,err)}
	west.Enabled=false
	if err:=SaveRelayNode(west);err!=nil{t.Fatal(err)}
	active,err=EnabledRelayNodes()
	if err!=nil||len(active)!=1||active[0].Address!=east.Address{
		t.Fatalf("disabled relay still advertised: %v %v",active,err)
	}
	if err:=DeleteRelayNode(east.Id);err!=nil{t.Fatal(err)}
	active,err=EnabledRelayNodes()
	if err!=nil||len(active)!=0{t.Fatalf("deleted relay still advertised: %v %v",active,err)}
	if err:=DeleteRelayNode(east.Id);err==nil{t.Fatal("relay deleted twice")}
	for _,entry:=range []*model.RelayNode{
		{Name:"bad",Address:"http://relay.example.test:21117",Enabled:true},
		{Name:"bad",Address:"relay.example.test:65536",Enabled:true},
		{Name:"bad",Address:"bad host",Enabled:true},
		{Name:"bad",Address:"relay.example.test:21117",Subnets:"203.0.113.7/99",Enabled:true},
		{Name:"bad",Address:"relay.example.test:21117",Priority:-1,Enabled:true},
	}{
		if err:=SaveRelayNode(entry);err==nil{
			t.Fatalf("invalid relay node accepted: %+v",entry)
		}
	}
}
