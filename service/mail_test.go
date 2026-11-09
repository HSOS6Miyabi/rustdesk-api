package service

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func startMockSMTP(t *testing.T) (int,<-chan string) {
	t.Helper()
	listener,err:=net.Listen("tcp","127.0.0.1:0")
	if err!=nil{t.Fatal(err)}
	t.Cleanup(func(){listener.Close()})
	result:=make(chan string,1)
	go func(){
		defer close(result)
		conn,err:=listener.Accept()
		if err!=nil{result<-"";return}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(10*time.Second))
		reader:=bufio.NewReader(conn)
		writer:=bufio.NewWriter(conn)
		write:=func(message string) {fmt.Fprint(writer,message);writer.Flush()}
		write("220 localhost ESMTP ready\r\n")
		var content strings.Builder
		readingMessage:=false
		for{
			line,err:=reader.ReadString('\n')
			if err!=nil{result<-content.String();return}
			trimmed:=strings.TrimRight(line,"\r\n")
			if readingMessage{
				if trimmed=="."{
					readingMessage=false
					write("250 OK\r\n")
				}else{content.WriteString(line)}
				continue
			}
			switch {
			case strings.HasPrefix(line,"EHLO"),strings.HasPrefix(line,"HELO"):
				write("250 localhost\r\n")
			case strings.HasPrefix(line,"MAIL FROM:"),strings.HasPrefix(line,"RCPT TO:"):
				write("250 OK\r\n")
			case strings.HasPrefix(line,"DATA"):
				readingMessage=true
				write("354 send message\r\n")
			case strings.HasPrefix(line,"QUIT"):
				write("221 goodbye\r\n")
				result<-content.String()
				return
			default:
				write("502 unrecognized command\r\n")
			}
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port,result
}

func TestSMTPSettingsAndLocalMailDelivery(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.SMTPSetting{});err!=nil{t.Fatal(err)}
	t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","test-only-password-encryption-secret-1234")
	if _,err:=SaveMailSettings(MailSettingsInput{Enabled:true,Port:587,Host:"example.com",
		SenderEmail:"bad-mail",Security:"starttls"});err==nil{t.Fatal("invalid sender accepted")}
	if _,err:=SaveMailSettings(MailSettingsInput{Enabled:true,Port:587,Host:"example.com",
		SenderEmail:"admin@example.com",Security:"bad"});err==nil{t.Fatal("invalid SMTP security accepted")}
	settings,err:=SaveMailSettings(MailSettingsInput{
		Enabled:true,Host:"localhost",Port:587,Security:"none",
		SenderEmail:"admin@example.com",SenderName:"RustDesk 团队",
		Username:"smtp-account",Password:"private-smtp-password",
		BaseURL:"https://remote.example.com",
	})
	if err!=nil{t.Fatal(err)}
	if !settings.PasswordConfigured||settings.BaseURL!="https://remote.example.com"{t.Fatalf("SMTP config not saved: %+v",settings)}
	var record model.SMTPSetting
	if err:=DB.First(&record,1).Error;err!=nil{t.Fatal(err)}
	if strings.Contains(record.PasswordCiphertext,"private-smtp-password"){
		t.Fatal("SMTP credentials stored as plain text")
	}
	clear,err:=decryptMailPassword(record.PasswordCiphertext)
	if err!=nil||clear!="private-smtp-password"{t.Fatalf("cannot decrypt SMTP credentials: %v",err)}
	port,message:=startMockSMTP(t)
	settings,err=SaveMailSettings(MailSettingsInput{
		Enabled:true,Host:"127.0.0.1",Port:port,Security:"none",
		SenderEmail:"team@example.com",SenderName:"远程支持",
		BaseURL:"https://remote.example.com",
	})
	if err!=nil{t.Fatal(err)}
	if !settings.PasswordConfigured{t.Fatal("updating non-credential fields cleared stored secret")}
	if err:=SendMail("user@example.com","邮件测试","中文内容与设备信息");err!=nil{
		t.Fatal(err)
	}
	select{
	case body:=<-message:
		if !strings.Contains(body,"user@example.com")||!strings.Contains(body,"中文内容与设备信息")||
			!strings.Contains(body,"Subject: =?utf-8?"){
			t.Fatalf("unexpected SMTP message: %s",body)
		}
	case <-time.After(3*time.Second):t.Fatal("SMTP test did not finish")
	}
	settings,err=SaveMailSettings(MailSettingsInput{
		Enabled:false,Host:"localhost",Port:port,Security:"none",
		SenderEmail:"team@example.com",ClearPassword:true,
	})
	if err!=nil||settings.PasswordConfigured{t.Fatalf("password was not cleared: %+v %v",settings,err)}
	if err:=SendMail("user@example.com","should fail","mail disabled");err==nil{
		t.Fatal("disabled SMTP service attempted to send")
	}
}

func TestSMTPPasswordRequiresEncryptionKey(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.SMTPSetting{});err!=nil{t.Fatal(err)}
	t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","")
	if _,err:=SaveMailSettings(MailSettingsInput{
		Host:"localhost",Port:25,Security:"none",
		SenderEmail:"team@example.com",Password:"do-not-store-plain-text",
	});err==nil{t.Fatal("SMTP password saved without a configured encryption key")}
}
