package service

import (
    "context"
    "encoding/json"
    "sort"
    "strings"
    "sync"
    "sync/atomic"
    "time"

    "github.com/google/uuid"
    "github.com/lejianwen/rustdesk-api/v2/global"
)

const streamRedisEvents = "rustdesk-api:sse:v1:events"
const streamRedisChanges = "rustdesk-api:sse:v1:changes"
const redisCacheChangeChannel = "rustdesk-api:cache:v1:invalidation"
const streamRedisKeyPrefix = "rustdesk-api:sse:v1:instance:"
const streamRedisLease = 75 * time.Second

var streamRedisNode = uuid.NewString()
var streamRedisReady atomic.Bool
var streamInterestWake = make(chan struct{}, 1)
var remoteStreamInterest = struct {
    sync.RWMutex
    events map[string]struct{}
}{events: make(map[string]struct{})}

type remoteStreamMessage struct {
    Origin string `json:"origin"`
    UserIDs []uint `json:"user_ids"`
    Type string `json:"type"`
    Data json.RawMessage `json:"data"`
}

func signalStreamInterest() {
    if !streamRedisReady.Load() { return }
    select { case streamInterestWake <- struct{}{}: default: }
}

func HasRemotePushSubscribers(eventType string) bool {
    remoteStreamInterest.RLock()
    _, specific := remoteStreamInterest.events[eventType]
    _, wildcard := remoteStreamInterest.events["*"]
    remoteStreamInterest.RUnlock()
    return specific || wildcard
}

// A single Redis subscription per API process forwards events to subscribers
// connected to other instances. Each instance advertises current interests in
// an expiring key; stale entries disappear after an unclean shutdown.
func startPushStreamRedisBridge() {
    if global.Redis == nil { return }
    launchServiceWorker(func(ctx context.Context) {
        for ctx.Err() == nil {
            if err := runPushStreamRedisBridge(ctx); err != nil && ctx.Err() == nil && Logger != nil {
                Logger.WithError(err).Warn("distributed event stream unavailable")
            }
            streamRedisReady.Store(false)
            remoteStreamInterest.Lock()
            remoteStreamInterest.events = make(map[string]struct{})
            remoteStreamInterest.Unlock()
            if !waitForWorkContext(ctx, nil, 5*time.Second, true) { return }
        }
    })
}

func runPushStreamRedisBridge(ctx context.Context) error {
    subscription := global.Redis.Subscribe(ctx, streamRedisEvents, streamRedisChanges, redisCacheChangeChannel)
    defer subscription.Close()
    if _, err := subscription.Receive(ctx); err != nil { return err }
    streamRedisReady.Store(true)
    defer streamRedisReady.Store(false)
    defer func() {
        cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        _ = global.Redis.Del(cleanup, streamRedisKeyPrefix+streamRedisNode).Err()
        _ = global.Redis.Publish(cleanup, streamRedisChanges, streamRedisNode).Err()
    }()
    lastAdvertised := "\x00"
    if err := writeLocalStreamInterests(ctx, &lastAdvertised); err != nil { return err }
    if err := loadRemoteStreamInterests(ctx); err != nil { return err }
    incoming := subscription.Channel()
    renew := time.NewTicker(30 * time.Second)
    defer renew.Stop()
    for {
        select {
        case <-ctx.Done(): return nil
        case <-streamInterestWake:
            if err := writeLocalStreamInterests(ctx, &lastAdvertised); err != nil { return err }
            if err := loadRemoteStreamInterests(ctx); err != nil { return err }
        case <-renew.C:
            if err := writeLocalStreamInterests(ctx, &lastAdvertised); err != nil { return err }
            if err := loadRemoteStreamInterests(ctx); err != nil { return err }
        case msg, ok := <-incoming:
            if !ok { return context.Canceled }
            if msg.Channel == streamRedisChanges {
                if err := loadRemoteStreamInterests(ctx); err != nil { return err }
                continue
            }
            if msg.Channel == redisCacheChangeChannel {
                applyRemoteCacheInvalidation(msg.Payload)
                continue
            }
            var event remoteStreamMessage
            if json.Unmarshal([]byte(msg.Payload), &event) != nil ||
                event.Origin == streamRedisNode || len(event.UserIDs) == 0 ||
                len(event.UserIDs) > 16 || len(event.Data) > 16384 {
                continue
            }
            PublishLivePushForUsers(event.UserIDs, event.Type, event.Data)
        }
    }
}

func writeLocalStreamInterests(ctx context.Context, lastAdvertised *string) error {
    activeStreams.RLock()
    names := make([]string, 0, len(activeStreams.interested))
    for name, count := range activeStreams.interested {
        if count > 0 { names = append(names, name) }
    }
    activeStreams.RUnlock()
    sort.Strings(names)
    signature := strings.Join(names, ",")
    changed := *lastAdvertised != signature
    key := streamRedisKeyPrefix + streamRedisNode
    if len(names) == 0 {
        if changed {
            if err := global.Redis.Del(ctx, key).Err(); err != nil { return err }
        }
    } else {
        payload, _ := json.Marshal(names)
        if err := global.Redis.Set(ctx, key, payload, streamRedisLease).Err(); err != nil { return err }
    }
    if changed {
        if err := global.Redis.Publish(ctx, streamRedisChanges, streamRedisNode).Err(); err != nil { return err }
    }
    *lastAdvertised = signature
    return nil
}

func loadRemoteStreamInterests(ctx context.Context) error {
    events := make(map[string]struct{})
    cursor := uint64(0)
    scanned := 0
    for {
        keys, next, err := global.Redis.Scan(ctx, cursor, streamRedisKeyPrefix+"*", 100).Result()
        if err != nil { return err }
        for _, key := range keys {
            if key == streamRedisKeyPrefix+streamRedisNode { continue }
            raw, err := global.Redis.Get(ctx, key).Bytes()
            if err != nil { continue }
            var names []string
            if json.Unmarshal(raw, &names) != nil { continue }
            for _, name := range names {
                if name == "*" || pushEventName(name) != "" { events[name] = struct{}{} }
            }
        }
        scanned += len(keys)
        if next == 0 || scanned >= 1000 { break }
        cursor = next
    }
    remoteStreamInterest.Lock()
    remoteStreamInterest.events = events
    remoteStreamInterest.Unlock()
    return nil
}

func publishRemotePushEvent(users []uint, code string, data []byte) {
    if !streamRedisReady.Load() || !HasRemotePushSubscribers(code) { return }
    payload, err := json.Marshal(remoteStreamMessage{
        Origin: streamRedisNode, UserIDs: users, Type: code, Data: data,
    })
    if err != nil { return }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    if err := global.Redis.Publish(ctx, streamRedisEvents, payload).Err(); err != nil && Logger != nil {
        Logger.WithError(err).Warn("cannot relay live event to other API instances")
    }
}

const cacheInvalidationPolicy = "policy"
const cacheInvalidationPush = "push"
const cacheInvalidationNotification = "notification"

// publishCacheInvalidation forwards configuration changes once, after the local
// database update succeeds. Heartbeats never make Redis requests for caching.
func publishCacheInvalidation(kind string) {
    if !streamRedisReady.Load() || global.Redis == nil { return }
    ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
    defer cancel()
    if err := global.Redis.Publish(ctx,redisCacheChangeChannel,
        streamRedisNode+":"+kind).Err(); err != nil && Logger != nil {
        Logger.WithError(err).Warn("cannot propagate cache invalidation")
    }
}

func applyRemoteCacheInvalidation(payload string) {
    separator := strings.IndexByte(payload, ':')
    if separator < 0 || separator == len(payload)-1 { return }
    origin, kind := payload[:separator], payload[separator+1:]
    if origin == "" || origin == streamRedisNode { return }
    switch kind {
    case cacheInvalidationPolicy:
        invalidatePolicyActivityLocal()
    case cacheInvalidationPush:
        invalidatePushActivityLocal()
    case cacheInvalidationNotification:
        invalidateAdministrativeNotificationActivityLocal()
    }
}
