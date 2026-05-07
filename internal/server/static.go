package server

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// StaticConfig defines the configuration for a static file server.
type StaticConfig struct {
	Addr     string
	Port     int
	Root     string // Directory or file path
	Path     string // URL path prefix
	IsDir    bool
	Requests uint64
}

// StaticManager manages multiple static HTTP servers.
type StaticManager struct {
	mu      sync.RWMutex
	servers map[int]*staticServerEntry
	logger  Logger
}

type staticServerEntry struct {
	httpSrv *http.Server
	config  *StaticConfig
	cancel  context.CancelFunc
}

// NewStaticManager creates a new StaticManager.
func NewStaticManager(logger Logger) *StaticManager {
	return &StaticManager{
		servers: make(map[int]*staticServerEntry),
		logger:  logger,
	}
}

// Start starts a static file server on the given port.
func (m *StaticManager) Start(config StaticConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[config.Port]; exists {
		return fmt.Errorf("port %d is already in use by another static server", config.Port)
	}

	// Verify root path exists
	info, err := os.Stat(config.Root)
	if err != nil {
		return fmt.Errorf("static root error: %w", err)
	}
	config.IsDir = info.IsDir()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&config.Requests, 1)

		// Serve file or directory
		if !config.IsDir {
			http.ServeFile(w, r, config.Root)
			return
		}

		// Directory serving
		fs := http.FileServer(http.Dir(config.Root))
		// Strip the prefix if specified
		if config.Path != "" && config.Path != "/" {
			http.StripPrefix(config.Path, fs).ServeHTTP(w, r)
		} else {
			fs.ServeHTTP(w, r)
		}
	})

	_, cancel := context.WithCancel(context.Background())

	addr := fmt.Sprintf("%s:%d", config.Addr, config.Port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	entry := &staticServerEntry{
		httpSrv: httpSrv,
		config:  &config,
		cancel:  cancel,
	}
	m.servers[config.Port] = entry

	go func() {
		if m.logger != nil {
			m.logger.Printf("Starting static server on %s (serving %s at %s)\n", addr, config.Root, config.Path)
		}
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if m.logger != nil {
				m.logger.Errorf("Static server on %d error: %v\n", config.Port, err)
			}
			m.mu.Lock()
			delete(m.servers, config.Port)
			m.mu.Unlock()
		}
	}()

	return nil
}

// Stop stops the static server on the given port.
func (m *StaticManager) Stop(port int) error {
	m.mu.Lock()
	entry, exists := m.servers[port]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("no static server running on port %d", port)
	}
	delete(m.servers, port)
	m.mu.Unlock()

	entry.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return entry.httpSrv.Shutdown(ctx)
}

// StopAll stops all managed static servers.
func (m *StaticManager) StopAll() {
	m.mu.Lock()
	ports := make([]int, 0, len(m.servers))
	for port := range m.servers {
		ports = append(ports, port)
	}
	m.mu.Unlock()

	for _, port := range ports {
		_ = m.Stop(port)
	}
}

// GetConfigs returns the current configurations of all active static servers.
func (m *StaticManager) GetConfigs() []StaticConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]StaticConfig, 0, len(m.servers))
	for _, entry := range m.servers {
		configs = append(configs, *entry.config)
	}
	return configs
}

// GenerateMinimalHTML creates a boilerplate HTML client at the target path.
func (m *StaticManager) GenerateMinimalHTML(targetPath string, wsURL string, style string) error {
	var templStr string
	if style != "" {
		var ok bool
		templStr, ok = CannedTemplates[style]
		if !ok {
			// If style is invalid, fallback to random
			templStr = RandomTemplate()
		}
	} else {
		// No style specified, pick random
		templStr = RandomTemplate()
	}

	tmpl, err := template.New("minimal").Parse(templStr)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	f, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", targetPath, err)
	}
	defer f.Close()

	data := struct {
		WSURL string
	}{
		WSURL: wsURL,
	}

	return tmpl.Execute(f, data)
}
