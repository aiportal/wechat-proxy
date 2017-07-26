package wxproxy

import (
	"sync"
	"time"
)

type cacheItem struct {
	value  interface{}
	expire int64
}

type cacheMap struct {
	m      map[string]cacheItem
	lock   sync.RWMutex
	config struct {
		duration time.Duration // duration for cache item in memory.
		limit    int           // limit for cache item count.
	}
}

func NewCacheMap(duration time.Duration, limit int) *cacheMap {
	tm := new(cacheMap)
	tm.m = make(map[string]cacheItem)
	tm.config.duration = duration
	tm.config.limit = limit
	return tm
}

func (tm *cacheMap) Set(key string, value interface{}) {
	expire := time.Now().Add(tm.config.duration).Unix()
	tm.lock.Lock()
	defer tm.lock.Unlock()
	tm.m[key] = cacheItem{value: value, expire: expire}
}

func (tm *cacheMap) Get(key string) (value interface{}, success bool) {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	if token, ok := tm.m[key]; ok {
		value = token.value
		success = (time.Now().Unix() < token.expire)
	}
	return
}

func (tm *cacheMap) Remove(key string) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	delete(tm.m, key)
}

func (tm *cacheMap) Len() int {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	return len(tm.m)
}

func (tm *cacheMap) Clean() {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	now := time.Now().Unix()
	for k := range tm.m {
		if tm.m[k].expire < now {
			delete(tm.m, k)
		}
	}
}

func (tm *cacheMap) Shrink() {
	if tm.Len() < tm.config.limit {
		return
	}
	go tm.Clean()
}
