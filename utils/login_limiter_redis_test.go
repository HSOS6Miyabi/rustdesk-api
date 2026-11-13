package utils

import (
    "os"
    "sync"
    "testing"
    "time"

    "github.com/go-redis/redis/v8"
)

func TestRedisLoginLimiterSharesAttemptsAndCaptcha(t *testing.T) {
    addr := os.Getenv("REDIS_TEST_ADDR")
    if addr == "" { t.Skip("requires REDIS_TEST_ADDR") }
    clientA := redis.NewClient(&redis.Options{Addr:addr})
    clientB := redis.NewClient(&redis.Options{Addr:addr})
    defer clientA.Close()
    defer clientB.Close()
    policy := SecurityPolicy{
        CaptchaThreshold: 2, BanThreshold: 6,
        AttemptsWindow: 2*time.Minute, BanDuration: 2*time.Minute,
    }
    a,b := NewLoginLimiter(policy),NewLoginLimiter(policy)
    defer a.Close()
    defer b.Close()
    a.UseRedis(clientA)
    b.UseRedis(clientB)
    a.RegisterProvider(&MockCaptchaProvider{})
    b.RegisterProvider(&MockCaptchaProvider{})
    ip := "198.51.100.25-" + time.Now().Format("150405.000000000")
    var workers sync.WaitGroup
    for i:=0;i<6;i++ {
        workers.Add(1)
        go func(index int) {
            defer workers.Done()
            if index%2==0 { a.RecordFailedAttempt(ip) } else { b.RecordFailedAttempt(ip) }
        }(i)
    }
    workers.Wait()
    if !a.IsIPBanned(ip) || !b.IsIPBanned(ip) { t.Fatal("distributed login ban was not shared") }
    captchaErr, challenge := a.RequireCaptcha()
    if captchaErr != nil { t.Fatal(captchaErr) }
    if !b.VerifyCaptcha(challenge.Id, challenge.Answer) { t.Fatal("CAPTCHA not shared") }
    if a.VerifyCaptcha(challenge.Id, challenge.Answer) { t.Fatal("CAPTCHA could be reused") }
    a.RemoveAttempts(ip)
    b.RemoveAttempts(ip)
}

func TestRedisLoginLimiterFallsBackWhenDisconnected(t *testing.T) {
    ll := NewLoginLimiter(SecurityPolicy{CaptchaThreshold:2,AttemptsWindow:time.Minute})
    defer ll.Close()
    client:=redis.NewClient(&redis.Options{Addr:"127.0.0.1:1",
        DialTimeout:10*time.Millisecond,ReadTimeout:10*time.Millisecond,
        WriteTimeout:10*time.Millisecond,MaxRetries:0})
    defer client.Close()
    ll.UseRedis(client)
    ll.RegisterProvider(&MockCaptchaProvider{})
    ip:="192.0.2.45"
    ll.RecordFailedAttempt(ip)
    ll.RecordFailedAttempt(ip)
    if _, required := ll.CheckSecurityStatus(ip); !required {
        t.Fatal("local login protection was not used during Redis outage")
    }
    err, challenge := ll.RequireCaptcha()
    if err != nil || !ll.VerifyCaptcha(challenge.Id,challenge.Answer) {
        t.Fatalf("local CAPTCHA fallback unavailable: %v",err)
    }
}
