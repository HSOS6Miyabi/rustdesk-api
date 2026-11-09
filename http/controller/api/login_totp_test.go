package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func totpCodeForHTTPTest(secret string,step int64) string {
	decoded,_:=base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:],uint64(step))
	mac:=hmac.New(sha1.New,decoded)
	mac.Write(counter[:])
	digest:=mac.Sum(nil)
	offset:=int(digest[len(digest)-1]&15)
	number:=binary.BigEndian.Uint32(digest[offset:offset+4])&0x7fffffff
	return fmt.Sprintf("%06d",number%1000000)
}

func TestNativeRustDeskLoginRequiresTOTP(t *testing.T) {
	db,err:=gorm.Open(sqlite.Open("file:native_totp_login?mode=memory&cache=shared"),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	conn,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err:=db.AutoMigrate(&model.User{},&model.Peer{},&model.UserToken{},
		&model.LoginLog{},&model.UserTOTP{},&model.TOTPRecoveryCode{},
		&model.TOTPLoginChallenge{});err!=nil{t.Fatal(err)}
	oldDB,oldService,oldCfg,oldJwt,oldLogger:=service.DB,service.AllService,service.Config,service.Jwt,service.Logger
	oldGlobalConfig,oldLimiter,oldGlobalLogger,oldValidator:=global.Config,global.LoginLimiter,global.Logger,global.Validator
	defer func(){
		service.DB,service.AllService,service.Config,service.Jwt,service.Logger=oldDB,oldService,oldCfg,oldJwt,oldLogger
		global.Config,global.LoginLimiter,global.Logger,global.Validator=oldGlobalConfig,oldLimiter,oldGlobalLogger,oldValidator
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
	t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","native-totp-integration-test-secret-12345")
	hash,err:=utils.EncryptPassword("TestPassword123")
	if err!=nil{t.Fatal(err)}
	user:=&model.User{Username:"team-member",Email:"owner@example.test",Status:model.COMMON_STATUS_ENABLE,Password:hash}
	if err:=db.Create(user).Error;err!=nil{t.Fatal(err)}
	setup,err:=service.BeginTOTPSetup(user.Id)
	if err!=nil{t.Fatal(err)}
	code:=totpCodeForHTTPTest(setup.Secret,time.Now().Unix()/30)
	if _,err:=service.ActivateTOTP(user.Id,code);err!=nil{t.Fatal(err)}
	gin.SetMode(gin.TestMode)
	router:=gin.New()
	router.POST("/api/login",(&Login{}).Login)
	send:=func(body interface{})*httptest.ResponseRecorder{
		t.Helper()
		raw,err:=json.Marshal(body)
		if err!=nil{t.Fatal(err)}
		rec:=httptest.NewRecorder()
		req:=httptest.NewRequest(http.MethodPost,"/api/login",bytes.NewReader(raw))
		req.Header.Set("Content-Type","application/json")
		router.ServeHTTP(rec,req)
		return rec
	}
	first:=send(map[string]interface{}{"username":user.Username,"password":"TestPassword123","type":"account"})
	if first.Code!=http.StatusOK{t.Fatalf("first login: %d %s",first.Code,first.Body.String())}
	var challenge struct {
		Type string `json:"type"`
		TfaType string `json:"tfa_type"`
		Secret string `json:"secret"`
		AccessToken string `json:"access_token"`
		User struct{Name string `json:"name"`} `json:"user"`
	}
	if err:=json.Unmarshal(first.Body.Bytes(),&challenge);err!=nil{t.Fatal(err)}
	if challenge.Type!="email_check"||challenge.TfaType!="tfa_check"||challenge.Secret==""||
		challenge.AccessToken!=""||challenge.User.Name!=user.Username {
		t.Fatalf("native client did not receive compatible two-factor challenge: %s",first.Body.String())
	}
	next:=totpCodeForHTTPTest(setup.Secret,time.Now().Unix()/30+1)
	second:=send(map[string]interface{}{
		"username":user.Username,"type":"email_code","secret":challenge.Secret,
		"tfaCode":next,"verificationCode":next,
	})
	if second.Code!=http.StatusOK{t.Fatalf("valid TOTP login rejected: %d %s",second.Code,second.Body.String())}
	var success struct{Type string `json:"type"`;AccessToken string `json:"access_token"`}
	if err:=json.Unmarshal(second.Body.Bytes(),&success);err!=nil{t.Fatal(err)}
	if success.Type!="access_token"||success.AccessToken==""{
		t.Fatalf("native client did not receive a login session: %s",second.Body.String())
	}
	var total int64
	if err:=db.Model(&model.UserToken{}).Where("user_id = ?",user.Id).Count(&total).Error;err!=nil||total!=1{
		t.Fatalf("unexpected login session count: %d %v",total,err)
	}
	repeated:=send(map[string]interface{}{
		"username":user.Username,"type":"email_code","secret":challenge.Secret,"tfaCode":next,
	})
	if repeated.Code==http.StatusOK{t.Fatalf("second-factor challenge was reused: %s",repeated.Body.String())}
	if err:=db.Model(&model.UserToken{}).Where("user_id = ?",user.Id).Count(&total).Error;err!=nil||total!=1{
		t.Fatalf("reused challenge issued another session: %d %v",total,err)
	}
}
