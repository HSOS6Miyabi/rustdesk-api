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

func TestAddressBookCSVControllerOwnershipAndContent(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:addressbook_csv_http?mode=memory&cache=shared"),&gorm.Config{})
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
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.Use(func(c *gin.Context){c.Set("curUser",owner);c.Next()})
	handler:=&AddressBookCSV{}
	router.POST("/my/import",handler.MyImport)
	router.GET("/my/export",handler.MyExport)
	router.POST("/admin/import",handler.AdminImport)
	router.GET("/admin/export",handler.AdminExport)
	call:=func(method,path,body string)*httptest.ResponseRecorder{
		t.Helper()
		req:=httptest.NewRequest(method,path,strings.NewReader(body))
		rec:=httptest.NewRecorder()
		router.ServeHTTP(rec,req)
		return rec
	}
	result:=call(http.MethodPost,"/my/import?user_id="+strconv.Itoa(int(other.Id)),"id,alias\ndesk-one,电脑一\n")
	if result.Code!=http.StatusOK{t.Fatalf("CSV import failed: %d %s",result.Code,result.Body.String())}
	var responseBody struct{Data struct{Created int `json:"created"`} `json:"data"`}
	if err:=json.Unmarshal(result.Body.Bytes(),&responseBody);err!=nil||responseBody.Data.Created!=1{
		t.Fatalf("wrong import result: %s %v",result.Body.String(),err)
	}
	var count int64
	if err:=db.Model(&model.AddressBook{}).Where("user_id = ?",other.Id).Count(&count).Error;err!=nil||count!=0{
		t.Fatalf("ordinary user imported into another account: %d %v",count,err)
	}
	csvResponse:=call(http.MethodGet,"/my/export", "")
	if csvResponse.Code!=http.StatusOK || !strings.Contains(csvResponse.Body.String(),"电脑一") ||
		!strings.Contains(csvResponse.Header().Get("Content-Disposition"),"rustdesk-addressbook.csv"){
		t.Fatalf("export did not contain imported entry: %d %s",csvResponse.Code,csvResponse.Body.String())
	}
	adminResult:=call(http.MethodPost,"/admin/import?user_id="+strconv.Itoa(int(other.Id)),"id,alias\ndesk-two,电脑二\n")
	if adminResult.Code!=http.StatusOK{t.Fatalf("admin import failed: %s",adminResult.Body.String())}
	adminExport:=call(http.MethodGet,"/admin/export?user_id="+strconv.Itoa(int(other.Id)),"")
	if adminExport.Code!=http.StatusOK || !strings.Contains(adminExport.Body.String(),"desk-two") ||
		strings.Contains(adminExport.Body.String(),"desk-one"){
		t.Fatalf("admin CSV export was not scoped: %s",adminExport.Body.String())
	}
	if invalid:=call(http.MethodPost,"/admin/import?user_id=no", "id\ndesk-three\n");invalid.Code!=http.StatusBadRequest{
		t.Fatalf("invalid owner ID accepted: %d",invalid.Code)
	}
}
