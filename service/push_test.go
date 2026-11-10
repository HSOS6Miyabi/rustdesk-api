package service

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func setupPushTestDB(t *testing.T) func() {
	t.Helper()
	cleanup := setupGroupDB(t)
	if err := DB.AutoMigrate(&model.PushChannel{}, &model.PushDelivery{}, &model.PushPresence{},
		&model.AuditConn{}, &model.AddressBook{}, &model.Contact{},
		&model.AdminRole{}, &model.AdminRoleAssignment{}); err != nil {
		cleanup()
		t.Fatal(err)
	}
	t.Setenv("RUSTDESK_API_PUSH_ENCRYPTION_KEY", "test-only-push-secret-32-characters")
	t.Setenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS", "")
	return cleanup
}

func TestPushChannelValidationAndUserSeparation(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	owner := &model.User{Username:"notify-owner",Status:1}
	other := &model.User{Username:"notify-other",Status:1}
	if err := DB.Create(owner).Error; err != nil { t.Fatal(err) }
	if err := DB.Create(other).Error; err != nil { t.Fatal(err) }
	created, err := SavePushChannel(owner.Id,PushChannelInput{
		Kind:"messagepusher", Name:"我的推送",Enabled:true,
		BaseURL:"https://push.example.test",
		APIKey:"mpk_example-secret-123",
		Events:[]string{"device.online","device.offline","device.online"},
	})
	if err != nil { t.Fatal(err) }
	if !created.APIKeyConfigured || len(created.Events)!=2 { t.Fatalf("invalid public channel: %+v",created) }
	var record model.PushChannel
	if err := DB.First(&record, created.Id).Error; err != nil { t.Fatal(err) }
	if strings.Contains(record.KeyCiphertext,"mpk_example-secret-123") {
		t.Fatal("stored credentials are not encrypted")
	}
	secret,err := decryptPushSecret(record.KeyCiphertext)
	if err != nil || secret!="mpk_example-secret-123" { t.Fatalf("cannot decrypt stored key: %v",err) }
	channels, err := ListPushChannels(other.Id)
	if err != nil || len(channels) != 0 { t.Fatalf("another user read channel: %+v %v",channels,err) }
	_,err=SavePushChannel(other.Id,PushChannelInput{
		Id:created.Id,Kind:"messagepusher",Name:"伪造修改",Enabled:false,
	})
	if err == nil { t.Fatal("another user modified the channel") }
	if err=DeletePushChannel(other.Id,created.Id);err==nil { t.Fatal("another user deleted the channel") }
	updated,err := SavePushChannel(owner.Id,PushChannelInput{
		Id:created.Id,Kind:"messagepusher",Name:"更新名称",Enabled:true,
		BaseURL:"https://push.example.test",Events:[]string{"device.online"},
	})
	if err != nil || !updated.APIKeyConfigured { t.Fatalf("existing credentials were not retained: %+v %v",updated,err) }
	if err:=DB.First(&record,created.Id).Error;err!=nil{t.Fatal(err)}
	if record.KeyCiphertext=="" { t.Fatal("credentials disappeared") }
	_,err=SavePushChannel(owner.Id,PushChannelInput{
		Id:created.Id,Kind:"messagepusher",Name:"无法启用",Enabled:true,
		BaseURL:"https://push.example.test",ClearAPIKey:true,Events:[]string{"device.online"},
	})
	if err==nil { t.Fatal("channel enabled without a send key") }
	_,err=SavePushChannel(owner.Id,PushChannelInput{
		Kind:"messagepusher",Name:"非法地址",Enabled:true,
		BaseURL:"http://127.0.0.1:8080",APIKey:"secret",Events:[]string{"device.online"},
	})
	if err==nil { t.Fatal("unsafe service URL was accepted") }
	_,err=SavePushChannel(owner.Id,PushChannelInput{
		Kind:"serverchan",Name:"Server酱",Enabled:true,
		APIKey:"SCT_example",Events:[]string{"audit.alarm"},
	})
	if err!=nil { t.Fatalf("serverchan default URL rejected: %v",err) }
	_,err=SavePushChannel(owner.Id,PushChannelInput{
		Kind:"smtp",Name:"邮件通知",Enabled:true,
		SMTPHost:"smtp.example.test",SMTPPort:587,SMTPSecurity:"starttls",
		SMTPUsername:"alerts@example.test",SMTPPassword:"mail-secret",
		SMTPFrom:"alerts@example.test",SMTPTo:"owner@example.test",
		Events:[]string{"connection.started"},
	})
	if err!=nil { t.Fatalf("SMTP configuration rejected: %v",err) }
}

func TestPushQueueHasPreviewDetailsAndDeduplicatesEvents(t *testing.T) {
	cleanup := setupPushTestDB(t)
	defer cleanup()
	owner := &model.User{Username:"viewer",Status:1}
	if err := DB.Create(owner).Error;err!=nil{t.Fatal(err)}
	peer := &model.Peer{Id:"123456789",Hostname:"办公室电脑",Username:"alice",Os:"Windows 11",UserId:owner.Id,
		LastOnlineIp:"192.168.1.15"}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	created,err:=SavePushChannel(owner.Id,PushChannelInput{
		Kind:"messagepusher",Name:"通知渠道",Enabled:true,APIKey:"mpk_secret",
		BaseURL:"https://push.example.test",Events:[]string{"device.online"},
	})
	if err!=nil{t.Fatal(err)}
	event:=PushEvent{
		Code:"device.online",Reference:"online-event-1",PeerRowID:peer.RowId,DeviceID:peer.Id,
		SourceIP:"192.168.1.15",When:time.Date(2026,11,10,9,30,0,0,time.FixedZone("CST",8*3600)),
	}
	if err:=QueuePushForUsers(event,[]uint{owner.Id,owner.Id});err!=nil{t.Fatal(err)}
	if err:=QueuePushForUsers(event,[]uint{owner.Id});err!=nil{t.Fatal(err)}
	var rows []model.PushDelivery
	if err:=DB.Where("channel_id = ?",created.Id).Find(&rows).Error;err!=nil{t.Fatal(err)}
	if len(rows)!=1 {t.Fatalf("duplicate deliveries inserted: %d",len(rows))}
	if !strings.Contains(rows[0].Title,"办公室电脑")||!strings.Contains(rows[0].Title,"设备上线")||
		!strings.Contains(rows[0].Title,"192.168.1.15（局域网地址）") {
		t.Fatalf("notification title omits preview details: %s",rows[0].Title)
	}
	for _,want:=range []string{"192.168.1.15（局域网地址）", "2026-11-10", "设备主机名", "设备登录用户", "操作系统"} {
		if !strings.Contains(rows[0].Body,want) { t.Errorf("body missing %s: %s",want,rows[0].Body) }
	}
	if err:=DB.Model(&model.PushChannel{}).Where("id = ?",created.Id).Update("enabled",false).Error;err!=nil{t.Fatal(err)}
	if err:=QueuePushForUsers(PushEvent{Code:"device.online",Reference:"online-event-2",DeviceID:peer.Id},[]uint{owner.Id});err!=nil{t.Fatal(err)}
	var total int64
	DB.Model(&model.PushDelivery{}).Count(&total)
	if total!=1 {t.Fatalf("disabled channel still received deliveries: %d",total)}
}

func TestDevicePresenceTransitionsAndOfflineNotifications(t *testing.T) {
	cleanup:=setupPushTestDB(t)
	defer cleanup()
	user:=&model.User{Username:"presence-owner",Status:1}
	if err:=DB.Create(user).Error;err!=nil{t.Fatal(err)}
	peer:=&model.Peer{Id:"presence-id",Hostname:"宿舍电脑",UserId:user.Id}
	if err:=DB.Create(peer).Error;err!=nil{t.Fatal(err)}
	_,err:=SavePushChannel(user.Id,PushChannelInput{
		Name:"设备状态",Kind:"messagepusher",Enabled:true,
		APIKey:"mpk_key",BaseURL:"https://push.example.test",
		Events:[]string{"device.online","device.offline"},
	})
	if err!=nil{t.Fatal(err)}
	event:=PushEvent{Code:"device.heartbeat",DeviceID:peer.Id,PeerRowID:peer.RowId,SourceIP:"10.0.0.5",When:time.Now()}
	if err:=processPushEvent(event);err!=nil{t.Fatal(err)}
	if err:=processPushEvent(event);err!=nil{t.Fatal(err)}
	var count int64
	if err:=DB.Model(&model.PushDelivery{}).Count(&count).Error;err!=nil||count!=1{
		t.Fatalf("heartbeats generated unexpected notifications: %d %v",count,err)
	}
	if err:=DB.Model(&model.PushPresence{}).Where("peer_row_id = ?",peer.RowId).
		Update("last_seen_at",time.Now().Add(-4*time.Minute).Unix()).Error;err!=nil{t.Fatal(err)}
	if err:=ProcessPushPresenceTimeouts(10);err!=nil{t.Fatal(err)}
	if err:=ProcessPushPresenceTimeouts(10);err!=nil{t.Fatal(err)}
	if err:=DB.Model(&model.PushDelivery{}).Count(&count).Error;err!=nil||count!=2 {
		t.Fatalf("offline transition was missed or repeated: %d %v",count,err)
	}
}

func TestPushDestinationRefusesLocalNetworkByDefault(t *testing.T) {
	t.Setenv("RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS","")
	ctx,cancel:=context.WithTimeout(context.Background(),time.Second)
	defer cancel()
	conn,err:=pushDialContext(ctx,"tcp",net.JoinHostPort("127.0.0.1","44444"))
	if conn!=nil {conn.Close();t.Fatal("private address reached")}
	if err==nil{t.Fatal("private address was not rejected")}
	if _,err:=validPushURL("https://example.com:99999");err==nil {t.Fatal("invalid port accepted")}
}

func TestGeoIPDescriptionAndBatch(t *testing.T) {
	locations:=LookupGeoIPBatch([]string{"127.0.0.1","192.168.0.1","not-an-ip"})
	if locations["127.0.0.1"]!="本地回环地址" { t.Fatal(locations) }
	if locations["192.168.0.1"]!="局域网地址" { t.Fatal(locations) }
	if _,found:=locations["not-an-ip"];found { t.Fatal("invalid IP was returned") }
	if got:=GeoIPDescription("::1");got!="本地回环地址" { t.Fatalf("invalid IPv6 location: %s",got) }
	if got:=GeoIPDescription("not-an-ip");got!="无效 IP 地址" { t.Fatalf("invalid IP marker: %s",got) }
	}
