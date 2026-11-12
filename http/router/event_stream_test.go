package router

import (
    "bufio"
    "context"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/http/controller/api"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "github.com/lejianwen/rustdesk-api/v2/service"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func TestSSEEndpointRequiresScopedKeyAndClosesOnDisconnect(t *testing.T) {
    db,err:=gorm.Open(sqlite.Open("file:event_stream_routes?mode=memory&cache=shared"),&gorm.Config{})
    if err!=nil{t.Fatal(err)}
    sqlDB,err:=db.DB();if err!=nil{t.Fatal(err)}
    defer sqlDB.Close()
    sqlDB.SetMaxOpenConns(1)
    if err:=db.AutoMigrate(&model.User{},&model.APIToken{});err!=nil{t.Fatal(err)}
    before:=service.DB
    service.DB=db
    defer func(){service.DB=before}()
    user:=&model.User{Username:"stream-router-test",Status:model.COMMON_STATUS_ENABLE}
    if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
    _,key,err:=service.IssueAPIToken(user.Id,"Stream reader",[]string{"events.read"},0)
    if err!=nil{t.Fatal(err)}
    _,bad,err:=service.IssueAPIToken(user.Id,"Device reader",[]string{"devices.read"},0)
    if err!=nil{t.Fatal(err)}
    gin.SetMode(gin.TestMode)
    engine:=gin.New()
    engine.GET("/api/events/v1/stream",(&api.EventStream{}).Stream)
    server:=httptest.NewServer(engine)
    defer server.Close()
    check:=func(token string,expected int){
        request,_:=http.NewRequest(http.MethodGet,server.URL+"/api/events/v1/stream",nil)
        if token!=""{request.Header.Set("Authorization","Bearer "+token)}
        response,err:=server.Client().Do(request)
        if err!=nil{t.Fatal(err)}
        response.Body.Close()
        if response.StatusCode!=expected{t.Fatalf("status %d expected %d",response.StatusCode,expected)}
    }
    check("",http.StatusUnauthorized)
    check(bad,http.StatusUnauthorized)
    check(key+"&invalid",http.StatusUnauthorized)

    ctx,cancel:=context.WithCancel(context.Background())
    defer cancel()
    req,_:=http.NewRequestWithContext(ctx,http.MethodGet,server.URL+"/api/events/v1/stream?events=device.online",nil)
    req.Header.Set("Authorization","Bearer "+key)
    res,err:=server.Client().Do(req)
    if err!=nil{t.Fatal(err)}
    defer res.Body.Close()
    if res.StatusCode!=http.StatusOK||!strings.HasPrefix(res.Header.Get("Content-Type"),"text/event-stream"){
        t.Fatalf("invalid SSE response: %d %s",res.StatusCode,res.Header.Get("Content-Type"))
    }
    reader:=bufio.NewReader(res.Body)
    if line,err:=reader.ReadString('\n');err!=nil||line!="retry: 3000\n"{
        t.Fatalf("SSE prelude: %q %v",line,err)
    }
    service.PublishLivePushForUsers([]uint{user.Id},"device.online",[]byte("{\"type\":\"device.online\"}"))
    var eventLine string
    for i:=0;i<4;i++{
        line,err:=reader.ReadString('\n')
        if err!=nil{t.Fatal(err)}
        if strings.HasPrefix(line,"event: "){eventLine=line;break}
    }
    if eventLine!="event: device.online\n"{t.Fatalf("missing SSE event: %q",eventLine)}
    cancel()
    deadline:=time.Now().Add(2*time.Second)
    for time.Now().Before(deadline) && service.HasLivePushSubscribers("device.online") {
        time.Sleep(10*time.Millisecond)
    }
    if service.HasLivePushSubscribers("device.online"){t.Fatal("disconnected listener not removed")}
}
