package cache

import (
	"context"
	"github.com/go-redis/redis/v8"
	"time"
)

type RedisCache struct {
	rdb *redis.Client
	ownsClient bool
}

func RedisCacheInit(conf *redis.Options) *RedisCache {
	c := &RedisCache{ownsClient: true}
	c.rdb = redis.NewClient(conf)
	return c
}

func (c *RedisCache) Get(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    data, err := c.rdb.Get(ctx, key).Result()
    if err == redis.Nil { return zeroCacheValue(value) }
    if err != nil { return err }
    return DecodeValue(data, value)
}

func (c *RedisCache) Set(key string, value interface{}, exp int) error {
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}
	if exp <= 0 {
		exp = MaxTimeOut
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    return c.rdb.Set(ctx, key, str, time.Duration(exp)*time.Second).Err()
}

func (c *RedisCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    return c.rdb.Del(ctx, key).Err()
}

func (c *RedisCache) Gc() error {
	return nil
}

func NewRedis(conf *redis.Options) *RedisCache {
	cache := RedisCacheInit(conf)
	return cache
}

// WrapRedis reuses a shared client without taking ownership of its connections.
func WrapRedis(client *redis.Client) *RedisCache {
	return &RedisCache{rdb: client}
}

func (c *RedisCache) Client() *redis.Client {
	if c == nil { return nil }
	return c.rdb
}

// Close releases connections owned by this cache instance.
func (c *RedisCache) Close() error {
    if c==nil || c.rdb==nil || !c.ownsClient { return nil }
    return c.rdb.Close()
}
