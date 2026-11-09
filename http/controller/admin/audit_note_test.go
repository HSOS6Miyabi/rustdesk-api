package admin

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConnectionAuditNoteHTTPAndCSV(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:audit_note_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.AuditConn{});err!=nil{t.Fatal(err)}
	oldDB,oldService:=service.DB,service.AllService
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{}}
	defer func(){service.DB,service.AllService=oldDB,oldService}()
	user:=&model.User{Username:"admin",Status:1}
	if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
	row:=&model.AuditConn{PeerId:"managed-desk",FromPeer:"support-desk",ConnId:42}
	if err:=db.Create(row).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.Use(func(c *gin.Context){c.Set("curUser",user);c.Next()})
	router.POST("/note",(&Audit{}).UpdateNote)
	router.GET("/export",(&AuditManagement{}).Export)
	body,_:=json.Marshal(map[string]interface{}{"id":row.Id,"note":"Ticket 123, 远程协助"})
	request:=httptest.NewRequest(http.MethodPost,"/note",bytes.NewReader(body))
	request.Header.Set("Content-Type","application/json")
	response:=httptest.NewRecorder()
	router.ServeHTTP(response,request)
	if response.Code!=http.StatusOK||!strings.Contains(response.Body.String(),`"code":0`){
		t.Fatalf("cannot save note: %d %s",response.Code,response.Body.String())
	}
	export:=httptest.NewRecorder()
	router.ServeHTTP(export,httptest.NewRequest(http.MethodGet,"/export?kind=audit_conn",nil))
	if export.Code!=http.StatusOK{t.Fatalf("cannot export notes: %s",export.Body.String())}
	records,err:=csv.NewReader(strings.NewReader(export.Body.String())).ReadAll()
	if err!=nil{t.Fatal(err)}
	if len(records)!=2||len(records[0])!=len(records[1])||records[0][len(records[0])-2]!="note"||
		records[1][len(records[1])-2]!="Ticket 123, 远程协助"{
		t.Fatalf("CSV does not contain saved connection note: %v",records)
	}
	invalid:=httptest.NewRecorder()
	tooLong:=[]byte(`{"id":`+strconv.FormatUint(uint64(row.Id),10)+`,"note":"`+strings.Repeat("x",2001)+`"}`)
	req:=httptest.NewRequest(http.MethodPost,"/note",bytes.NewReader(tooLong))
	req.Header.Set("Content-Type","application/json")
	router.ServeHTTP(invalid,req)
	if invalid.Code!=http.StatusBadRequest{t.Fatalf("invalid note saved: %s",invalid.Body.String())}
}
