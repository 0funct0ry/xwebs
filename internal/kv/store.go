package kv

import (
	"sync"
	"time"
)

type item struct {
	value     interface{}
	expiresAt time.Time
}

// Store represents a thread-safe key-value store with TTL support.
type Store struct {
	mu   sync.RWMutex
	data map[string]item
}

// NewStore creates a new key-value store and starts a cleanup goroutine.
func NewStore() *Store {
	s := &Store{
		data: make(map[string]item),
	}
	go s.cleanupLoop()
	return s
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for k, v := range s.data {
		if !v.expiresAt.IsZero() && now.After(v.expiresAt) {
			delete(s.data, k)
		}
	}
}

// Get retrieves a value from the store, checking for expiration.
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	val, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if !val.expiresAt.IsZero() && time.Now().After(val.expiresAt) {
		s.mu.Lock()
		delete(s.data, key)
		s.mu.Unlock()
		return nil, false
	}

	return val.value, true
}

// Set stores a value in the store with an optional TTL.
func (s *Store) Set(key string, val interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.data[key] = item{
		value:     val,
		expiresAt: expiresAt,
	}
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// List returns a copy of all non-expired entries in the store.
func (s *Store) List() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	res := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		if v.expiresAt.IsZero() || now.Before(v.expiresAt) {
			res[k] = v.value
		}
	}
	return res
}
