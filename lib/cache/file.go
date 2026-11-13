package cache

import (
	"crypto/md5"
	"fmt"
	"os"
	"sync"
	"time"
)

type FileCache struct {
	locks [64]sync.Mutex
	Dir   string
}

func (fc *FileCache) getLock(key string) *sync.Mutex {
	var hash uint32
	for i := len(key) - 2; i < len(key); i++ {
		if i >= 0 {
			hash = hash*33 + uint32(key[i])
		}
	}
	return &fc.locks[hash%uint32(len(fc.locks))]
}

func (c *FileCache) Get(key string, value interface{}) error {
	data, err := c.getValue(key)
    if err != nil { return err }
    if data == "" { return zeroCacheValue(value) }
	return DecodeValue(data, value)
}

// 获取值,如果文件不存在或者过期，返回空，过滤掉错误
func (c *FileCache) getValue(key string) (string, error) {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()
	fileInfo, err := os.Stat(f)
	if err != nil {
		//文件不存在
		return "", nil
	}
	difT := time.Now().Sub(fileInfo.ModTime())
	if difT >= 0 {
		os.Remove(f)
		return "", nil
	}
	data, err := os.ReadFile(f)
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

// 保存值
func (c *FileCache) saveValue(key string, value string, exp int) error {
	f := c.fileName(key)
	lock := c.getLock(f)
	lock.Lock()
	defer lock.Unlock()

	err := os.MkdirAll(c.Dir,0700)
    if err != nil { return err }
    err = os.WriteFile(f,([]byte)(value),0600)
	if err != nil {
		return err
	}
	if exp <= 0 {
		exp = MaxTimeOut
	}
	expFromNow := time.Now().Add(time.Duration(exp) * time.Second)
	err = os.Chtimes(f, expFromNow, expFromNow)
	return err
}

func (c *FileCache) Set(key string, value interface{}, exp int) error {
	str, err := EncodeValue(value)
	if err != nil {
		return err
	}

	err = c.saveValue(key, str, exp)
	return err
}

func (c *FileCache) SetDir(path string) {
	c.Dir = path
}

func (c *FileCache) fileName(key string) string {
	f := c.Dir + string(os.PathSeparator) + fmt.Sprintf("%x", md5.Sum([]byte(key)))
	return f
}

// Gc only removes expired entries owned by this cache. The configured
// directory may contain unrelated files that must be preserved.
func (c *FileCache) Delete(key string) error {
	path := c.fileName(key)
	mutex := c.getLock(path)
	mutex.Lock()
	defer mutex.Unlock()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (c *FileCache) Gc() error {
	entries, err := os.ReadDir(c.Dir)
	if os.IsNotExist(err) { return nil }
	if err != nil { return err }
	now := time.Now()
	for _, entry := range entries {
		if !entry.Type().IsRegular() { continue }
		name := entry.Name()
		if len(name) != 32 { continue }
		valid := true
		for _, ch := range name {
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') { valid = false; break }
		}
		if !valid { continue }
		path := c.Dir + string(os.PathSeparator) + name
		lock := c.getLock(path)
		lock.Lock()
		info, infoErr := os.Stat(path)
		if infoErr == nil && info.Mode().IsRegular() && !info.ModTime().After(now) {
			infoErr = os.Remove(path)
		}
		lock.Unlock()
		if infoErr != nil && !os.IsNotExist(infoErr) { return infoErr }
	}
	return nil
}

func NewFileCache() *FileCache {
	return &FileCache{Dir: os.TempDir()}
}
