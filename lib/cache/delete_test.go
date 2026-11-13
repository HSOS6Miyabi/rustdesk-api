package cache

import (
    "os"
    "testing"

    "github.com/go-redis/redis/v8"
)

func TestCacheDeletionAndMissingValue(t *testing.T) {
    file := NewFileCache()
    file.SetDir(t.TempDir())
    stores := map[string]Handler{
        "memory": NewMemoryCache(64 * 1024),
        "file": file,
    }
    for name, store := range stores {
        t.Run(name,func(t *testing.T) {
            if closer,ok:=store.(interface{Close()error});ok {defer closer.Close()}
            if err:=store.Set("key",struct{Value string}{Value:"cached"},5);err!=nil {t.Fatal(err)}
            var value struct{Value string}
            if err:=store.Get("key",&value);err!=nil || value.Value!="cached" {
                t.Fatalf("cache read: %+v, %v",value,err)
            }
            if err:=store.Delete("key");err!=nil {t.Fatal(err)}
            if err:=store.Get("key",&value);err!=nil || value.Value!="" {
                t.Fatalf("stale deleted entry: %+v, %v",value,err)
            }
        })
    }
}

func TestCacheFactoryWithoutRedisOptionsUsesMemory(t *testing.T) {
    store := New(TypeRedis)
    if _,ok:=store.(*MemoryCache);!ok {t.Fatalf("unconfigured factory cache: %T",store)}
    if closer,ok:=store.(interface{Close()error});ok {defer closer.Close()}
}

func TestRedisCacheMissingValueAndDelete(t *testing.T) {
    addr:=os.Getenv("REDIS_TEST_ADDR")
    if addr=="" {t.Skip("requires REDIS_TEST_ADDR")}
    store:=NewRedis(&redis.Options{Addr:addr})
    defer store.Close()
    key:="rustdesk-api:test:cache-delete"
    if err:=store.Set(key,"stored",30);err!=nil {t.Fatal(err)}
    var value string
    if err:=store.Get(key,&value);err!=nil || value!="stored" {
        t.Fatalf("redis read: %q, %v",value,err)
    }
    if err:=store.Delete(key);err!=nil {t.Fatal(err)}
    if err:=store.Get(key,&value);err!=nil || value!="" {
        t.Fatalf("redis stale value: %q, %v",value,err)
    }
}
