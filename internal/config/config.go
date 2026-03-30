package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Bookmark represents a saved WebSocket endpoint with optional headers.
type Bookmark struct {
	URL     string            `mapstructure:"url"`
	Headers map[string]string `mapstructure:"headers"`
}

// AppConfig represents the root configuration structure for aliases and bookmarks.
type AppConfig struct {
	Aliases   map[string]string   `mapstructure:"aliases"`
	Bookmarks map[string]Bookmark `mapstructure:"bookmarks"`
}

// ResolveConnDetails resolves a short name or a direct URL into the full WebSocket URL and any associated headers.
func ResolveConnDetails(nameOrUrl string) (string, map[string]string, error) {
	// 1. Check if it's potentially a URL (contains a protocol separator)
	if strings.Contains(nameOrUrl, "://") {
		if strings.HasPrefix(nameOrUrl, "ws://") || strings.HasPrefix(nameOrUrl, "wss://") {
			return nameOrUrl, nil, nil
		}
		return "", nil, fmt.Errorf("invalid WebSocket scheme in URL %q: only ws:// and wss:// are supported", nameOrUrl)
	}

	// 2. Load current config (aliases and bookmarks) from Viper
	var cfg AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return "", nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// 3. Try resolving as an alias
	if url, ok := cfg.Aliases[nameOrUrl]; ok {
		return url, nil, nil
	}

	// 4. Try resolving as a bookmark
	if bookmark, ok := cfg.Bookmarks[nameOrUrl]; ok {
		if bookmark.URL == "" {
			return "", nil, fmt.Errorf("bookmark '%s' has no URL", nameOrUrl)
		}
		return bookmark.URL, bookmark.Headers, nil
	}

	// 5. If not found, it's an error
	return "", nil, fmt.Errorf("undefined alias or bookmark: %s", nameOrUrl)
}
