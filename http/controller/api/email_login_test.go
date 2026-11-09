package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func emailLoginSMTPServer(t *testing.T) (int,<-chan string) {
	t.Helper()
	listener,err:=net.Listen("tcp","127.0.0.1:0")
	if err!=nil{t.Fatal(err)}
	t.Cleanup(func(){listener.Close()})
	mail:=make(chan string,1)
	go func(){
		conn,err:=listener.Accept()
		if err!=nil{mail<-"";return}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10*time.Second))
		reader:=bufio.NewReader(conn)
		writer:=bufio.NewWriter(conn)
		say:=func(response string){writer.WriteString(response);writer.Flush()}
		say("220 localhost ESMTP ready\r\n")
		var message strings.Builder
		readingData:=false
		for{
			line,err:=reader.ReadString('\n')
			if err!=nil{mail<-message.String();return}
			if readingData{
				if strings.TrimRight(line,"\r\n")=="."{
					readingData=false
					say("250 stored\r\n")
				}else{message.WriteString(line)}
				continue
			}
			switch {
			case strings.HasPrefix(line,"EHLO"),strings.HasPrefix(line,"HELO"):
				say("250 localhost\r\n")
			case strings.HasPrefix(line,"MAIL FROM:"),strings.HasPrefix(line,"RCPT TO:"):
				say("250 accepted\r\n")
			case strings.HasPrefix(line,"DATA"):
				readingData=true
				say("354 content follows\r\n")
			case strings.HasPrefix(line,"QUIT"):
				say("221 bye\r\n")
				mail<-message.String()
				return
			default:
				say("502 command unsupported\r\n")
			}
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port,mail
}

func TestNativeRustDeskEmailCodeLogin(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:native_email_code_login?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.UserToken{},&model.LoginLog{},
		&model.EmailLoginSetting{},&model.EmailCodeChallenge{},&model.TOTPLoginChallenge{},
		&model.UserTOTP{},&model.SMTPSetting{});err!=nil{t.Fatal(err)}
	oldDB,oldSvc,oldCfg,oldJWT,oldLogger:=service.DB,service.AllService,service.Config,service.Jwt,service.Logger
	oldGlobal,oldLimiter,oldGLogger,oldValidator:=global.Config,global.LoginLimiter,global.Logger,global.Validator
	defer func(){
		service.DB,service.AllService,service.Config,service.Jwt,service.Logger=oldDB,oldSvc,oldCfg,oldJWT,oldLogger
		global.Config,global.LoginLimiter,global.Logger,global.Validator=oldGlobal,oldLimiter,oldGLogger,oldValidator
	}()
	conf:=&config.Config{}
	logger:=log.New()
	service.New(conf,db,logger,jwt.NewJwt("",0),nil)
	service.AllService.UserService=&service.UserService{}
	service.AllService.PeerService=&service.PeerService{}
	global.Config=*conf
	global.Logger=logger
	global.LoginLimiter=utils.NewLoginLimiter(utils.SecurityPolicy{})
	global.ApiInitValidator()
	pwd,err:=utils.EncryptPassword("TestPassword123")
	if err!=nil{t.Fatal(err)}
	user:=&model.User{
		Username:"email-member",Email:"email-member@example.test",
		Password:pwd,Status:model.COMMON_STATUS_ENABLE,
	}
	if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
	if err:=db.Create(&model.EmailLoginSetting{
		UserId:user.Id,VerifiedEmail:user.Email,Enabled:true,
	}).Error;err!=nil{t.Fatal(err)}
	port,received:=emailLoginSMTPServer(t)
	if _,err:=service.SaveMailSettings(service.MailSettingsInput{
		Enabled:true,Host:"127.0.0.1",Port:port,Security:"none",
		SenderEmail:"support@example.test",
	});err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/login",(&Login{}).Login)
	send:=func(payload interface{}) *httptest.ResponseRecorder {
		t.Helper()
		raw,err:=json.Marshal(payload)
		if err!=nil{t.Fatal(err)}
		rec:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodPost,"/api/login",bytes.NewReader(raw))
		req.Header.Set("Content-Type","application/json")
		router.ServeHTTP(rec,req)
		return rec
	}
	first:=send(map[string]interface{}{
		"username":user.Username,"password":"TestPassword123","type":"account",
	})
	if first.Code!=http.StatusOK{t.Fatalf("native login did not request email code: %d %s",first.Code,first.Body.String())}
	var challenge struct{
		Type string `json:"type"`
		TfaType string `json:"tfa_type"`
		Secret string `json:"secret"`
		AccessToken string `json:"access_token"`
	}
	if err:=json.Unmarshal(first.Body.Bytes(),&challenge);err!=nil{t.Fatal(err)}
	if challenge.Type!="email_check"||challenge.TfaType!="email_check"||
		challenge.Secret==""||challenge.AccessToken!=""{
		t.Fatalf("invalid email verification response: %s",first.Body.String())
	}
	var body string
	select{case body=<-received:case <-time.After(5*time.Second):t.Fatal("email code never arrived")}
	match:=regexp.MustCompile(`验证码：([0-9]{6})`).FindStringSubmatch(body)
	if len(match)!=2{t.Fatalf("verification code not found: %q",body)}
	second:=send(map[string]interface{}{
		"username":user.Username,"type":"email_code",
		"secret":challenge.Secret,"verificationCode":match[1],
	})
	if second.Code!=http.StatusOK{t.Fatalf("correct email code was refused: %d %s",second.Code,second.Body.String())}
	var success struct{Type string `json:"type"`;AccessToken string `json:"access_token"`}
	if err:=json.Unmarshal(second.Body.Bytes(),&success);err!=nil{t.Fatal(err)}
	if success.Type!="access_token"||success.AccessToken==""{
		t.Fatalf("no session created after email code: %s",second.Body.String())
	}
	var count int64
	if err:=db.Model(&model.UserToken{}).Where("user_id = ?",user.Id).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("wrong login session count: %d %v",count,err)
	}
	again:=send(map[string]interface{}{
		"username":user.Username,"type":"email_code",
		"secret":challenge.Secret,"verificationCode":match[1],
	})
	if again.Code==http.StatusOK{t.Fatalf("used verification code accepted: %s",again.Body.String())}
	if err:=db.Model(&model.UserToken{}).Where("user_id = ?",user.Id).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("used code created additional session: %d %v",count,err)
	}
}
