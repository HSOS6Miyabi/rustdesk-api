package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const pushMaxAttempts = 5

var pushDeliveryWake = make(chan struct{}, 2)

func wakePushDelivery() {
	for i := 0; i < 2; i++ {
		select { case pushDeliveryWake <- struct{}{}: default: }
	}
}

func eventHash(reference string) string {
	hash := sha256.Sum256([]byte(reference))
	return hex.EncodeToString(hash[:])
}

func QueuePushForUsers(event PushEvent, users []uint) error {
	if DB == nil { return nil }
	eventName := pushEventName(event.Code)
	if eventName == "" { return nil }
	seen := make(map[uint]bool)
	now := time.Now().Unix()
	title, body := "", ""
	for _, userID := range users {
		if userID == 0 || seen[userID] { continue }
		seen[userID] = true
		var channels []model.PushChannel
		if err := DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&channels).Error; err != nil {
			return err
		}
		for _, channel := range channels {
			if !pushSubscribed(&channel, event.Code) { continue }
			if title == "" { title, body = formatPushMessage(event) }
			if title == "" { continue }
			key := event.Reference
			if key == "" { key = fmt.Sprintf("%d:%d", now, time.Now().UnixNano()) }
			record := &model.PushDelivery{
				UserId: userID, ChannelId: channel.Id, EventType: event.Code,
				EventKey: eventHash(event.Code + ":" + key), Title: title, Body: body,
				Status: "pending", NextAttemptAt: now,
			}
			if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(record).Error; err != nil {
				return err
			}
		}
	}
	wakePushDelivery()
	return nil
}

func QueuePushTest(userID, channelID uint) error {
	if userID == 0 || channelID == 0 { return errors.New("无效的推送渠道") }
	var channel model.PushChannel
	if err := DB.Where("id = ? AND user_id = ?", channelID, userID).First(&channel).Error; err != nil {
		return errors.New("推送渠道不存在")
	}
	if !channel.Enabled { return errors.New("请先启用该渠道") }
	now := time.Now()
	record := &model.PushDelivery{
		UserId: userID, ChannelId: channelID, EventType: "test.manual",
		EventKey: eventHash(fmt.Sprintf("test:%d:%d", channelID, now.UnixNano())),
		Title: "[RustDesk] 测试通知",
		Body: "通知渠道测试\n事件：手动测试\n发送时间：" + now.Format(time.RFC3339) + "\n渠道：" + channel.Name,
		Status: "pending", NextAttemptAt: now.Unix(),
	}
	if err := DB.Create(record).Error; err != nil { return err }
	wakePushDelivery()
	return nil
}

// The resolver and the dial operation share the same checked IP address.
// Private addresses are allowed only when the operator explicitly opts in.
func pushDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil { return nil, err }
	if _, err := strconv.Atoi(port); err != nil { return nil, errors.New("invalid destination port") }
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil { return nil, err }
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	for _, entry := range ips {
		ip := entry.IP
		if !pushAllowPrivate() && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
			ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), port))
		if dialErr == nil { return conn, nil }
		err = dialErr
	}
	if err == nil { err = errors.New("destination IP is not permitted") }
	return nil, err
}

var pushHTTP = &http.Client{
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		Proxy: nil, DialContext: pushDialContext,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns: 20, MaxIdleConnsPerHost: 4, IdleConnTimeout: 30 * time.Second,
	},
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

func sendPushHTTP(channel *model.PushChannel, title, body string) error {
	secret, err := decryptPushSecret(channel.KeyCiphertext)
	if err != nil { return err }
	raw, err := validPushURL(channel.BaseURL)
	if err != nil { return err }
	endpoint := strings.TrimRight(raw, "/")
	var payload io.Reader
	contentType := ""
	authHeader := ""
	if channel.Kind == "messagepusher" {
		endpoint += "/v1/send"
		content, err := json.Marshal(map[string]string{"title": title, "body": body})
		if err != nil { return err }
		payload = bytes.NewReader(content)
		contentType = "application/json"
		authHeader = "Bearer " + secret
	} else {
		endpoint += "/" + url.PathEscape(secret) + ".send"
		form := url.Values{"title": {title}, "desp": {body}}
		payload = strings.NewReader(form.Encode())
		contentType = "application/x-www-form-urlencoded"
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, payload)
	if err != nil { return err }
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", "RustDesk-API-Push/1.0")
	if authHeader != "" { req.Header.Set("Authorization", authHeader) }
	res, err := pushHTTP.Do(req)
	if err != nil { return errors.New("推送服务连接失败或请求超时") }
	defer res.Body.Close()
	result, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	if channel.Kind == "messagepusher" {
		if res.StatusCode != http.StatusAccepted { return fmt.Errorf("MessagePusher HTTP %d: %s", res.StatusCode, safePushError(result)) }
	} else {
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("Server酱 HTTP %d: %s", res.StatusCode, safePushError(result))
		}
		var status struct {
			Code *int `json:"code"`
			ErrNo *int `json:"errno"`
		}
		if json.Unmarshal(result, &status) == nil {
			if status.Code != nil && *status.Code != 0 { return fmt.Errorf("Server酱返回错误码 %d", *status.Code) }
			if status.ErrNo != nil && *status.ErrNo != 0 { return fmt.Errorf("Server酱返回错误码 %d", *status.ErrNo) }
		}
	}
	return nil
}

// Do not persist raw provider responses: they can echo authentication secrets.
func safePushError(body []byte) string {
	if len(body) == 0 { return "request rejected" }
	var response struct{ Message string `json:"message"` }
	if json.Unmarshal(body, &response) == nil && response.Message != "" {
		if len(response.Message) > 120 { return "request rejected" }
		return strings.Map(func(r rune) rune {
			if r < 32 || r == 127 { return -1 }
			return r
		}, response.Message)
	}
	return "request rejected"
}

func sendPushSMTP(channel *model.PushChannel, title, body string) error {
	if channel == nil { return errors.New("missing SMTP configuration") }
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	addr := net.JoinHostPort(channel.SMTPHost, strconv.Itoa(channel.SMTPPort))
	conn, err := pushDialContext(ctx, "tcp", addr)
	if err != nil { return err }
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil { return err }
	tlsOptions := &tls.Config{ServerName: channel.SMTPHost, MinVersion: tls.VersionTLS12}
	if channel.SMTPSecurity == "tls" {
		secure := tls.Client(conn, tlsOptions)
		if err = secure.HandshakeContext(ctx); err != nil { return err }
		conn = secure
	}
	client, err := smtp.NewClient(conn, channel.SMTPHost)
	if err != nil { return err }
	defer client.Close()
	if channel.SMTPSecurity == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok { return errors.New("SMTP server does not support STARTTLS") }
		if err := client.StartTLS(tlsOptions); err != nil { return err }
	}
	if channel.SMTPUsername != "" {
		secret := ""
		if channel.SMTPPasswordCipher != "" {
			secret, err = decryptPushSecret(channel.SMTPPasswordCipher)
			if err != nil { return err }
		}
		if err := client.Auth(smtp.PlainAuth("", channel.SMTPUsername, secret, channel.SMTPHost)); err != nil { return err }
	}
	if err := client.Mail(channel.SMTPFrom); err != nil { return err }
	if err := client.Rcpt(channel.SMTPTo); err != nil { return err }
	writer, err := client.Data()
	if err != nil { return err }
	sender := (&mail.Address{Address: channel.SMTPFrom}).String()
	recipient := (&mail.Address{Address: channel.SMTPTo}).String()
	mailBody := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s",
		sender, recipient, mime.QEncoding.Encode("utf-8", title), time.Now().Format(time.RFC1123Z),
		strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	if _, err := io.WriteString(writer, mailBody); err != nil { writer.Close(); return err }
	if err := writer.Close(); err != nil { return err }
	return client.Quit()
}

func sendPushDelivery(channel *model.PushChannel, delivery *model.PushDelivery) error {
	if channel == nil || !channel.Enabled { return errors.New("推送渠道已关闭") }
	if channel.Kind == "smtp" { return sendPushSMTP(channel, delivery.Title, delivery.Body) }
	if channel.Kind == "messagepusher" || channel.Kind == "serverchan" {
		return sendPushHTTP(channel, delivery.Title, delivery.Body)
	}
	return errors.New("不支持的推送渠道")
}

// Claims are conditional, so concurrent workers cannot send the same row
// simultaneously. An abandoned claim becomes eligible after 45 seconds.
func ProcessPushQueue(limit int) (int, error) {
	if DB == nil { return 0, nil }
	if limit < 1 || limit > 20 { limit = 10 }
	now := time.Now().Unix()
	var rows []model.PushDelivery
	if err := DB.Where("(status IN ? AND attempts < ? AND next_attempt_at <= ?) OR (status = ? AND locked_at < ?)",
		[]string{"pending", "retry"}, pushMaxAttempts, now, "sending", now-45).
		Order("id ASC").Limit(limit).Find(&rows).Error; err != nil { return 0, err }
	sent := 0
	for _, row := range rows {
		// Each claim starts its own lease after the preceding network request.
		now = time.Now().Unix()
		claim := DB.Model(&model.PushDelivery{}).
			Where("id = ? AND status = ? AND attempts = ? AND locked_at = ?", row.Id, row.Status, row.Attempts, row.LockedAt).
			Updates(map[string]interface{}{"status": "sending", "locked_at": now})
		if claim.Error != nil { return sent, claim.Error }
		if claim.RowsAffected != 1 { continue }
		var channel model.PushChannel
		err := DB.Where("id = ?", row.ChannelId).Take(&channel).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = DB.Where("id = ?", row.Id).Delete(&model.PushDelivery{}).Error
			continue
		}
		if err != nil { return sent, err }
		if !channel.Enabled {
			_ = DB.Model(&model.PushDelivery{}).Where("id = ?", row.Id).Updates(map[string]interface{}{"status":"paused","locked_at":0}).Error
			continue
		}
		err = sendPushDelivery(&channel, &row)
		values := map[string]interface{}{"attempts": row.Attempts + 1, "locked_at": int64(0)}
		if err == nil {
			values["status"] = "sent"
			values["sent_at"] = time.Now().Unix()
			values["last_error"] = ""
			sent++
		} else {
			// Provider error messages are recorded without provider response bodies.
			message := err.Error()
			if len(message) > 250 { message = message[:250] }
			values["last_error"] = message
			if row.Attempts+1 >= pushMaxAttempts {
				values["status"] = "failed"
			} else {
				values["status"] = "retry"
				values["next_attempt_at"] = time.Now().Unix() + int64(10*(1<<row.Attempts))
			}
		}
		if err := DB.Model(&model.PushDelivery{}).Where("id = ? AND status = ?", row.Id, "sending").
			Updates(values).Error; err != nil { return sent, err }
	}
	return sent, nil
}

var pushWorkerOnce sync.Once

func StartPushWorker() {
	pushWorkerOnce.Do(func() {
		startPersonalDeliverySchedulers()
		startPushEventWorker()
	})
}
