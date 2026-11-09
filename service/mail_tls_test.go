package service

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

type smtpTLSFixture struct {
	port int
	trusted *tls.Config
	messages <-chan string
	secured <-chan bool
}

func startSMTPTLSFixture(t *testing.T,mode,username,password string) smtpTLSFixture {
	t.Helper()
	key,err:=ecdsa.GenerateKey(elliptic.P256(),rand.Reader)
	if err!=nil{t.Fatal(err)}
	template:=&x509.Certificate{
		SerialNumber:big.NewInt(33),
		NotBefore:time.Now().Add(-time.Hour),
		NotAfter:time.Now().Add(time.Hour),
		IPAddresses:[]net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign,
		ExtKeyUsage:[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid:true,
		IsCA:true,
	}
	der,err:=x509.CreateCertificate(rand.Reader,template,template,&key.PublicKey,key)
	if err!=nil{t.Fatal(err)}
	certificate,err:=x509.ParseCertificate(der)
	if err!=nil{t.Fatal(err)}
	roots:=x509.NewCertPool()
	roots.AddCert(certificate)
	listener,err:=net.Listen("tcp","127.0.0.1:0")
	if err!=nil{t.Fatal(err)}
	t.Cleanup(func(){listener.Close()})
	serverConfig:=&tls.Config{
		Certificates:[]tls.Certificate{{Certificate:[][]byte{der},PrivateKey:key}},
		MinVersion:tls.VersionTLS12,
	}
	messages:=make(chan string,1)
	secured:=make(chan bool,1)
	go func(){
		defer close(messages)
		defer close(secured)
		conn,err:=listener.Accept()
		if err!=nil{return}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(10*time.Second))
		useTLS:=mode=="tls"
		if useTLS{
			secure:=tls.Server(conn,serverConfig)
			if err:=secure.Handshake();err!=nil{return}
			conn=secure
		}
		reader:=bufio.NewReader(conn)
		writer:=bufio.NewWriter(conn)
		write:=func(line string)error{
			if _,err:=writer.WriteString(line);err!=nil{return err}
			return writer.Flush()
		}
		if err:=write("220 localhost ESMTP ready\r\n");err!=nil{return}
		var message strings.Builder
		inData:=false
		for{
			line,err:=reader.ReadString('\n')
			if err!=nil{return}
			trimmed:=strings.TrimRight(line,"\r\n")
			if inData{
				if trimmed=="."{
					inData=false
					if err:=write("250 Message queued\r\n");err!=nil{return}
				}else{message.WriteString(line)}
				continue
			}
			switch {
			case strings.HasPrefix(trimmed,"EHLO"),strings.HasPrefix(trimmed,"HELO"):
				if mode=="starttls"&&!useTLS{
					if err:=write("250-localhost\r\n250 STARTTLS\r\n");err!=nil{return}
				}else{
					if err:=write("250-localhost\r\n250 AUTH PLAIN\r\n");err!=nil{return}
				}
			case trimmed=="STARTTLS":
				if mode!="starttls"||useTLS{write("502 Not supported\r\n");return}
				if err:=write("220 Ready for TLS\r\n");err!=nil{return}
				secure:=tls.Server(conn,serverConfig)
				if err:=secure.Handshake();err!=nil{return}
				conn=secure
				useTLS=true
				reader=bufio.NewReader(conn)
				writer=bufio.NewWriter(conn)
			case strings.HasPrefix(trimmed,"AUTH PLAIN "):
				raw,err:=base64.StdEncoding.DecodeString(strings.TrimPrefix(trimmed,"AUTH PLAIN "))
				if err!=nil||!strings.HasSuffix(string(raw),"\x00"+username+"\x00"+password){
					write("535 Authentication failed\r\n")
					return
				}
				if err:=write("235 Authenticated\r\n");err!=nil{return}
			case strings.HasPrefix(trimmed,"MAIL FROM:"),strings.HasPrefix(trimmed,"RCPT TO:"):
				if err:=write("250 OK\r\n");err!=nil{return}
			case trimmed=="DATA":
				inData=true
				if err:=write("354 Go ahead\r\n");err!=nil{return}
			case trimmed=="QUIT":
				if err:=write("221 Bye\r\n");err!=nil{return}
				secured<-useTLS
				messages<-message.String()
				return
			default:
				write("500 Unknown command\r\n")
				return
			}
		}
	}()
	return smtpTLSFixture{
		port:listener.Addr().(*net.TCPAddr).Port,
		trusted:&tls.Config{
			ServerName:"127.0.0.1",RootCAs:roots,MinVersion:tls.VersionTLS12,
		},
		messages:messages,secured:secured,
	}
}

func TestSMTPRealTLSAndSTARTTLSWithAuthentication(t *testing.T){
	for _,mode:=range []string{"tls","starttls"}{
		t.Run(mode,func(t *testing.T){
			fixture:=startSMTPTLSFixture(t,mode,"smtp-test-user","smtp-test-secret")
			t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","local-tls-test-key-123456789")
			password,err:=encryptMailPassword("smtp-test-secret")
			if err!=nil{t.Fatal(err)}
			settings:=&model.SMTPSetting{
				Enabled:true,Host:"127.0.0.1",Port:fixture.port,
				Security:mode,Username:"smtp-test-user",PasswordCiphertext:password,
				SenderEmail:"staff@example.test",SenderName:"Support",
			}
			if err:=sendSMTPWithTLS(settings,"member@example.test","通知主题","安全的邮件内容",fixture.trusted);err!=nil{
				t.Fatal(err)
			}
			select{
			case message:=<-fixture.messages:
				if !strings.Contains(message,"member@example.test")||
					!strings.Contains(message,"安全的邮件内容"){
					t.Fatalf("SMTP TLS message was lost: %s",message)
				}
			case <-time.After(3*time.Second):
				t.Fatal("SMTP server did not receive an authenticated message")
			}
			select{
			case encrypted:=<-fixture.secured:
				if !encrypted{t.Fatal("SMTP message was transferred without TLS")}
			case <-time.After(3*time.Second):
				t.Fatal("SMTP server did not report TLS status")
			}
		})
	}
}

func TestSMTPStartTLSRequiredAndAuthenticationErrors(t *testing.T){
	for _,tc:=range []struct{mode,username,password string}{
		{"plain","", ""},
		{"tls","smtp-test-user","wrong-password"},
	}{
		t.Run(tc.mode,func(t *testing.T){
			t.Setenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY","local-tls-test-key-123456789")
			fixture:=startSMTPTLSFixture(t,tc.mode,"smtp-test-user","smtp-test-secret")
			password,err:=encryptMailPassword(tc.password)
			if err!=nil{t.Fatal(err)}
			security:=tc.mode
			if tc.mode=="plain"{security="starttls"}
			settings:=&model.SMTPSetting{
				Enabled:true,Host:"127.0.0.1",Port:fixture.port,
				Security:security,Username:tc.username,
				PasswordCiphertext:password,SenderEmail:"staff@example.test",
			}
			if err:=sendSMTPWithTLS(settings,"member@example.test","Test","Body",fixture.trusted);err==nil{
				t.Fatal("SMTP sent mail without the required security or credentials")
			}
		})
	}
}
