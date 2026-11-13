package main

import (
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

func configuredCacheTest(t *testing.T, cfg config.Config) {
    t.Helper()
    oldConfig, oldCache, oldRedis := global.Config, global.Cache, global.Redis
    t.Cleanup(func() {
        if closer, ok := global.Cache.(interface{ Close() error }); ok { _ = closer.Close() }
        if global.Redis != nil { _ = global.Redis.Close() }
        global.Config, global.Cache, global.Redis = oldConfig, oldCache, oldRedis
    })
    global.Config = cfg
    if err := initOptionalCache(); err != nil { t.Fatal(err) }
}

func TestNoRedisConfigurationUsesMemory(t *testing.T) {
    configuredCacheTest(t, config.Config{})
    if global.Redis != nil { t.Fatal("Redis client created without configuration") }
    if _, ok := global.Cache.(*cache.MemoryCache); !ok {
        t.Fatalf("expected memory cache, got %T", global.Cache)
    }
    if err := global.Cache.Set("optional-cache", 42, 2); err != nil { t.Fatal(err) }
    var value int
    if err := global.Cache.Get("optional-cache", &value); err != nil || value != 42 {
        t.Fatalf("memory cache read: %d %v", value, err)
    }
    if err := global.Cache.Delete("optional-cache"); err != nil { t.Fatal(err) }
    if err := global.Cache.Get("optional-cache", &value); err != nil || value != 0 {
        t.Fatalf("memory cache deletion: %d %v", value, err)
    }
}

func TestExplicitRedisCacheWithoutAddressUsesMemory(t *testing.T) {
    configuredCacheTest(t, config.Config{Cache: config.Cache{Type: cache.TypeRedis}})
    if global.Redis != nil { t.Fatal("Redis client created without an address") }
    if _, ok := global.Cache.(*cache.MemoryCache); !ok { t.Fatal("cache did not fall back to memory") }
}

func TestAvailableRedisConfigurationChoosesSharedCache(t *testing.T) {
    configuredCacheTest(t, config.Config{Redis: config.Redis{Addr: "127.0.0.1:6379"}})
    value, ok := global.Cache.(*cache.RedisCache)
    if !ok || value.Client() != global.Redis { t.Fatal("default cache did not reuse Redis") }
    if err := value.Close(); err != nil { t.Fatal(err) }
}

func TestExplicitMemoryOverridesRedis(t *testing.T) {
    configuredCacheTest(t, config.Config{
        Redis: config.Redis{Addr: "127.0.0.1:6379"},
        Cache: config.Cache{Type: cache.TypeMem},
    })
    if _, ok := global.Cache.(*cache.MemoryCache); !ok { t.Fatal("explicit memory cache was ignored") }
}

func TestCacheAddressConfiguresSharedRedis(t *testing.T) {
    configuredCacheTest(t, config.Config{Cache: config.Cache{
        Type: cache.TypeRedis, RedisAddr: "127.0.0.1:6379",
    }})
    value, ok := global.Cache.(*cache.RedisCache)
    if !ok || value.Client() != global.Redis { t.Fatal("cache Redis address was ignored") }
}
