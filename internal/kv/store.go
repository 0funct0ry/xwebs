package kv

import (
	"sync"
)

// Store represents a thread-safe key-value store.
type Store struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewStore creates a new key-value store.
func NewStore() *Store {
	return &Store{
		data: make(map[string]interface{}),
	}
}

// Get retrieves a value from the store.
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// Set stores a value in the store.
func (s *Store) Set(key string, val interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// List returns a copy of all entries in the store.
func (s *Store) List() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	copy := make(map[string]interface{}, len(s.data))
	for k, v := range s.data {
		copy[k] = v
	}
	return copy
}
