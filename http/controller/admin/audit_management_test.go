package admin

import (
	"encoding/csv"
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

func TestAuditCSVExportEscapesSpreadsheetFormulas(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:csv_admin_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,_:=db.DB()
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.AuditOperation{},&model.AuditConn{});err!=nil{t.Fatal(err)}
	oldDB:=service.DB
	service.DB=db
	defer func(){service.DB=oldDB}()
	if err:=db.Create(&model.AuditOperation{UserId:3,Account:"=HYPERLINK(1)",Method:"POST",Path:"/api/admin/group/create",Success:true,HttpStatus:200}).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.AuditConn{PeerId:"target",FromPeer:"source",ConnId:10,SessionId:"9007199254740993123"}).Error;err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.GET("/export",(&AuditManagement{}).Export)

	for _,kind:=range []string{"audit_operation","audit_conn"}{
		w:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodGet,"/export?kind="+kind,nil)
		router.ServeHTTP(w,req)
		if w.Code!=http.StatusOK{t.Fatalf("%s export status %d: %s",kind,w.Code,w.Body.String())}
		if !strings.Contains(w.Header().Get("Content-Disposition"),kind+".csv"){t.Fatalf("no CSV filename for %s",kind)}
		records,err:=csv.NewReader(strings.NewReader(w.Body.String())).ReadAll()
		if err!=nil{t.Fatal(err)}
		if len(records)!=2{t.Fatalf("%s: expected header and record, got %d",kind,len(records))}
		if kind=="audit_operation"&&records[1][2]!="'=HYPERLINK(1)" {
			t.Fatalf("spreadsheet formula not escaped: %q",records[1][2])
		}
		if kind=="audit_conn"&&records[1][9]!="9007199254740993123"{
			t.Fatalf("session identifier was truncated: %q",records[1][9])
		}
	}
}

func TestAuditExportRejectsUnknownCategories(t *testing.T){
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.GET("/export",(&AuditManagement{}).Export)
	w:=httptest.NewRecorder()
	router.ServeHTTP(w,httptest.NewRequest(http.MethodGet,"/export?kind=users",nil))
	if !strings.Contains(w.Body.String(),"unsupported export category"){
		t.Fatalf("unknown export category returned unexpected response: %s",w.Body.String())
	}
}
