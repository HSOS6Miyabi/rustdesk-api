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

func TestInvitationPublicRegistrationRejectsInvalidToken(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:invitation_http?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.UserInvitation{});err!=nil{t.Fatal(err)}
	oldDB:=service.DB
	service.DB=db
	defer func(){service.DB=oldDB}()
	gin.SetMode(gin.TestMode)
	r:=gin.New()
	r.POST("/invite/accept",(&Invitation{}).Accept)
	req:=httptest.NewRequest(http.MethodPost,"/invite/accept",
		strings.NewReader(`{"token":"bad","username":"member","password":"NewPassword123"}`))
	req.Header.Set("Content-Type","application/json")
	out:=httptest.NewRecorder()
	r.ServeHTTP(out,req)
	if out.Code!=http.StatusBadRequest{t.Fatalf("invalid invitation accepted: %s",out.Body.String())}
	var users int64
	if err:=db.Model(&model.User{}).Count(&users).Error;err!=nil||users!=0{
		t.Fatalf("invalid invitation created a user: %d %v",users,err)
	}
}
