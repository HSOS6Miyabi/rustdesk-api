package middleware

import (
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

func TestOperationMiddlewareLogsOnlyAuthenticatedChanges(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:operation_audit_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.AuditOperation{});err!=nil{t.Fatal(err)}
	oldDB,oldService:=service.DB,service.AllService
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{}}
	defer func(){service.DB=oldDB;service.AllService=oldService}()
	user:=&model.User{IdModel:model.IdModel{Id:12},Username:"operator"}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	api:=router.Group("/api")
	api.Use(func(c *gin.Context){c.Set("curUser",user);c.Next()},RecordOperations())
	api.POST("/success",func(c *gin.Context){c.JSON(200,gin.H{"code":0,"data":"ok"})})
	api.POST("/reject",func(c *gin.Context){c.JSON(200,gin.H{"code":101,"message":"not allowed"})})
	api.GET("/read",func(c *gin.Context){c.JSON(200,gin.H{"code":0})})
	for _,item:=range []struct{method,path string}{
		{"POST","/api/success"},{"POST","/api/reject"},{"GET","/api/read"},
	}{
		w:=httptest.NewRecorder()
		request:=httptest.NewRequest(item.method,item.path,strings.NewReader(`{"password":"secret-value"}`))
		request.Header.Set("User-Agent","RustDesk desktop")
		router.ServeHTTP(w,request)
	}
	var history []model.AuditOperation
	if err:=db.Order("id").Find(&history).Error;err!=nil{t.Fatal(err)}
	if len(history)!=2 || !history[0].Success || history[1].Success {
		t.Fatalf("operation outcomes were not recorded: %+v",history)
	}
	for _,entry:=range history{
		if entry.Account!="operator"||entry.UserId!=12||strings.Contains(entry.Path,"password")||
			strings.Contains(entry.UserAgent,"secret-value"){
			t.Fatalf("operation record contains sensitive or unexpected values: %+v",entry)
		}
	}
}
