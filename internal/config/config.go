package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Bookmark represents a saved WebSocket endpoint with optional headers and TLS settings.
type Bookmark struct {
	URL      string            `mapstructure:"url"`
	Headers  map[string]string `mapstructure:"headers"`
	Insecure bool              `mapstructure:"insecure"`
	CA       string            `mapstructure:"ca"`
	Cert     string            `mapstructure:"cert"`
	Key      string            `mapstructure:"key"`
	Proxy    string            `mapstructure:"proxy"`
	PingInterval      time.Duration `mapstructure:"ping-interval"`
	PongWait          time.Duration `mapstructure:"pong-wait"`
	Reconnect         bool          `mapstructure:"reconnect"`
	ReconnectBackoff  time.Duration `mapstructure:"reconnect-backoff"`
	ReconnectMax      time.Duration `mapstructure:"reconnect-max"`
	ReconnectAttempts int           `mapstructure:"reconnect-attempts"`
}

// AppConfig represents the root configuration structure for aliases and bookmarks.
type AppConfig struct {
	Aliases   map[string]string   `mapstructure:"aliases"`
	Bookmarks map[string]Bookmark `mapstructure:"bookmarks"`
}

// ConnDetails contains all information needed to establish a connection.
type ConnDetails struct {
	URL      string
	Headers  map[string]string
	Insecure bool
	CA       string
	Cert     string
	Key      string
	Proxy    string
	PingInterval      time.Duration
	PongWait          time.Duration
	Reconnect         bool
	ReconnectBackoff  time.Duration
	ReconnectMax      time.Duration
	ReconnectAttempts int
}

// ResolveConnDetails resolves a short name or a direct URL into the full WebSocket URL and any associated headers.
func ResolveConnDetails(nameOrUrl string) (ConnDetails, error) {
	// 1. Check if it's potentially a URL (contains a protocol separator)
	if strings.Contains(nameOrUrl, "://") {
		if strings.HasPrefix(nameOrUrl, "ws://") || strings.HasPrefix(nameOrUrl, "wss://") {
			return ConnDetails{URL: nameOrUrl}, nil
		}
		return ConnDetails{}, fmt.Errorf("invalid WebSocket scheme in URL %q: only ws:// and wss:// are supported", nameOrUrl)
	}

	// 2. Load current config (aliases and bookmarks) from Viper
	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return ConnDetails{}, fmt.Errorf("unmarshaling config: %w", err)
	}

	// 3. Try resolving as an alias
	if url, ok := cfg.Aliases[nameOrUrl]; ok {
		return ConnDetails{URL: url}, nil
	}

	// 4. Try resolving as a bookmark
	if bookmark, ok := cfg.Bookmarks[nameOrUrl]; ok {
		if bookmark.URL == "" {
			return ConnDetails{}, fmt.Errorf("bookmark %q has no URL", nameOrUrl)
		}
		return ConnDetails(bookmark), nil
	}

	// 5. If not found, it's an error
	return ConnDetails{}, fmt.Errorf("undefined alias or bookmark: %s", nameOrUrl)
}
