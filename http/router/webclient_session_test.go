package router

import (
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/http/middleware"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func TestWebClientStaticRequiresActiveBrowserLogin(t *testing.T) {
    db, err := gorm.Open(sqlite.Open("file:webclient_access_test?mode=memory&cache=shared"),&gorm.Config{})
    if err != nil { t.Fatal(err) }
    sqlDB, err := db.DB()
    if err != nil { t.Fatal(err) }
    defer sqlDB.Close()
    sqlDB.SetMaxOpenConns(1)
    if err := db.AutoMigrate(&model.User{},&model.UserToken{});err!=nil { t.Fatal(err) }
    oldDB, oldConfig := service.DB, global.Config
    service.DB = db
    t.Cleanup(func(){service.DB=oldDB;global.Config=oldConfig})
    u := &model.User{Username:"webclient-access-test",Status:model.COMMON_STATUS_ENABLE}
    if err:=db.Create(u).Error;err!=nil {t.Fatal(err)}
    const token="webclient-session-access-test"
    row:=&model.UserToken{UserId:u.Id,Token:token,ExpiredAt:time.Now().Add(time.Hour).Unix()}
    if err:=db.Create(row).Error;err!=nil {t.Fatal(err)}
    root:=t.TempDir()
    for name,body:=range map[string]string{
        "web/index.html":"private remote-control engine",
        "web/assets/engine.js":"private engine asset",
        "web2/index.html":"private WebClient2",
        "web2/app.js":"private WebClient2 script",
    } {
        file:=filepath.Join(root,name)
        if err:=os.MkdirAll(filepath.Dir(file),0755);err!=nil {t.Fatal(err)}
        if err:=os.WriteFile(file,[]byte(body),0600);err!=nil {t.Fatal(err)}
    }
    global.Config=config.Config{
        App:config.App{WebClient:1},
        Gin:config.Gin{ResourcesPath:root},
    }
    gin.SetMode(gin.TestMode)
    router:=gin.New()
    WebInit(router)
    request:=func(method,path,cookie string)*httptest.ResponseRecorder{
        req:=httptest.NewRequest(method,path,nil)
        if cookie!="" {req.AddCookie(&http.Cookie{Name:"rustdesk_webclient_session",Value:cookie})}
        reply:=httptest.NewRecorder()
        router.ServeHTTP(reply,req)
        return reply
    }

    for _,path:=range []string{
        "/webclient2/?id=464956471&name=meili",
        "/webclient2/index.html",
        "/webclient/",
    } {
        res:=request(http.MethodGet,path,"")
        if res.Code!=http.StatusFound||!strings.HasPrefix(res.Header().Get("Location"),"/_admin/#/login?"){
            t.Fatalf("unauthorized navigation %s: %d %q",path,res.Code,res.Header().Get("Location"))
        }
        if !strings.Contains(res.Header().Get("Cache-Control"),"no-store"){t.Fatal("unauthorized page can be cached")}
    }
    link:=request(http.MethodGet,"/webclient2/?id=464956471&name=meili","")
    location:=link.Header().Get("Location")
    for _,fragment:=range []string{"id=464956471","name=meili","return=webclient2"}{
        if !strings.Contains(location,fragment){t.Fatalf("login destination lost %s: %q",fragment,location)}
    }
    for _,path:=range []string{
        "/webclient2/app.js","/webclient/assets/engine.js",
        "/webclient-config/index.js",
    } {
        if res:=request(http.MethodGet,path,"");res.Code!=http.StatusUnauthorized {
            t.Fatalf("unauthorized asset %s leaked: %d",path,res.Code)
        }
    }
    if res:=request(http.MethodGet,"/webclient2/app.js","invalid");res.Code!=http.StatusUnauthorized {
        t.Fatalf("invalid cookie was accepted: %d",res.Code)
    }

    for path,want:=range map[string]string{
        "/webclient2/":"private WebClient2",
        "/webclient2/app.js":"private WebClient2 script",
        "/webclient/":"private remote-control engine",
        "/webclient/assets/engine.js":"private engine asset",
    } {
        res:=request(http.MethodGet,path,token)
        if res.Code!=http.StatusOK||!strings.Contains(res.Body.String(),want){
            t.Fatalf("authorized asset %s: %d %q",path,res.Code,res.Body.String())
        }
        if !strings.Contains(res.Header().Get("Cache-Control"),"no-store"){t.Fatalf("protected asset %s can be cached",path)}
    }
    if res:=request(http.MethodGet,"/webclient-config/index.js",token);res.Code!=http.StatusOK {
        t.Fatalf("config asset requires a valid login: %d",res.Code)
    }

    if err:=db.Model(&model.UserToken{}).Where("id = ?",row.Id).
        Update("expired_at",time.Now().Add(-time.Minute).Unix()).Error;err!=nil{t.Fatal(err)}
    if res:=request(http.MethodGet,"/webclient2/",token);res.Code!=http.StatusFound {
        t.Fatalf("expired session still opened WebClient2: %d",res.Code)
    }
    if err:=db.Model(&model.UserToken{}).Where("id = ?",row.Id).
        Update("expired_at",time.Now().Add(time.Hour).Unix()).Error;err!=nil{t.Fatal(err)}
    if err:=db.Model(&model.User{}).Where("id = ?",u.Id).
        Update("status",model.COMMON_STATUS_DISABLED).Error;err!=nil{t.Fatal(err)}
    if res:=request(http.MethodGet,"/webclient/",token);res.Code!=http.StatusFound {
        t.Fatalf("disabled account still opened WebClient: %d",res.Code)
    }
    if err:=db.Model(&model.User{}).Where("id = ?",u.Id).
        Update("status",model.COMMON_STATUS_ENABLE).Error;err!=nil{t.Fatal(err)}
    if err:=db.Where("id = ?",row.Id).Delete(&model.UserToken{}).Error;err!=nil{t.Fatal(err)}
    if res:=request(http.MethodGet,"/webclient2/",token);res.Code!=http.StatusFound {
        t.Fatalf("revoked session still opened WebClient2: %d",res.Code)
    }
}

func TestWebClientSessionCookieUsesSecureHttpOnlySessions(t *testing.T) {
    gin.SetMode(gin.TestMode)
    engine:=gin.New()
    engine.GET("/issue",func(c *gin.Context){middleware.SetWebClientSessionCookie(c,"valid-cookie-token");c.Status(204)})
    engine.GET("/logout",func(c *gin.Context){middleware.ClearWebClientSessionCookie(c);c.Status(204)})
    for _,protocol:=range []string{"http","https"}{
        req:=httptest.NewRequest("GET","http://localhost/issue",nil)
        req.Header.Set("X-Forwarded-Proto",protocol)
        res:=httptest.NewRecorder()
        engine.ServeHTTP(res,req)
        cookies:=res.Result().Cookies()
        if len(cookies)!=1 {t.Fatalf("missing session cookie: %d",len(cookies))}
        cookie:=cookies[0]
        if cookie.Value!="valid-cookie-token"||!cookie.HttpOnly||
            cookie.SameSite!=http.SameSiteLaxMode||
            cookie.Secure!=(protocol=="https") {
            t.Fatalf("unexpected cookie attributes: %+v",cookie)
        }
        req=httptest.NewRequest("GET","http://localhost/logout",nil)
        req.Header.Set("X-Forwarded-Proto",protocol)
        res=httptest.NewRecorder()
        engine.ServeHTTP(res,req)
        cookies=res.Result().Cookies()
        if len(cookies)!=1||cookies[0].MaxAge>=0 {
            t.Fatalf("logout did not clear session cookie: %+v",cookies)
        }
    }
}
