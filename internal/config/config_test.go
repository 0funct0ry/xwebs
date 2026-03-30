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
	url, headers, err := ResolveConnDetails("ws://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "ws://example.com", url)
	assert.Nil(t, headers)

	url, headers, err = ResolveConnDetails("wss://example.com")
	assert.NoError(t, err)
	assert.Equal(t, "wss://example.com", url)
	assert.Nil(t, headers)

	// 2. Alias resolution
	viper.Set("aliases", map[string]string{
		"prod": "wss://api.prod.com",
	})
	url, headers, err = ResolveConnDetails("prod")
	assert.NoError(t, err)
	assert.Equal(t, "wss://api.prod.com", url)
	assert.Nil(t, headers)

	// 3. Bookmark resolution with headers
	viper.Set("bookmarks", map[string]interface{}{
		"staging": map[string]interface{}{
			"url": "wss://api.staging.com",
			"headers": map[string]string{
				"X-API-Key": "test-key",
			},
		},
	})
	url, headers, err = ResolveConnDetails("staging")
	assert.NoError(t, err)
	assert.Equal(t, "wss://api.staging.com", url)
	assert.Equal(t, "test-key", headers["X-API-Key"])

	// 4. Undefined alias
	_, _, err = ResolveConnDetails("unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined alias or bookmark")

	// 5. Bookmark without URL
	viper.Set("bookmarks", map[string]interface{}{
		"invalid": map[string]interface{}{
			"headers": map[string]string{"foo": "bar"},
		},
	})
	_, _, err = ResolveConnDetails("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no URL")

	// 6. Invalid scheme in URL
	_, _, err = ResolveConnDetails("http://google.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid WebSocket scheme")
}
