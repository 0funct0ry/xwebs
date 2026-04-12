package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityManager_IPFiltering(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowIPs = []string{"127.0.0.1", "192.168.1.0/24"}
	opts.DenyIPs = []string{"192.168.1.50"}

	sm, err := NewSecurityManager(opts)
	require.NoError(t, err)

	tests := []struct {
		addr    string
		allowed bool
	}{
		{"127.0.0.1:1234", true},
		{"127.0.0.2:1234", false},
		{"192.168.1.1:1234", true},
		{"192.168.1.50:1234", false},
		{"192.168.1.254:1234", true},
		{"10.0.0.1:1234", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.allowed, sm.IsIPAllowed(tt.addr), "IP %s allowance mismatch", tt.addr)
	}
}

func TestSecurityManager_RateLimiting(t *testing.T) {
	opts := DefaultOptions()
	opts.RateLimit = "10/s,100/s" // 10 per-client, 100 global

	sm, err := NewSecurityManager(opts)
	require.NoError(t, err)

	addr := "1.2.3.4:1234"

	// Per-client check
	for i := 0; i < 10; i++ {
		assert.True(t, sm.CheckRateLimit(addr), "Request %d should be allowed", i)
	}
	assert.False(t, sm.CheckRateLimit(addr), "Request 11 should be ratelimited")

	// Global check (other IP)
	addr2 := "5.6.7.8:1234"
	for i := 0; i < 10; i++ {
		assert.True(t, sm.CheckRateLimit(addr2), "Request %d from IP2 should be allowed", i)
	}
}

func TestSecurityManager_OriginFiltering(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowedOrigins = []string{"http://localhost:3000", "https://example.com"}

	sm, err := NewSecurityManager(opts)
	require.NoError(t, err)

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"http://localhost:3000", true},
		{"https://example.com", true},
		{"https://evil.com", false},
		{"", true}, // Missing origin is allowed
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		if tt.origin != "" {
			req.Header.Set("Origin", tt.origin)
		}
		assert.Equal(t, tt.allowed, sm.IsOriginAllowed(req), "Origin %s allowance mismatch", tt.origin)
	}
}

func TestSecurityManager_Middleware(t *testing.T) {
	opts := DefaultOptions()
	opts.AllowIPs = []string{"127.0.0.1"}

	sm, err := NewSecurityManager(opts)
	require.NoError(t, err)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := sm.Middleware(nextHandler)

	// Allowed IP
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Denied IP
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w = httptest.NewRecorder()
	middleware.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
