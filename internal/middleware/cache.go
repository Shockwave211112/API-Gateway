package middleware

import (
	"sync"
	"time"
)

type TokenCache struct {
	mu    sync.RWMutex
	items map[string]cacheValue
}

type cacheValue struct {
	valid     bool
	expiresAt time.Time
}

func NewTokenCache() *TokenCache {
	return &TokenCache{
		items: make(map[string]cacheValue),
	}
}

func (tc *TokenCache) Get(hash string) (valid, found bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	value, ok := tc.items[hash]
	if !ok {
		return false, false
	}
	return value.valid && time.Now().Before(value.expiresAt), true
}

func (tc *TokenCache) Set(hash string, valid bool, ttl time.Duration) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.items[hash] = cacheValue{
		valid,
		time.Now().Add(ttl),
	}
}
