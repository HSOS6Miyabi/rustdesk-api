package admin

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDelegatedAdminCanOnlyViewScopedAccountsAndDevices(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:admin_scoped_management_test?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.Group{},
		&model.GroupMembership{},&model.AdminRole{},&model.AdminRoleAssignment{});err!=nil{t.Fatal(err)}
	oldDB,oldServices:=service.DB,service.AllService
	defer func(){service.DB,service.AllService=oldDB,oldServices}()
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{},PeerService:&service.PeerService{}}
	f:=false
	user:=&model.User{Username:"delegated-operator",IsAdmin:&f,GroupId:1,Status:1}
	a:=&model.User{Username:"allowed-account",IsAdmin:&f,GroupId:2,Status:1}
	b:=&model.User{Username:"hidden-account",IsAdmin:&f,GroupId:1,Status:1}
	for _,item:=range []*model.User{user,a,b}{
		if err:=db.Create(item).Error;err!=nil{t.Fatal(err)}
	}
	if err:=db.Create(&model.Group{IdModel:model.IdModel{Id:2},Name:"remote support",Type:1}).Error;err!=nil{t.Fatal(err)}
	for _,device:=range []*model.Peer{
		{Id:"visible-desk",UserId:a.Id,Hostname:"front desk"},
		{Id:"private-desk",UserId:b.Id,Hostname:"hidden desk"},
	}{
		if err:=db.Create(device).Error;err!=nil{t.Fatal(err)}
	}
	role:=&model.AdminRole{
		Name:"Group read only",Permissions:"users.view,devices.view",
		ScopeType:service.AdminScopeGroup,ScopeId:2,Enabled:true,
	}
	if err:=service.SaveAdminRole(role);err!=nil{t.Fatal(err)}
	if err:=service.SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId:user.Id,RoleId:role.Id,
	});err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.Use(func(c *gin.Context){c.Set("curUser",user);c.Next()})
	r:=router.Group("/api/admin")
	adminUsers:=&User{}
	adminPeers:=&Peer{}
	r.GET("/user/list",middleware.AdminPrivilege(),adminUsers.List)
	r.GET("/user/detail/:id",middleware.AdminPrivilege(),adminUsers.Detail)
	r.GET("/peer/list",middleware.AdminPrivilege(),adminPeers.List)
	r.GET("/peer/detail/:id",middleware.AdminPrivilege(),adminPeers.Detail)
	deleted:=false
	r.POST("/user/delete",middleware.AdminPrivilege(),func(c *gin.Context){deleted=true;c.Status(http.StatusNoContent)})
	send:=func(method,path string)*httptest.ResponseRecorder{
		t.Helper()
		req:=httptest.NewRequest(method,path,nil)
		rec:=httptest.NewRecorder()
		router.ServeHTTP(rec,req)
		return rec
	}
	users:=send("GET","/api/admin/user/list?page=1&page_size=20")
	if users.Code!=http.StatusOK||!strings.Contains(users.Body.String(),"allowed-account")||
		strings.Contains(users.Body.String(),"hidden-account")||strings.Contains(users.Body.String(),"delegated-operator"){
		t.Fatalf("delegated user listing was not restricted: %d %s",users.Code,users.Body.String())
	}
	peers:=send("GET","/api/admin/peer/list?page=1&page_size=20")
	if peers.Code!=http.StatusOK||!strings.Contains(peers.Body.String(),"visible-desk")||
		strings.Contains(peers.Body.String(),"private-desk"){
		t.Fatalf("delegated device listing was not restricted: %d %s",peers.Code,peers.Body.String())
	}
	private:=send("GET","/api/admin/user/detail/"+strconv.Itoa(int(b.Id)))
	if strings.Contains(private.Body.String(),"hidden-account"){
		t.Fatalf("private account detail was exposed: %s",private.Body.String())
	}
	visible:=send("GET","/api/admin/user/detail/"+strconv.Itoa(int(a.Id)))
	if !strings.Contains(visible.Body.String(),"allowed-account"){
		t.Fatalf("allowed account detail was not available: %s",visible.Body.String())
	}
	privateDevice:=send("GET","/api/admin/peer/detail/2")
	if strings.Contains(privateDevice.Body.String(),"private-desk"){
		t.Fatalf("private device detail was exposed: %s",privateDevice.Body.String())
	}
	send("POST","/api/admin/user/delete")
	if deleted{t.Fatal("read-only role was able to delete a user")}
}

func TestAdminRolesCanBeGloballyDelegatedWithoutBecomingFullAdmins(t *testing.T){
	db,err:=gorm.Open(sqlite.Open("file:global_admin_delegate?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.AdminRole{},&model.AdminRoleAssignment{});err!=nil{t.Fatal(err)}
	oldDB,oldService:=service.DB,service.AllService
	defer func(){service.DB,service.AllService=oldDB,oldService}()
	service.DB=db
	service.AllService=&service.Service{UserService:&service.UserService{}}
	f:=false
	member:=&model.User{Username:"helpdesk",Status:1,IsAdmin:&f}
	if err:=db.Create(member).Error;err!=nil{t.Fatal(err)}
	role:=&model.AdminRole{
		Name:"Helpdesk editor",Permissions:"users.view,users.edit",
		ScopeType:service.AdminScopeGlobal,Enabled:true,
	}
	if err:=service.SaveAdminRole(role);err!=nil{t.Fatal(err)}
	if err:=service.SaveAdminRoleAssignment(&model.AdminRoleAssignment{
		UserId:member.Id,RoleId:role.Id,
	});err!=nil{t.Fatal(err)}
	if service.AllService.UserService.IsAdmin(member){
		t.Fatal("delegated account became a full system administrator")
	}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.Use(func(c *gin.Context){c.Set("curUser",member);c.Next()})
	approved,blocked:=false,false
	router.POST("/api/admin/user/update",middleware.AdminPrivilege(),func(c *gin.Context){approved=true;c.Status(200)})
	router.POST("/api/admin/user/delete",middleware.AdminPrivilege(),func(c *gin.Context){blocked=true;c.Status(200)})
	router.GET("/api/admin/admin_role/list",middleware.AdminPrivilege(),func(c *gin.Context){blocked=true;c.Status(200)})
	for _,tc:=range []struct{method,path string}{
		{"POST","/api/admin/user/update"},
		{"POST","/api/admin/user/delete"},
		{"GET","/api/admin/admin_role/list"},
	}{
		out:=httptest.NewRecorder()
		router.ServeHTTP(out,httptest.NewRequest(tc.method,tc.path,nil))
	}
	if !approved||blocked{t.Fatalf("permissions allowed wrong operations: approved=%v blocked=%v",approved,blocked)}
}
