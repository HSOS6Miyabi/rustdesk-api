package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type MailSettingsInput struct {
	Enabled bool `json:"enabled"`
	Host string `json:"host"`
	Port int `json:"port"`
	Security string `json:"security"`
	Username string `json:"username"`
	Password string `json:"password"`
	ClearPassword bool `json:"clear_password"`
	SenderEmail string `json:"sender_email"`
	SenderName string `json:"sender_name"`
	BaseURL string `json:"base_url"`
}

type PublicMailSettings struct {
	Enabled bool `json:"enabled"`
	Host string `json:"host"`
	Port int `json:"port"`
	Security string `json:"security"`
	Username string `json:"username"`
	PasswordConfigured bool `json:"password_configured"`
	SenderEmail string `json:"sender_email"`
	SenderName string `json:"sender_name"`
	BaseURL string `json:"base_url"`
}

func mailEncryptionKey() ([]byte,error) {
	secret:=os.Getenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY")
	if len(secret)<16{return nil,errors.New("set RUSTDESK_API_MAIL_ENCRYPTION_KEY to at least 16 characters to store an SMTP password")}
	derived:=sha256.Sum256([]byte(secret))
	return derived[:],nil
}

func encryptMailPassword(plain string) (string,error) {
	key,err:=mailEncryptionKey()
	if err!=nil{return "",err}
	block,err:=aes.NewCipher(key)
	if err!=nil{return "",err}
	gcm,err:=cipher.NewGCM(block)
	if err!=nil{return "",err}
	nonce:=make([]byte,gcm.NonceSize())
	if _,err:=rand.Read(nonce);err!=nil{return "",err}
	return base64.RawStdEncoding.EncodeToString(gcm.Seal(nonce,nonce,[]byte(plain),nil)),nil
}

func decryptMailPassword(ciphertext string) (string,error) {
	if ciphertext==""{return "",nil}
	key,err:=mailEncryptionKey()
	if err!=nil{return "",err}
	payload,err:=base64.RawStdEncoding.DecodeString(ciphertext)
	if err!=nil{return "",err}
	block,err:=aes.NewCipher(key)
	if err!=nil{return "",err}
	gcm,err:=cipher.NewGCM(block)
	if err!=nil{return "",err}
	if len(payload)<gcm.NonceSize(){return "",errors.New("invalid encrypted SMTP credentials")}
	plain,err:=gcm.Open(nil,payload[:gcm.NonceSize()],payload[gcm.NonceSize():],nil)
	if err!=nil{return "",err}
	return string(plain),nil
}

func currentMailSettings() (*model.SMTPSetting,error) {
	settings:=&model.SMTPSetting{}
	err:=DB.Where("id = ?",1).First(settings).Error
	if errors.Is(err,gorm.ErrRecordNotFound){
		return &model.SMTPSetting{IdModel:model.IdModel{Id:1},Port:587,Security:"starttls"},nil
	}
	return settings,err
}

func GetPublicMailSettings() (*PublicMailSettings,error){
	settings,err:=currentMailSettings()
	if err!=nil{return nil,err}
	return &PublicMailSettings{
		Enabled:settings.Enabled,Host:settings.Host,Port:settings.Port,
		Security:settings.Security,Username:settings.Username,
		PasswordConfigured:settings.PasswordCiphertext!="",
		SenderEmail:settings.SenderEmail,SenderName:settings.SenderName,
		BaseURL:settings.BaseURL,
	},nil
}

func normalizeMailSettings(input *MailSettingsInput) error {
	input.Host=strings.TrimSpace(input.Host)
	input.Username=strings.TrimSpace(input.Username)
	input.SenderName=strings.TrimSpace(input.SenderName)
	input.SenderEmail=strings.TrimSpace(input.SenderEmail)
	input.BaseURL=strings.TrimRight(strings.TrimSpace(input.BaseURL),"/")
	input.Security=strings.ToLower(strings.TrimSpace(input.Security))
	if input.Port==0{input.Port=587}
	if input.Security==""{input.Security="starttls"}
	if input.Security!="none"&&input.Security!="starttls"&&input.Security!="tls"{
		return errors.New("SMTP security must be none, starttls or tls")
	}
	if input.Port<1||input.Port>65535{return errors.New("invalid SMTP port")}
	if len(input.Host)>253||len(input.Username)>253||len(input.SenderName)>100||
		len(input.SenderEmail)>253||len(input.BaseURL)>512||
		strings.ContainsAny(input.Host+input.Username+input.SenderName+input.SenderEmail,"\r\n"){
		return errors.New("invalid SMTP configuration")
	}
	if input.Enabled{
		if input.Host==""||input.SenderEmail==""{return errors.New("SMTP host and sender email are required")}
	}
	if input.SenderEmail!=""{
		address,err:=mail.ParseAddress(input.SenderEmail)
		if err!=nil||address.Address!=input.SenderEmail{return errors.New("invalid sender email address")}
	}
	if input.BaseURL!=""{
		base,err:=url.Parse(input.BaseURL)
		if err!=nil||base.Host==""||(base.Scheme!="http"&&base.Scheme!="https")||
			base.User!=nil||base.RawQuery!=""||base.Fragment!=""{
			return errors.New("invalid public console URL")
		}
	}
	return nil
}

func SaveMailSettings(input MailSettingsInput) (*PublicMailSettings,error){
	if err:=normalizeMailSettings(&input);err!=nil{return nil,err}
	existing,err:=currentMailSettings()
	if err!=nil{return nil,err}
	password:=existing.PasswordCiphertext
	if input.ClearPassword{password=""}
	if input.Password!=""{
		password,err=encryptMailPassword(input.Password)
		if err!=nil{return nil,err}
	}
	if len(input.Password)>1024{return nil,errors.New("SMTP password is too long")}
	settings:=&model.SMTPSetting{
		IdModel:model.IdModel{Id:1},Enabled:input.Enabled,Host:input.Host,Port:input.Port,
		Security:input.Security,Username:input.Username,PasswordCiphertext:password,
		SenderEmail:input.SenderEmail,SenderName:input.SenderName,BaseURL:input.BaseURL,
	}
	err=DB.Transaction(func(tx *gorm.DB)error{
		var count int64
		if err:=tx.Model(&model.SMTPSetting{}).Where("id = ?",1).Count(&count).Error;err!=nil{return err}
		if count==0{return tx.Create(settings).Error}
		return tx.Model(&model.SMTPSetting{}).Where("id = ?",1).
			Updates(map[string]interface{}{
				"enabled":settings.Enabled,"host":settings.Host,"port":settings.Port,
				"security":settings.Security,"username":settings.Username,
				"password_ciphertext":settings.PasswordCiphertext,
				"sender_email":settings.SenderEmail,"sender_name":settings.SenderName,
				"base_url":settings.BaseURL,
			}).Error
	})
	if err!=nil{return nil,err}
	return GetPublicMailSettings()
}

func sendSMTP(settings *model.SMTPSetting,to,subject,body string) error{
	if settings==nil{return errors.New("SMTP settings are required")}
	return sendSMTPWithTLS(settings,to,subject,body,&tls.Config{
		ServerName:settings.Host,MinVersion:tls.VersionTLS12,
	})
}

// sendSMTPWithTLS shares the production SMTP implementation with tests that
// supply a private certificate authority for their isolated mail server.
func sendSMTPWithTLS(settings *model.SMTPSetting,to,subject,body string,tlsOptions *tls.Config) error{
	if settings==nil||tlsOptions==nil{return errors.New("SMTP TLS settings are required")}
	if tlsOptions.MinVersion<tls.VersionTLS12{return errors.New("SMTP requires TLS 1.2 or newer")}
	if !settings.Enabled{return errors.New("SMTP email service is disabled")}
	recipient,err:=mail.ParseAddress(strings.TrimSpace(to))
	if err!=nil||recipient.Address!=strings.TrimSpace(to){return errors.New("invalid recipient email address")}
	if strings.ContainsAny(subject,"\r\n")||strings.ContainsAny(body,"\x00"){
		return errors.New("invalid email content")
	}
	addr:=net.JoinHostPort(settings.Host,strconv.Itoa(settings.Port))
	dialer:=&net.Dialer{Timeout:8*time.Second}
	var conn net.Conn
	if settings.Security=="tls"{
		conn,err=tls.DialWithDialer(dialer,"tcp",addr,tlsOptions)
	}else{
		conn,err=dialer.Dial("tcp",addr)
	}
	if err!=nil{return err}
	defer conn.Close()
	if err:=conn.SetDeadline(time.Now().Add(15*time.Second));err!=nil{return err}
	client,err:=smtp.NewClient(conn,settings.Host)
	if err!=nil{return err}
	defer client.Close()
	if settings.Security=="starttls"{
		if ok,_:=client.Extension("STARTTLS");!ok{return errors.New("SMTP server does not support STARTTLS")}
		if err:=client.StartTLS(tlsOptions);err!=nil{return err}
	}
	if settings.Username!=""{
		password,err:=decryptMailPassword(settings.PasswordCiphertext)
		if err!=nil{return err}
		if err:=client.Auth(smtp.PlainAuth("",settings.Username,password,settings.Host));err!=nil{return err}
	}
	if err:=client.Mail(settings.SenderEmail);err!=nil{return err}
	if err:=client.Rcpt(recipient.Address);err!=nil{return err}
	stream,err:=client.Data()
	if err!=nil{return err}
	sender:=(&mail.Address{Name:settings.SenderName,Address:settings.SenderEmail}).String()
	content:=fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s",
		sender,recipient.Address,mime.QEncoding.Encode("utf-8",subject),
		time.Now().UTC().Format(time.RFC1123Z),strings.ReplaceAll(strings.ReplaceAll(body,"\r\n","\n"),"\n","\r\n"))
	if _,err:=io.WriteString(stream,content);err!=nil{stream.Close();return err}
	if err:=stream.Close();err!=nil{return err}
	return client.Quit()
}

func SendMail(to,subject,body string) error {
	settings,err:=currentMailSettings()
	if err!=nil{return err}
	return sendSMTP(settings,to,subject,body)
}
