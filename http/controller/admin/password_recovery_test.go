package admin

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

func TestPublicPasswordRecoveryRepliesWithoutDisclosingAccounts(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:recovery_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.SMTPSetting{},&model.PasswordResetToken{},&model.UserToken{});err!=nil{t.Fatal(err)}
	oldDB:=service.DB
	service.DB=db
	defer func(){service.DB=oldDB}()
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/password/forgot",(&PasswordRecovery{}).Request)
	router.POST("/password/reset",(&PasswordRecovery{}).Complete)
	respond:=func(path,payload string)*httptest.ResponseRecorder{
		req:=httptest.NewRequest(http.MethodPost,path,strings.NewReader(payload))
		req.Header.Set("Content-Type","application/json")
		out:=httptest.NewRecorder()
		router.ServeHTTP(out,req)
		return out
	}
	known:=respond("/password/forgot",`{"email":"existing@example.com"}`)
	unknown:=respond("/password/forgot",`{"email":"unknown@example.com"}`)
	if known.Code!=http.StatusOK||unknown.Code!=http.StatusOK||known.Body.String()!=unknown.Body.String(){
		t.Fatalf("password recovery exposed account details: %s / %s",known.Body.String(),unknown.Body.String())
	}
	if response:=respond("/password/reset",`{"token":"invalid","password":"NewPassword456"}`);
		response.Code!=http.StatusBadRequest{t.Fatalf("invalid reset succeeded: %s",response.Body.String())}
}
