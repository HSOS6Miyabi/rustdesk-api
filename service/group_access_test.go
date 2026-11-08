package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupGroupDB(t *testing.T) func() {
	t.Helper()
	oldDB, oldService := DB, AllService
	db, err := gorm.Open(sqlite.Open("file:group_access_tests?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	sqlDB, err := db.DB()
	if err != nil { t.Fatal(err) }
	sqlDB.SetMaxOpenConns(1)
	if err = db.AutoMigrate(&model.Group{}, &model.GroupMembership{}, &model.GroupAnchor{},
		&model.User{}, &model.Contact{}, &model.Peer{}, &model.Oauth{}); err != nil {
		t.Fatal(err)
	}
	DB = db
	AllService = &Service{
		GroupService: &GroupService{},
		PeerService: &PeerService{},
		OauthService: &OauthService{},
	}
	return func() {
		sqlDB.Close()
		DB = oldDB
		AllService = oldService
	}
}

func seedGroups(t *testing.T) map[string]*model.User {
	t.Helper()
	groups := []model.Group{
		{IdModel: model.IdModel{Id: 1}, Name: "default", Type: model.GroupTypeDefault},
		{IdModel: model.IdModel{Id: 2}, Name: "shared", Type: model.GroupTypeShare},
		{IdModel: model.IdModel{Id: 3}, Name: "anchors", Type: model.GroupTypeAnchor},
	}
	for _, group := range groups {
		if err := DB.Create(&group).Error; err != nil { t.Fatal(err) }
	}
	admin, ordinary := true, false
	users := map[string]*model.User{
		"admin": {Username: "admin", GroupId: 3, IsAdmin: &admin, Status: 1},
		"member": {Username: "member", GroupId: 3, IsAdmin: &ordinary, Status: 1},
		"other": {Username: "other", GroupId: 3, IsAdmin: &ordinary, Status: 1},
		"anchor": {Username: "public", GroupId: 1, IsAdmin: &ordinary, Status: 1},
		"contact": {Username: "contact", GroupId: 1, IsAdmin: &ordinary, Status: 1},
	}
	for _, user := range users {
		if err := DB.Create(user).Error; err != nil { t.Fatal(err) }
	}
	return users
}

func includeID(ids []uint, target uint) bool {
	for _, id := range ids { if id == target { return true } }
	return false
}

func TestAnchorGroupVisibilityIncludesAdmin(t *testing.T) {
	cleanup := setupGroupDB(t); defer cleanup()
	users := seedGroups(t)
	if err := AllService.GroupService.ReplaceAnchors(3, []uint{users["anchor"].Id}); err != nil { t.Fatal(err) }
	for _, who := range []string{"admin", "member"} {
		visible, err := AllService.GroupService.VisibleUserIds(users[who])
		if err != nil { t.Fatal(err) }
		if !includeID(visible, users[who].Id) || !includeID(visible, users["anchor"].Id) {
			t.Fatalf("%s cannot see self and public anchor: %v", who, visible)
		}
		for _, hidden := range []string{"other", "contact"} {
			if includeID(visible, users[hidden].Id) {
				t.Errorf("%s sees unexpected user %s", who, hidden)
			}
		}
	}
}

func TestContactsGrantVisibilityOneWay(t *testing.T) {
	cleanup := setupGroupDB(t); defer cleanup()
	users := seedGroups(t)
	if err := DB.Create(&model.Contact{
		OwnerUserId: users["other"].Id, RecipientUserId: users["member"].Id,
	}).Error; err != nil { t.Fatal(err) }
	visible, err := AllService.GroupService.VisibleUserIds(users["member"])
	if err != nil { t.Fatal(err) }
	if !includeID(visible, users["other"].Id) { t.Fatal("owner not visible to recipient") }
	reverse, err := AllService.GroupService.VisibleUserIds(users["other"])
	if err != nil { t.Fatal(err) }
	if includeID(reverse, users["member"].Id) { t.Fatal("one-way grant became reciprocal") }
}

func TestSharedDevicesDeduplicated(t *testing.T) {
	cleanup := setupGroupDB(t); defer cleanup()
	users := seedGroups(t)
	if err := AllService.GroupService.ReplaceAnchors(3, []uint{users["anchor"].Id}); err != nil { t.Fatal(err) }
	if err := DB.Create(&model.Contact{
		OwnerUserId: users["anchor"].Id, RecipientUserId: users["member"].Id,
	}).Error; err != nil { t.Fatal(err) }
	for _, owner := range []string{"anchor", "other"} {
		if err := DB.Create(&model.Peer{Id: owner, UserId: users[owner].Id}).Error; err != nil { t.Fatal(err) }
	}
	result, err := AllService.PeerService.SharedDeviceList(users["member"], 1, 20, "", "all", 0)
	if err != nil { t.Fatal(err) }
	if result.Total != 1 || len(result.List) != 1 {
		t.Fatalf("want one public device, got %+v", result)
	}
	if len(result.List[0].Sources) != 2 {
		t.Fatalf("want contact and anchor sources, got %+v", result.List[0].Sources)
	}
}

func TestGroupMemberChangesAndOAuthEnrollment(t *testing.T) {
	cleanup := setupGroupDB(t); defer cleanup()
	users := seedGroups(t)
	svc := AllService.GroupService
	if err := svc.ChangeMembers(2, []uint{users["member"].Id}, "add"); err != nil { t.Fatal(err) }
	ids, err := svc.MemberIds(2)
	if err != nil || !includeID(ids, users["member"].Id) { t.Fatalf("add: %v %v", ids, err) }
	if err := svc.ChangeMembers(2, []uint{}, "replace"); err != nil { t.Fatal(err) }
	ids, err = svc.MemberIds(2)
	if err != nil || includeID(ids, users["member"].Id) { t.Fatalf("replace: %v %v", ids, err) }
	auto := true
	if err := DB.Create(&model.Oauth{Op: "company", OauthType: "oidc", AutoGroupIds: "2,3", AutoRegister: &auto}).Error; err != nil { t.Fatal(err) }
	if err := svc.AutoJoinOAuthGroups(users["other"].Id, "company"); err != nil { t.Fatal(err) }
	ids, err = svc.GroupIdsForUser(users["other"])
	if err != nil { t.Fatal(err) }
	if !includeID(ids, 2) || !includeID(ids, 3) { t.Fatalf("auto enrollment: %v", ids) }
	if err := svc.AutoJoinOAuthGroups(users["other"].Id, "company"); err != nil { t.Fatal(err) }
	ids, err = svc.MemberIds(2)
	if err != nil { t.Fatal(err) }
	if len(ids) != 1 { t.Errorf("duplicate group membership: %v", ids) }
}
