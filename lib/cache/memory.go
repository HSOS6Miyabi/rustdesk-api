package cache

import (
	"container/heap"
	"container/list"
	"errors"
	"reflect"
	"sync"
	"time"
)

type MemoryCache struct {
	data      map[string]*CacheItem
	ll        *list.List    // 用于实现LRU
	pq        PriorityQueue // 用于实现TTL
	quit      chan struct{}
	wake      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	maxBytes  int64
	usedBytes int64
}

type CacheItem struct {
	Key        string
	Value      string
	Expiration int64
	Index      int
	ListEle    *list.Element
}

type PriorityQueue []*CacheItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Expiration < pq[j].Expiration
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*CacheItem)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (m *MemoryCache) Get(key string, value interface{}) error {
	// 使用反射将存储的值设置到传入的指针变量中
	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Ptr {
		return errors.New("value must be a pointer")
	}
	//设为空值
	val.Elem().Set(reflect.Zero(val.Elem().Type()))

	m.mu.Lock()
	if m.data == nil {
		m.mu.Unlock()
		return nil
	}
	var encoded string
	if item, ok := m.data[key]; ok {
		if item.Expiration < time.Now().UnixNano() {
			m.deleteItem(item)
			m.signalEviction()
			m.mu.Unlock()
			return nil
		}
		m.ll.MoveToBack(item.ListEle)
		encoded = item.Value
	}
	m.mu.Unlock()
	if encoded == "" {
		return nil
	}
	return DecodeValue(encoded, value)
}

func (m *MemoryCache) Set(key string, value interface{}, exp int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, err := EncodeValue(value)
	if err != nil {
		return err
	}
	//key 所占用的内存
	keyBytes := int64(len(key))
	//value所占用的内存空间大小
	valueBytes := int64(len(v))
	//判断是否超过最大内存限制
	if m.maxBytes != 0 && m.maxBytes < keyBytes+valueBytes {
		return errors.New("exceed maxBytes")
	}
	if exp <= 0 {
		exp = MaxTimeOut
	}
	expiration := time.Now().Add(time.Duration(exp) * time.Second).UnixNano()
	item, exists := m.data[key]
	if exists {
		m.usedBytes -= int64(len(item.Key) + len(item.Value))
	}
	m.usedBytes += keyBytes + valueBytes
	if exists {
		item.Value = v
		item.Expiration = expiration
		heap.Fix(&m.pq, item.Index)
		m.ll.MoveToBack(item.ListEle)
	} else {
		ele := m.ll.PushBack(key)
		item = &CacheItem{
			Key:        key,
			Value:      v,
			Expiration: expiration,
			ListEle:    ele,
		}
		m.data[key] = item
		heap.Push(&m.pq, item)
	}
	if m.maxBytes != 0 && m.usedBytes > m.maxBytes {
		m.RemoveOldest()
	}
	m.signalEviction()
	return nil
}

func (m *MemoryCache) RemoveOldest() {
	for m.maxBytes != 0 && m.usedBytes > m.maxBytes {
		elem := m.ll.Front()
		if elem != nil {
			key := elem.Value.(string)
			item := m.data[key]
			m.deleteItem(item)
		}
	}
}

// evictExpiredItems removes all expired items from the cache.
func (m *MemoryCache) evictExpiredItems() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UnixNano()
	for m.pq.Len() > 0 {
		item := m.pq[0]
		if item.Expiration > now {
			break
		}
		m.deleteItem(item)
	}
}

// Cache expiration uses the next item's deadline. With no entries, the
// worker remains asleep until an entry is inserted.
func (m *MemoryCache) signalEviction() {
	select {case m.wake <- struct{}{}:default:}
}

func (m *MemoryCache) startEviction() {
	go func() {
		for {
			m.mu.Lock()
			if len(m.pq)==0 {
				m.mu.Unlock()
				select {
				case <-m.wake:continue
				case <-m.quit:return
				}
			}
			wait:=time.Until(time.Unix(0,m.pq[0].Expiration))
			m.mu.Unlock()
			if wait<=0 {
				m.evictExpiredItems()
				continue
			}
			timer:=time.NewTimer(wait)
			select {
			case <-m.wake:timer.Stop()
			case <-timer.C:m.evictExpiredItems()
			case <-m.quit:timer.Stop();return
			}
		}
	}()
}

func (m *MemoryCache) stopEviction() {
	m.closeOnce.Do(func(){close(m.quit)})
}

// deleteItem removes a key from the cache.
func (m *MemoryCache) deleteItem(item *CacheItem) {
	m.ll.Remove(item.ListEle)
	m.usedBytes -= int64(len(item.Key)) + int64(len(item.Value))
	heap.Remove(&m.pq, item.Index)
	delete(m.data, item.Key)
}

func (m *MemoryCache) Gc() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string]*CacheItem)
	m.ll = list.New()
	m.pq = make(PriorityQueue, 0)
	heap.Init(&m.pq)
	m.usedBytes = 0
	m.signalEviction()
	return nil
}

// NewMemoryCache creates a new MemoryCache.default maxBytes is 0, means no limit.
func NewMemoryCache(maxBytes int64) *MemoryCache {
	cache := &MemoryCache{
		data:     make(map[string]*CacheItem),
		pq:       make(PriorityQueue, 0),
		quit:     make(chan struct{}),
		wake:     make(chan struct{},1),
		ll:       list.New(),
		maxBytes: maxBytes,
	}
	heap.Init(&cache.pq)
	cache.startEviction()
	return cache
}

// Close stops the expiration worker when the cache is no longer used.
func (m *MemoryCache) Close() error {
    if m != nil { m.stopEviction() }
    return nil
}
