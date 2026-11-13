package utils

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "log"
    "sync/atomic"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/google/uuid"
)

const authRedisPrefix = "rustdesk-api:auth:v1:"
const authRedisTimeout = 750 * time.Millisecond

var lastAuthRedisWarning atomic.Int64

func redisAuthContext() (context.Context, context.CancelFunc) {
    return context.WithTimeout(context.Background(), authRedisTimeout)
}

func logRedisAuthError(err error) {
    if err == nil { return }
    now := time.Now().Unix()
    last := lastAuthRedisWarning.Load()
    if now-last < 60 || !lastAuthRedisWarning.CompareAndSwap(last, now) { return }
    log.Printf("Redis login security state unavailable; using local state: %v", err)
}

func authRedisIPKeys(ip string) (attempts, ban string) {
    digest := sha256.Sum256([]byte(ip))
    key := hex.EncodeToString(digest[:])
    return authRedisPrefix + "attempts:" + key, authRedisPrefix + "ban:" + key
}

func authRedisCaptchaKey(id string) string {
    digest := sha256.Sum256([]byte(id))
    return authRedisPrefix + "captcha:" + hex.EncodeToString(digest[:])
}

var redisFailedAttemptScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return -1 end
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now-window)
redis.call('ZADD', KEYS[1], now, ARGV[5])
redis.call('PEXPIRE', KEYS[1], window)
local count = redis.call('ZCARD', KEYS[1])
local cap = tonumber(ARGV[6])
if count > cap then
    redis.call('ZREMRANGEBYRANK', KEYS[1], 0, count-cap-1)
    count = cap
end
if tonumber(ARGV[3]) > 0 and count >= tonumber(ARGV[3]) then
    redis.call('SET', KEYS[2], '1', 'PX', ARGV[4])
    redis.call('DEL', KEYS[1])
    return -1
end
return count
`)

var redisSecurityStatusScript = redis.NewScript(`
if redis.call('EXISTS', KEYS[2]) == 1 then return {1, 0} end
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
local count = redis.call('ZCARD', KEYS[1])
return {0, count}
`)

var redisVerifyCaptchaScript = redis.NewScript(`
local saved = redis.call('GET', KEYS[1])
if not saved or saved ~= ARGV[1] then return 0 end
redis.call('DEL', KEYS[1])
return 1
`)

func (ll *LoginLimiter) redisRecordFailedAttempt(ip string) {
    if ll == nil || ll.redis == nil { return }
    attempts, ban := authRedisIPKeys(ip)
    window := ll.policy.AttemptsWindow.Milliseconds()
    if window < 1 { window = 1 }
    duration := ll.policy.BanDuration.Milliseconds()
    if duration < 1 { duration = 1 }
    cap := ll.policy.BanThreshold
    if ll.policy.CaptchaThreshold > cap { cap = ll.policy.CaptchaThreshold }
    if cap < 32 { cap = 32 }
    ctx, cancel := redisAuthContext()
    defer cancel()
    err := redisFailedAttemptScript.Run(ctx, ll.redis, []string{attempts,ban},
        time.Now().UnixMilli(), window, ll.policy.BanThreshold,
        duration, uuid.NewString(), cap).Err()
    logRedisAuthError(err)
}

func (ll *LoginLimiter) redisSecurityStatus(ip string) (banned, captcha bool, ok bool) {
    if ll == nil || ll.redis == nil { return false, false, false }
    attempts, ban := authRedisIPKeys(ip)
    ctx, cancel := redisAuthContext()
    defer cancel()
    result, err := redisSecurityStatusScript.Run(ctx, ll.redis,
        []string{attempts,ban},
        time.Now().Add(-ll.policy.AttemptsWindow).UnixMilli()).Result()
    if err != nil { logRedisAuthError(err); return false,false,false }
    values, valid := result.([]interface{})
    if !valid || len(values) != 2 { return false,false,false }
    flag, okFlag := values[0].(int64)
    count, okCount := values[1].(int64)
    if !okFlag || !okCount { return false,false,false }
    if flag != 0 { return true,false,true }
    return false, count >= int64(ll.policy.CaptchaThreshold), true
}

func (ll *LoginLimiter) redisIsIPBanned(ip string) (bool, bool) {
    if ll == nil || ll.redis == nil { return false,false }
    _, ban := authRedisIPKeys(ip)
    ctx, cancel := redisAuthContext()
    defer cancel()
    count, err := ll.redis.Exists(ctx,ban).Result()
    if err != nil { logRedisAuthError(err); return false,false }
    return count > 0,true
}

func (ll *LoginLimiter) redisRemoveAttempts(ip string) {
    if ll == nil || ll.redis == nil { return }
    attempts,_ := authRedisIPKeys(ip)
    ctx,cancel := redisAuthContext()
    defer cancel()
    logRedisAuthError(ll.redis.Del(ctx,attempts).Err())
}

func (ll *LoginLimiter) redisStoreCaptcha(meta CaptchaMeta) {
    if ll == nil || ll.redis == nil { return }
    ttl := time.Until(meta.ExpiresAt)
    if ttl <= 0 { return }
    digest := sha256.Sum256([]byte(meta.Answer))
    ctx,cancel := redisAuthContext()
    defer cancel()
    if _, err := ll.redis.SetNX(ctx,authRedisCaptchaKey(meta.Id),
        hex.EncodeToString(digest[:]),ttl).Result(); err != nil {
        logRedisAuthError(err)
    }
}

// The Redis script checks and consumes a CAPTCHA in one operation. Missing
// challenges stay invalid even when a local copy exists on this instance.
func (ll *LoginLimiter) redisVerifyCaptcha(id, answer string) (bool, bool) {
    if ll == nil || ll.redis == nil { return false,false }
    if len(id) == 0 || len(id) > 256 { return false,true }
    digest := sha256.Sum256([]byte(answer))
    ctx,cancel := redisAuthContext()
    defer cancel()
    result, err := redisVerifyCaptchaScript.Run(ctx,ll.redis,
        []string{authRedisCaptchaKey(id)},
        hex.EncodeToString(digest[:])).Int64()
    if err != nil { logRedisAuthError(err); return false,false }
    return result == 1,true
}
