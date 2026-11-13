package cache

import (
    "encoding/json"
    "errors"
    "reflect"
)

type Handler interface {
	Get(key string, value interface{}) error
	Set(key string, value interface{}, exp int) error
	Delete(key string) error
	Gc() error
}

// MaxTimeOut 最大超时时间

const (
	TypeMem    = "memory"
	TypeRedis  = "redis"
	TypeFile   = "file"
	MaxTimeOut = 365 * 24 * 3600
)

func New(typ string) Handler {
	var cache Handler
	switch typ {
	case TypeFile:
		cache = NewFileCache()
	case TypeRedis:
		// This constructor has no connection options; use an in-memory cache.
		// A configured Redis cache is created with NewRedis or WrapRedis.
		cache = NewMemoryCache(0)
	case TypeMem: // memory
		cache = NewMemoryCache(0)
	default:
		cache = NewMemoryCache(0)
	}
	return cache
}

func zeroCacheValue(value interface{}) error {
    target := reflect.ValueOf(value)
    if target.Kind() != reflect.Ptr || target.IsNil() {
        return errors.New("cache result must be a non-nil pointer")
    }
    target.Elem().Set(reflect.Zero(target.Elem().Type()))
    return nil
}

func EncodeValue(value interface{}) (string, error) {
	/*if v, ok := value.(string); ok {
		return v, nil
	}
	if v, ok := value.([]byte); ok {
		return string(v), nil
	}*/
	b, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DecodeValue(value string, rtv interface{}) error {
	//判断rtv的类型是否是string，如果是string，直接赋值并返回
	/*switch rtv.(type) {
	case *string:
		*(rtv.(*string)) = value
		return nil
	case *[]byte:
		*(rtv.(*[]byte)) = []byte(value)
		return nil
	//struct
	case *interface{}:
		err := json.Unmarshal(([]byte)(value), rtv)
		return err
	default:
		err := json.Unmarshal(([]byte)(value), rtv)
		return err
	}
	*/
	err := json.Unmarshal(([]byte)(value), rtv)
	return err
}
