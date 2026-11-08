package service

import (
	"fmt"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

func TestDirectoryQueriesStayBoundedWithManyGroups(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	users := seedGroups(t)
	viewer, sharedOwner, publicAnchor := users["member"], users["other"], users["anchor"]
	if err := DB.Model(viewer).Update("group_id", 1).Error; err != nil { t.Fatal(err) }
	viewer.GroupId = 1

	for i:=0; i<40; i++ {
		group := &model.Group{Name:fmt.Sprintf("Shared %02d",i),Type:model.GroupTypeShare}
		if err := DB.Create(group).Error; err != nil { t.Fatal(err) }
		for _, id := range []uint{viewer.Id, sharedOwner.Id} {
			if err := DB.Create(&model.GroupMembership{UserId:id,GroupId:group.Id}).Error; err != nil {
				t.Fatal(err)
			}
		}
	}

	queries:=0
	if err := DB.Callback().Query().Before("gorm:query").Register("test:directory_query_counter",
		func(*gorm.DB){ queries++ }); err != nil { t.Fatal(err) }
	defer DB.Callback().Query().Remove("test:directory_query_counter")

	visible, err := AllService.GroupService.VisibleUserIds(viewer)
	if err != nil { t.Fatal(err) }
	if queries>12 { t.Fatalf("visible directory ran %d SQL queries for 40 groups",queries) }
	if !includeID(visible,sharedOwner.Id) { t.Fatal("shared group member hidden") }

	queries=0
	sources, err := sharedOwnerSources(viewer)
	if err != nil { t.Fatal(err) }
	if queries>12 { t.Fatalf("shared device directory ran %d SQL queries for 40 groups",queries) }
	if len(sources[sharedOwner.Id])!=40 {
		t.Fatalf("want 40 unique shared group sources, got %d",len(sources[sharedOwner.Id]))
	}

	anchor := &model.Group{Name:"Private public anchors",Type:model.GroupTypeAnchor}
	if err := DB.Create(anchor).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(&model.GroupMembership{GroupId:anchor.Id,UserId:viewer.Id}).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(&model.GroupAnchor{GroupId:anchor.Id,UserId:publicAnchor.Id}).Error; err != nil { t.Fatal(err) }

	visible, err = AllService.GroupService.VisibleUserIds(viewer)
	if err != nil { t.Fatal(err) }
	if includeID(visible,sharedOwner.Id) || !includeID(visible,publicAnchor.Id) {
		t.Fatalf("anchor precedence failed: %v",visible)
	}
	sources, err = sharedOwnerSources(viewer)
	if err != nil { t.Fatal(err) }
	if _, ok := sources[sharedOwner.Id]; ok {
		t.Fatal("shared group source bypassed public anchor visibility")
	}
	if len(sources[publicAnchor.Id])!=1 || sources[publicAnchor.Id][0].Id!=anchor.Id {
		t.Fatalf("missing public anchor source: %+v",sources[publicAnchor.Id])
	}
	if err := DB.Create(&model.Contact{OwnerUserId:sharedOwner.Id,RecipientUserId:viewer.Id}).Error; err != nil {
		t.Fatal(err)
	}
	visible, err = AllService.GroupService.VisibleUserIds(viewer)
	if err != nil || !includeID(visible,sharedOwner.Id) {
		t.Fatalf("explicit one-way contact grant was hidden: %v %v",visible,err)
	}
	sources, err = sharedOwnerSources(viewer)
	if err != nil || len(sources[sharedOwner.Id])!=1 || sources[sharedOwner.Id][0].Type!="contact" {
		t.Fatalf("contact source not distinguished from shared group: %+v %v",sources[sharedOwner.Id],err)
	}
}
