package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestMessagePusherAndServerChanHTTPDelivery(t *testing.T) {
	t.Setenv("RUSTDESK_API_PUSH_ENCRYPTION_KEY","test-only-push-delivery-secret-32")
	t.Setenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS","1")
	received:=make(chan string,4)
	server:=httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Method!="POST" { t.Errorf("unexpected method %s",r.Method) }
		switch {
		case r.URL.Path=="/v1/send":
			if r.Header.Get("Authorization")!="Bearer mpk_testing" {
				t.Errorf("MessagePusher authorization missing")
			}
			var body struct { Title string `json:"title"`; Body string `json:"body"` }
			if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil{t.Error(err)}
			if body.Title!="设备上线"||body.Body!="来源 IP：10.0.0.1（局域网地址）"{
				t.Errorf("invalid MessagePusher payload: %+v",body)
			}
			received<-"messagepusher"
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w,`{"status":"queued"}`)
		case r.URL.Path=="/SCT_testing.send":
			if err:=r.ParseForm();err!=nil{t.Error(err)}
			if r.Form.Get("title")!="设备上线"||r.Form.Get("desp")==""{t.Error("invalid Server酱 payload")}
			received<-"serverchan"
			w.Header().Set("Content-Type","application/json")
			fmt.Fprint(w,`{"code":0,"data":{"pushid":"sample"}}`)
		default:
			t.Errorf("unexpected path: %s",r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	original:=pushHTTP
	pushHTTP=server.Client()
	pushHTTP.Timeout=5*time.Second
	defer func(){pushHTTP=original}()
	secret,err:=encryptPushSecret("mpk_testing")
	if err!=nil{t.Fatal(err)}
	msg:=&model.PushChannel{Kind:"messagepusher",Enabled:true,BaseURL:server.URL,KeyCiphertext:secret}
	if err:=sendPushHTTP(msg,"设备上线","来源 IP：10.0.0.1（局域网地址）");err!=nil{t.Fatal(err)}
	secret,err=encryptPushSecret("SCT_testing")
	if err!=nil{t.Fatal(err)}
	chanID:=&model.PushChannel{Kind:"serverchan",Enabled:true,BaseURL:server.URL,KeyCiphertext:secret}
	if err:=sendPushHTTP(chanID,"设备上线","连接事件详情");err!=nil{t.Fatal(err)}
	close(received)
	kinds:=[]string{}
	for x:=range received{kinds=append(kinds,x)}
	if strings.Join(kinds,",")!="messagepusher,serverchan"{t.Fatalf("missing sends: %v",kinds)}
}

func TestPersonalSMTPDeliveryOverTrustedLocalConnection(t *testing.T) {
	t.Setenv("RUSTDESK_API_PUSH_ENCRYPTION_KEY","test-only-push-delivery-secret-32")
	t.Setenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS","1")
	port,message:=startMockSMTP(t)
	channel:=&model.PushChannel{
		Kind:"smtp",Enabled:true,SMTPHost:"127.0.0.1",SMTPPort:port,
		SMTPSecurity:"none",SMTPFrom:"alerts@example.test",SMTPTo:"owner@example.test",
	}
	if err:=sendPushSMTP(channel,"设备离线","机器 A 失去心跳。IP：10.0.0.12（局域网地址）");err!=nil{
		t.Fatal(err)
	}
	select{
	case body:=<-message:
		if !strings.Contains(body,"机器 A 失去心跳")||!strings.Contains(body,"owner@example.test"){
			t.Fatalf("unexpected SMTP message: %s",body)
		}
	case <-time.After(3*time.Second):t.Fatal("SMTP delivery timed out")
	}
}
