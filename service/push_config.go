package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"net/mail"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type PushEventOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var pushEventCatalog = []PushEventOption{
	{"device.online", "设备上线"},
	{"device.offline", "设备离线"},
	{"device.registered", "新设备登记"},
	{"connection.started", "远程连接建立"},
	{"connection.ended", "远程连接结束"},
	{"audit.file", "文件传输"},
	{"audit.alarm", "设备安全告警"},
	{"deployment.changed", "设备部署状态变更"},
	{"session.disconnect_requested", "远程连接结束请求"},
	{"session.ended", "远程会话已结束"},
	{"login.failure", "账户登录失败"},
}

func PushEventCatalog() []PushEventOption {
	return append([]PushEventOption(nil), pushEventCatalog...)
}

type PushChannelInput struct {
	Id            uint     `json:"id"`
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	Events        []string `json:"events"`
	BaseURL       string   `json:"base_url"`
	APIKey        string   `json:"api_key"`
	ClearAPIKey   bool     `json:"clear_api_key"`
	SMTPHost      string   `json:"smtp_host"`
	SMTPPort      int      `json:"smtp_port"`
	SMTPSecurity  string   `json:"smtp_security"`
	SMTPUsername  string   `json:"smtp_username"`
	SMTPPassword  string   `json:"smtp_password"`
	ClearPassword bool     `json:"clear_password"`
	SMTPFrom      string   `json:"smtp_from"`
	SMTPTo        string   `json:"smtp_to"`
}

type PublicPushChannel struct {
	Id                 uint     `json:"id"`
	Kind               string   `json:"kind"`
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	Events             []string `json:"events"`
	BaseURL            string   `json:"base_url"`
	APIKeyConfigured   bool     `json:"api_key_configured"`
	SMTPHost           string   `json:"smtp_host"`
	SMTPPort           int      `json:"smtp_port"`
	SMTPSecurity       string   `json:"smtp_security"`
	SMTPUsername       string   `json:"smtp_username"`
	PasswordConfigured bool     `json:"password_configured"`
	SMTPFrom           string   `json:"smtp_from"`
	SMTPTo             string   `json:"smtp_to"`
}

func pushKey() ([]byte, error) {
	secret := os.Getenv("RUSTDESK_API_PUSH_ENCRYPTION_KEY")
	if secret == "" { secret = os.Getenv("RUSTDESK_API_MAIL_ENCRYPTION_KEY") }
	if len(secret) < 16 { return nil, errors.New("请设置至少 16 个字符的 RUSTDESK_API_PUSH_ENCRYPTION_KEY") }
	digest := sha256.Sum256([]byte(secret))
	return digest[:], nil
}

func encryptPushSecret(plain string) (string, error) {
	key, err := pushKey()
	if err != nil { return "", err }
	block, err := aes.NewCipher(key)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil { return "", err }
	return base64.RawStdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func decryptPushSecret(ciphertext string) (string, error) {
	key, err := pushKey()
	if err != nil { return "", err }
	payload, err := base64.RawStdEncoding.DecodeString(ciphertext)
	if err != nil { return "", err }
	block, err := aes.NewCipher(key)
	if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block)
	if err != nil { return "", err }
	if len(payload) < gcm.NonceSize() { return "", errors.New("invalid encrypted push credentials") }
	plain, err := gcm.Open(nil, payload[:gcm.NonceSize()], payload[gcm.NonceSize():], nil)
	if err != nil { return "", err }
	return string(plain), nil
}

func pushAllowPrivate() bool {
	return os.Getenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS") == "1"
}

// A user-controlled URL is checked both when saved and when a connection is opened.
func validPushURL(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if len(value) > 512 { return "", errors.New("推送地址过长") }
	p, err := url.Parse(value)
	if err != nil || p.Hostname() == "" || p.User != nil || p.Fragment != "" ||
		p.RawQuery != "" || p.Opaque != "" || strings.ContainsAny(value, "\r\n\t") ||
		strings.Contains(p.EscapedPath(), "..") {
		return "", errors.New("推送地址无效")
	}
	if p.Scheme != "https" && !(pushAllowPrivate() && p.Scheme == "http") {
		return "", errors.New("推送地址需要使用 HTTPS")
	}
	if port := p.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 { return "", errors.New("推送端口无效") }
	}
	return value, nil
}

func normalizePushEvents(input []string) ([]string, error) {
	known := make(map[string]bool, len(pushEventCatalog))
	for _, item := range pushEventCatalog { known[item.Code] = true }
	seen := make(map[string]bool)
	result := make([]string, 0, len(input))
	for _, entry := range input {
		event := strings.TrimSpace(entry)
		if !known[event] { return nil, fmt.Errorf("不支持的事件：%s", event) }
		if !seen[event] { seen[event] = true; result = append(result, event) }
	}
	sort.Strings(result)
	return result, nil
}

func publicPushChannel(row *model.PushChannel) PublicPushChannel {
	events := make([]string, 0)
	for _, event := range strings.Split(row.Events, ",") {
		if event != "" { events = append(events, event) }
	}
	return PublicPushChannel{
		Id: row.Id, Kind: row.Kind, Name: row.Name, Enabled: row.Enabled, Events: events,
		BaseURL: row.BaseURL,
		APIKeyConfigured: row.KeyCiphertext != "", SMTPHost: row.SMTPHost,
		SMTPPort: row.SMTPPort, SMTPSecurity: row.SMTPSecurity,
		SMTPUsername: row.SMTPUsername, PasswordConfigured: row.SMTPPasswordCipher != "",
		SMTPFrom: row.SMTPFrom, SMTPTo: row.SMTPTo,
	}
}

func ListPushChannels(userID uint) ([]PublicPushChannel, error) {
	if userID == 0 { return nil, errors.New("请先登录") }
	var rows []model.PushChannel
	if err := DB.Where("user_id = ?", userID).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]PublicPushChannel, 0, len(rows))
	for i := range rows { out = append(out, publicPushChannel(&rows[i])) }
	return out, nil
}

func validEmail(value string) bool {
	if strings.ContainsAny(value, "\r\n") { return false }
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func SavePushChannel(userID uint, input PushChannelInput) (PublicPushChannel, error) {
	if userID == 0 { return PublicPushChannel{}, errors.New("请先登录") }
	if input.Kind != "messagepusher" && input.Kind != "smtp" && input.Kind != "serverchan" {
		return PublicPushChannel{}, errors.New("不支持的推送服务类型")
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 100 || strings.ContainsAny(input.Name, "\r\n") {
		return PublicPushChannel{}, errors.New("请填写有效的渠道名称")
	}
	events, err := normalizePushEvents(input.Events)
	if err != nil { return PublicPushChannel{}, err }
	var existing model.PushChannel
	if input.Id != 0 {
		if err = DB.Where("id = ? AND user_id = ?", input.Id, userID).Take(&existing).Error; err != nil {
			return PublicPushChannel{}, errors.New("推送渠道不存在")
		}
		if existing.Kind != input.Kind { return PublicPushChannel{}, errors.New("无法更改渠道类型") }
	}
	record := model.PushChannel{
		IdModel: model.IdModel{Id: input.Id}, UserId: userID, Kind: input.Kind,
		Name: input.Name, Enabled: input.Enabled, Events: strings.Join(events, ","),
	}
	if len(input.APIKey) > 512 || len(input.SMTPPassword) > 1024 ||
		strings.ContainsAny(input.APIKey + input.SMTPPassword, "\r\n") {
		return PublicPushChannel{}, errors.New("推送服务凭据无效")
	}
	if input.Kind == "smtp" {
		record.SMTPHost = strings.TrimSpace(input.SMTPHost)
		record.SMTPPort = input.SMTPPort
		if record.SMTPPort == 0 { record.SMTPPort = 587 }
		record.SMTPSecurity = strings.ToLower(strings.TrimSpace(input.SMTPSecurity))
		if record.SMTPSecurity == "" { record.SMTPSecurity = "starttls" }
		record.SMTPUsername = strings.TrimSpace(input.SMTPUsername)
		record.SMTPFrom = strings.TrimSpace(input.SMTPFrom)
		record.SMTPTo = strings.TrimSpace(input.SMTPTo)
		if record.SMTPPort < 1 || record.SMTPPort > 65535 || len(record.SMTPHost) > 253 ||
			strings.ContainsAny(record.SMTPHost+record.SMTPUsername, "\r\n\t /\\") ||
			(record.SMTPSecurity != "tls" && record.SMTPSecurity != "starttls" &&
				!(record.SMTPSecurity == "none" && pushAllowPrivate())) {
			return PublicPushChannel{}, errors.New("SMTP 配置无效，请使用 TLS 或 STARTTLS")
		}
		if record.Enabled && (record.SMTPHost == "" || !validEmail(record.SMTPFrom) || !validEmail(record.SMTPTo)) {
			return PublicPushChannel{}, errors.New("启用 SMTP 需要填写服务器、发件地址和收件地址")
		}
		if input.ClearPassword { record.SMTPPasswordCipher = "" } else { record.SMTPPasswordCipher = existing.SMTPPasswordCipher }
		if input.SMTPPassword != "" {
			record.SMTPPasswordCipher, err = encryptPushSecret(input.SMTPPassword)
			if err != nil { return PublicPushChannel{}, err }
		}
	} else {
		record.BaseURL = strings.TrimSpace(input.BaseURL)
		if record.BaseURL == "" && input.Kind == "serverchan" { record.BaseURL = "https://sctapi.ftqq.com" }
		if record.BaseURL != "" {
			record.BaseURL, err = validPushURL(record.BaseURL)
			if err != nil { return PublicPushChannel{}, err }
		}
		if input.ClearAPIKey { record.KeyCiphertext = "" } else { record.KeyCiphertext = existing.KeyCiphertext }
		if input.APIKey != "" {
			key := strings.TrimSpace(input.APIKey)
			if key == "" || strings.ContainsAny(key, "/?# \t") {
				return PublicPushChannel{}, errors.New("发送密钥不能包含空格或 URL 特殊字符")
			}
			record.KeyCiphertext, err = encryptPushSecret(key)
			if err != nil { return PublicPushChannel{}, err }
		}
		if record.Enabled && (record.BaseURL == "" || record.KeyCiphertext == "") {
			return PublicPushChannel{}, errors.New("启用推送需要填写服务地址及发送密钥")
		}
	}
	if record.Enabled && len(events) == 0 { return PublicPushChannel{}, errors.New("请选择至少一个订阅事件") }
	if input.Id == 0 {
		err = DB.Create(&record).Error
	} else {
		err = DB.Model(&model.PushChannel{}).Where("id = ? AND user_id = ?", input.Id, userID).
			Updates(map[string]interface{}{
				"name":record.Name, "enabled":record.Enabled, "events":record.Events,
				"base_url":record.BaseURL,
				"key_ciphertext":record.KeyCiphertext, "smtp_host":record.SMTPHost,
				"smtp_port":record.SMTPPort, "smtp_security":record.SMTPSecurity,
				"smtp_username":record.SMTPUsername, "smtp_password_cipher":record.SMTPPasswordCipher,
				"smtp_from":record.SMTPFrom, "smtp_to":record.SMTPTo,
			}).Error
	}
	if err != nil { return PublicPushChannel{}, err }
	// Pause queued messages on a disabled destination and resume when enabled.
	if input.Id != 0 {
		if record.Enabled {
			err = DB.Model(&model.PushDelivery{}).Where("channel_id = ? AND status = ?", input.Id, "paused").Update("status", "pending").Error
		} else {
			err = DB.Model(&model.PushDelivery{}).Where("channel_id = ? AND status IN ?", input.Id, []string{"pending","retry"}).Update("status", "paused").Error
		}
	}
	if err == nil && record.Enabled { wakePushDelivery() }
	return publicPushChannel(&record), err
}

func DeletePushChannel(userID, channelID uint) error {
	if userID == 0 || channelID == 0 { return errors.New("无效的推送渠道") }
	return DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND user_id = ?", channelID, userID).Delete(&model.PushChannel{})
		if result.Error != nil { return result.Error }
		if result.RowsAffected == 0 { return gorm.ErrRecordNotFound }
		return tx.Where("channel_id = ?", channelID).Delete(&model.PushDelivery{}).Error
	})
}

func ListPushDeliveries(userID uint, page, size uint) ([]model.PushDelivery, int64, error) {
	if userID == 0 { return nil, 0, errors.New("请先登录") }
	if page == 0 { page = 1 }
	if size == 0 { size = 20 }
	if size > 100 { size = 100 }
	query := DB.Model(&model.PushDelivery{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil { return nil, 0, err }
	var rows []model.PushDelivery
	err := query.Order("id DESC").Scopes(Paginate(page, size)).Find(&rows).Error
	return rows, total, err
}

func pushSubscribed(channel *model.PushChannel, event string) bool {
	if channel == nil || !channel.Enabled { return false }
	for _, name := range strings.Split(channel.Events, ",") {
		if name == event { return true }
	}
	return false
}

func RetryPushDelivery(userID, deliveryID uint) error {
	if userID == 0 || deliveryID == 0 { return errors.New("无效的发送记录") }
	res := DB.Model(&model.PushDelivery{}).Where("id = ? AND user_id = ? AND status = ?", deliveryID, userID, "failed").
		Updates(map[string]interface{}{"status":"pending","attempts":0,"next_attempt_at":0,"last_error":""})
	if res.Error != nil { return res.Error }
	if res.RowsAffected != 1 { return errors.New("该消息目前无法重新发送") }
	wakePushDelivery()
	return nil
}

