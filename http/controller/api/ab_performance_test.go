package api

import (
	"fmt"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAnchorScopeUsesOneBatchQueryForManyGroups(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:anchor_scope_performance?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	connection,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer connection.Close()
	connection.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Group{},&model.GroupMembership{},
		&model.GroupAnchor{},&model.Contact{});err!=nil{t.Fatal(err)}
	previousDB,previousServices,previousConfig,previousLogger:=service.DB,service.AllService,service.Config,service.Logger
	service.New(&config.Config{},db,log.New(),nil,nil)
	service.AllService.GroupService=&service.GroupService{}
	defer func(){
		service.DB,service.AllService,service.Config,service.Logger=
			previousDB,previousServices,previousConfig,previousLogger
	}()
	defaultGroup:=&model.Group{Name:"Default",Type:model.GroupTypeDefault}
	if err:=db.Create(defaultGroup).Error;err!=nil{t.Fatal(err)}
	viewer:=&model.User{Username:"scope-viewer",GroupId:defaultGroup.Id,Status:1}
	if err:=db.Create(viewer).Error;err!=nil{t.Fatal(err)}
	for i:=0;i<40;i++{
		shared:=&model.Group{Name:fmt.Sprintf("Shared-%02d",i),Type:model.GroupTypeShare}
		if err:=db.Create(shared).Error;err!=nil{t.Fatal(err)}
		if err:=db.Create(&model.GroupMembership{GroupId:shared.Id,UserId:viewer.Id}).Error;err!=nil{t.Fatal(err)}
	}
	queries:=0
	if err:=db.Callback().Query().Before("gorm:query").
		Register("perf:anchor_scope_queries",func(*gorm.DB){queries++});err!=nil{t.Fatal(err)}
	scope,hasAnchor,err:=anchorDirectoryScope(viewer)
	db.Callback().Query().Remove("perf:anchor_scope_queries")
	if err!=nil||hasAnchor||len(scope)!=0{t.Fatalf("unexpected scope: %v %v %v",scope,hasAnchor,err)}
	if queries>3{t.Fatalf("40 groups caused %d database reads",queries)}
	owner:=&model.User{Username:"scope-owner",GroupId:defaultGroup.Id,Status:1}
	if err:=db.Create(owner).Error;err!=nil{t.Fatal(err)}
	anchor:=&model.Group{Name:"Anchor",Type:model.GroupTypeAnchor}
	if err:=db.Create(anchor).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.GroupMembership{UserId:viewer.Id,GroupId:anchor.Id}).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.GroupAnchor{UserId:owner.Id,GroupId:anchor.Id}).Error;err!=nil{t.Fatal(err)}
	scope,hasAnchor,err=anchorDirectoryScope(viewer)
	if err!=nil||!hasAnchor||!scope[owner.Id]{
		t.Fatalf("anchor group was not resolved: %v %v %v",scope,hasAnchor,err)
	}
}
