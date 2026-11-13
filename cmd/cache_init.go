package main

import (
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

const defaultMemoryCacheBytes = 32 * 1024 * 1024

func redisConnectionOptions(addr, password string, db int) *redis.Options {
    return &redis.Options{
        Addr: addr, Password: password, DB: db,
        DialTimeout: 2 * time.Second,
        ReadTimeout: 2 * time.Second,
        WriteTimeout: 2 * time.Second,
    }
}

// initOptionalCache enables Redis when an address is configured and otherwise
// uses a bounded, expiring in-memory cache. Both Redis settings are supported.
func initOptionalCache() error {
    cfg := global.Config
    cacheType := strings.ToLower(strings.TrimSpace(cfg.Cache.Type))
    cacheAddr := strings.TrimSpace(cfg.Cache.RedisAddr)
    sharedAddr := strings.TrimSpace(cfg.Redis.Addr)
    sharedPassword, sharedDB := cfg.Redis.Password, cfg.Redis.Db

    if sharedAddr == "" && (cacheType == "" || cacheType == cache.TypeRedis) && cacheAddr != "" {
        sharedAddr = cacheAddr
        sharedPassword, sharedDB = cfg.Cache.RedisPwd, cfg.Cache.RedisDb
    }
    global.Redis = nil
    if sharedAddr != "" {
        global.Redis = redis.NewClient(redisConnectionOptions(sharedAddr, sharedPassword, sharedDB))
    }

    switch cacheType {
    case "", cache.TypeMem:
        if cacheType == "" && global.Redis != nil {
            global.Cache = cache.WrapRedis(global.Redis)
        } else {
            global.Cache = cache.NewMemoryCache(defaultMemoryCacheBytes)
        }
    case cache.TypeRedis:
        if cacheAddr == "" {
            if global.Redis != nil {
                global.Cache = cache.WrapRedis(global.Redis)
            } else {
                global.Cache = cache.NewMemoryCache(defaultMemoryCacheBytes)
                if global.Logger != nil {
                    global.Logger.Warn("Redis cache selected without an address; using in-memory cache")
                }
            }
        } else if global.Redis != nil && cacheAddr == sharedAddr &&
            cfg.Cache.RedisPwd == sharedPassword && cfg.Cache.RedisDb == sharedDB {
            global.Cache = cache.WrapRedis(global.Redis)
        } else {
            global.Cache = cache.NewRedis(redisConnectionOptions(
                cacheAddr, cfg.Cache.RedisPwd, cfg.Cache.RedisDb))
        }
    case cache.TypeFile:
        directory := cfg.Cache.FileDir
        if directory == "" { directory = "./data/cache" }
        if err := os.MkdirAll(directory, 0700); err != nil {
            return fmt.Errorf("prepare file cache directory: %w", err)
        }
        fileCache := cache.NewFileCache()
        fileCache.SetDir(directory)
        global.Cache = fileCache
        global.Config.Cache.FileDir = directory
    default:
        return fmt.Errorf("unsupported cache type %q", cfg.Cache.Type)
    }
    return nil
}
