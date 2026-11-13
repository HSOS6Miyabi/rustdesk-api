package service

import (
    "os"
    "testing"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

func withOAuthStore(t *testing.T, store cache.Handler) {
    t.Helper()
    old := global.Cache
    global.Cache = store
    t.Cleanup(func() {
        global.Cache = old
        if closer, ok := store.(interface{ Close() error }); ok { _ = closer.Close() }
    })
}

func TestOAuthMemoryCachePreservesExpiryAndDeletes(t *testing.T) {
    withOAuthStore(t, cache.NewMemoryCache(1024 * 1024))
    service := &OauthService{}
    if err := service.SetOauthCache("local-state", &OauthCacheItem{Op:"github", Nonce:"nonce", ReturnTo:"/webclient2/"}, 1); err != nil {
        t.Fatal(err)
    }
    state := service.GetOauthCache("local-state")
    if state == nil || state.Op != "github" || state.ReturnTo != "/webclient2/" { t.Fatalf("missing state: %+v", state) }
    state.UserId = 42
    if err := service.SetOauthCache("local-state", state, 0); err != nil { t.Fatal(err) }
    updated := service.GetOauthCache("local-state")
    if updated == nil || updated.UserId != 42 || updated.ReturnTo != "/webclient2/" { t.Fatalf("update missing: %+v", updated) }
    time.Sleep(1100 * time.Millisecond)
    if got := service.GetOauthCache("local-state"); got != nil { t.Fatalf("expired state returned: %+v", got) }
    if err := service.SetOauthCache("local-state", state, 0); err == nil {
        t.Fatal("expired state was renewed without a new authorization")
    }
    if err := service.SetOauthCache("new-state", &OauthCacheItem{Op:"github"}, 10); err != nil { t.Fatal(err) }
    service.DeleteOauthCache("new-state")
    if got := service.GetOauthCache("new-state"); got != nil { t.Fatal("authorization state survived deletion") }
}

func TestOAuthFileCacheSupportsDeletion(t *testing.T) {
    store := cache.NewFileCache()
    store.SetDir(t.TempDir())
    withOAuthStore(t, store)
    service := &OauthService{}
    if err := service.SetOauthCache("file-state", &OauthCacheItem{Action:OauthActionTypeLogin}, 10); err != nil {
        t.Fatal(err)
    }
    if service.GetOauthCache("file-state") == nil { t.Fatal("authorization state missing from file cache") }
    service.DeleteOauthCache("file-state")
    if service.GetOauthCache("file-state") != nil { t.Fatal("file cache failed to delete authorization state") }
}

func TestOAuthSharedRedisStateAcrossInstances(t *testing.T) {
    addr := os.Getenv("REDIS_TEST_ADDR")
    if addr == "" { t.Skip("requires REDIS_TEST_ADDR") }
    clientOne := cache.NewRedis(&redis.Options{Addr:addr})
    clientTwo := cache.NewRedis(&redis.Options{Addr:addr})
    t.Cleanup(func() { _ = clientOne.Close(); _ = clientTwo.Close() })
    stateID := "oauth-test-cross-instance-" + time.Now().Format("150405.000000000")
    service := &OauthService{}
    original := global.Cache
    t.Cleanup(func() { global.Cache = original })
    global.Cache = clientOne
    if err := service.SetOauthCache(stateID, &OauthCacheItem{Op:"google", Nonce:"abc"}, 3); err != nil {
        t.Fatal(err)
    }
    global.Cache = clientTwo
    received := service.GetOauthCache(stateID)
    if received == nil || received.Op != "google" { t.Fatalf("cross-instance state missing: %+v", received) }
    received.UserId = 91
    if err := service.SetOauthCache(stateID, received, 0); err != nil { t.Fatal(err) }
    global.Cache = clientOne
    if got := service.GetOauthCache(stateID); got == nil || got.UserId != 91 {
        t.Fatalf("cross-instance update missing: %+v", got)
    }
    service.DeleteOauthCache(stateID)
    global.Cache = clientTwo
    if got := service.GetOauthCache(stateID); got != nil { t.Fatal("Redis OAuth state survived deletion") }
}
