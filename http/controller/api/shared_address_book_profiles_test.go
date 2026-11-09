package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSharedAddressBookProfilesRespectIndividualAndAdditionalGroupPermissions(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:shared_profiles_group_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	sqlDB,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Group{},&model.GroupMembership{},
		&model.GroupAnchor{},&model.Contact{},&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{});err!=nil{t.Fatal(err)}
	previousDB,previousServices:=service.DB,service.AllService
	service.DB=db
	service.AllService=&service.Service{
		UserService:&service.UserService{},
		GroupService:&service.GroupService{},
		AddressBookService:&service.AddressBookService{},
	}
	defer func(){service.DB,service.AllService=previousDB,previousServices}()
	primary:=&model.Group{Name:"default",Type:model.GroupTypeDefault}
	secondary:=&model.Group{Name:"engineering",Type:model.GroupTypeShare}
	for _,group:=range []*model.Group{primary,secondary}{
		if err:=db.Create(group).Error;err!=nil{t.Fatal(err)}
	}
	owner:=&model.User{Username:"book-owner",GroupId:primary.Id,Status:1}
	viewer:=&model.User{Username:"viewer",GroupId:primary.Id,Status:1}
	for _,user:=range []*model.User{owner,viewer}{
		if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
	}
	book:=&model.AddressBookCollection{UserId:owner.Id,Name:"Shared service devices"}
	if err:=db.Create(book).Error;err!=nil{t.Fatal(err)}
	public:=&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,
		Type:model.ShareAddressBookRuleTypeAll,ToId:0,Rule:model.ShareAddressBookRuleRuleRead,
	}
	groupRule:=&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,
		Type:model.ShareAddressBookRuleTypeGroup,ToId:secondary.Id,Rule:model.ShareAddressBookRuleRuleFullControl,
	}
	direct:=&model.AddressBookCollectionRule{
		UserId:owner.Id,CollectionId:book.Id,
		Type:model.ShareAddressBookRuleTypePersonal,ToId:viewer.Id,Rule:model.ShareAddressBookRuleRuleReadWrite,
	}
	for _,rule:=range []*model.AddressBookCollectionRule{public,groupRule,direct}{
		if err:=db.Create(rule).Error;err!=nil{t.Fatal(err)}
	}
	if err:=db.Create(&model.GroupMembership{GroupId:secondary.Id,UserId:viewer.Id}).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/ab/shared/profiles",func(c *gin.Context){c.Set("curUser",viewer);c.Next()},(&Ab{}).SharedProfiles)
	verify:=func(want int){
		t.Helper()
		w:=httptest.NewRecorder()
		router.ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/api/ab/shared/profiles",nil))
		if w.Code!=http.StatusOK{
			t.Fatalf("unexpected API status: %d %s",w.Code,w.Body.String())
		}
		var body struct {
			Data []struct{
				Guid string `json:"guid"`
				Owner string `json:"owner"`
				Rule int `json:"rule"`
			} `json:"data"`
		}
		if err:=json.Unmarshal(w.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
		if len(body.Data)!=1 || body.Data[0].Rule!=want ||
			body.Data[0].Guid=="" || body.Data[0].Owner!=owner.Username{
			t.Fatalf("want one shared book with permission %d: %s",want,w.Body.String())
		}
	}
	verify(model.ShareAddressBookRuleRuleReadWrite)
	if err:=db.Delete(direct).Error;err!=nil{t.Fatal(err)}
	verify(model.ShareAddressBookRuleRuleFullControl)
	if err:=db.Where("group_id = ? AND user_id = ?",secondary.Id,viewer.Id).
		Delete(&model.GroupMembership{}).Error;err!=nil{t.Fatal(err)}
	verify(model.ShareAddressBookRuleRuleRead)
}
