package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestResolveConnDetails(t *testing.T) {
	// Initialize Viper for testing
	viper.Reset()

	// 1. Direct URL (no config needed)
	details, err := ResolveConnDetails("ws://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "ws://example.com", details.URL)
	assert.Nil(t, details.Headers)

	details, err = ResolveConnDetails("wss://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "wss://example.com", details.URL)
	assert.Nil(t, details.Headers)

	// 2. Alias resolution
	viper.Set("aliases", map[string]string{
		"prod": "wss://api.prod.com",
	})
	details, err = ResolveConnDetails("prod")
	assert.NoError(t, err)
	assert.Equal(t, "wss://api.prod.com", details.URL)
	assert.Nil(t, details.Headers)

	// 3. Bookmark resolution with headers and TLS
	viper.Set("bookmarks", map[string]interface{}{
		"staging": map[string]interface{}{
			"url": "wss://api.staging.com",
			"headers": map[string]string{
				"X-API-Key": "test-key",
			},
			"insecure": true,
			"ca":       "ca.crt",
			"cert":     "client.crt",
			"key":      "client.key",
		},
	})
	details, err = ResolveConnDetails("staging")
	assert.NoError(t, err)
	assert.Equal(t, "wss://api.staging.com", details.URL)
	assert.Equal(t, "test-key", details.Headers["X-API-Key"])
	assert.True(t, details.Insecure)
	assert.Equal(t, "ca.crt", details.CA)
	assert.Equal(t, "client.crt", details.Cert)
	assert.Equal(t, "client.key", details.Key)

	// 3.1 Proxy support in bookmarks
	viper.Set("bookmarks", map[string]interface{}{
		"proxied": map[string]interface{}{
			"url":   "ws://example.com",
			"proxy": "socks5://localhost:1080",
		},
	})
	details, err = ResolveConnDetails("proxied")
	assert.NoError(t, err)
	assert.Equal(t, "ws://example.com", details.URL)
	assert.Equal(t, "socks5://localhost:1080", details.Proxy)

	// 4. Undefined alias
	_, err = ResolveConnDetails("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined alias or bookmark")

	// 5. Bookmark without URL
	viper.Set("bookmarks", map[string]interface{}{
		"invalid": map[string]interface{}{
			"headers": map[string]string{"foo": "bar"},
		},
	})
	_, err = ResolveConnDetails("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no URL")

	// 6. Invalid scheme in URL
	_, err = ResolveConnDetails("http://google.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid WebSocket scheme")
}
