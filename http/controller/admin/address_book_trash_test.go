package admin

import (
	"encoding/json"
	"strconv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAddressBookTrashHTTPAuthorizationAndRestore(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:addressbook_trash_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	sqlDB,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.AddressBook{},&model.AddressBookCollection{});err!=nil{t.Fatal(err)}
	oldDB,oldServices:=service.DB,service.AllService
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{}}
	defer func(){service.DB,service.AllService=oldDB,oldServices}()
	owner:=&model.User{Username:"owner",Status:1}
	other:=&model.User{Username:"other",Status:1}
	if err:=db.Create(owner).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(other).Error;err!=nil{t.Fatal(err)}
	first:=&model.AddressBook{Id:"lost-desk",UserId:other.Id,Tags:[]byte("[]")}
	if err:=db.Create(first).Error;err!=nil{t.Fatal(err)}
	if err:=db.Delete(first).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.Use(func(c *gin.Context){c.Set("curUser",owner);c.Next()})
	h:=&AddressBookTrash{}
	router.GET("/my/list",h.MyList)
	router.POST("/my/restore",h.MyRestore)
	router.GET("/admin/list",h.AdminList)
	router.POST("/admin/restore",h.AdminRestore)
	router.POST("/admin/purge",h.AdminPurge)
	call:=func(method,path,body string)*httptest.ResponseRecorder{
		t.Helper()
		req:=httptest.NewRequest(method,path,strings.NewReader(body))
		req.Header.Set("Content-Type","application/json")
		rec:=httptest.NewRecorder()
		router.ServeHTTP(rec,req)
		return rec
	}
	if rec:=call("GET","/my/list?user_id="+strconv.Itoa(int(other.Id)),"");!strings.Contains(rec.Body.String(),`"total":0`){
		t.Fatalf("personal trash exposed another user's entries: %s",rec.Body.String())
	}
	admin:=call("GET","/admin/list?user_id="+strconv.Itoa(int(other.Id)),"")
	if !strings.Contains(admin.Body.String(),"lost-desk"){t.Fatalf("administrator cannot list archived entry: %s",admin.Body.String())}
	body,_:=json.Marshal(map[string]interface{}{"row_id":first.RowId,"user_id":other.Id})
	if rec:=call("POST","/my/restore",string(body));rec.Code==http.StatusOK{
		t.Fatal("ordinary account restored another user's device")
	}
	if rec:=call("POST","/admin/restore",string(body));rec.Code!=http.StatusOK{
		t.Fatalf("admin could not restore: %d %s",rec.Code,rec.Body.String())
	}
	var restored model.AddressBook
	if err:=db.First(&restored,first.RowId).Error;err!=nil{t.Fatal(err)}
	if rec:=call("POST","/admin/purge",string(body));rec.Code==http.StatusOK{
		t.Fatal("active item was permanently deleted without archiving")
	}
}
