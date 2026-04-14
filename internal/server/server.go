package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/kv"
	"github.com/0funct0ry/xwebs/internal/observability"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/0funct0ry/xwebs/ui"
	"github.com/gorilla/websocket"
)

// Server represents the WebSocket server.
type Server struct {
	opts     *Options
	httpSrv  *http.Server
	upgrader websocket.Upgrader
	registry *handler.Registry
	
	mu          sync.Mutex
	connections map[string]*ws.Connection
	kvStore     *kv.Store
	wg          sync.WaitGroup
	startTime   time.Time
	securityMgr *SecurityManager
}

// New creates a new WebSocket server with the given options.
func New(opts ...Option) *Server {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	s := &Server{
		opts: options,
		upgrader: websocket.Upgrader{},
		connections: make(map[string]*ws.Connection),
		kvStore:     kv.NewStore(),
		startTime:   time.Now(),
	}

	sm, err := NewSecurityManager(options)
	if err != nil {
		// Log error and proceed with nil security manager if it fails?
		// Better failed fast or handle it. Since New() doesn't return error,
		// we might need to change it or handle it here.
		// Actually I can return nil or handle it in Start.
		// Let's make New return (*Server, error) if possible, but it's used a lot.
		// I'll just fmt.Fprintf(os.Stderr, ...) for now or panic if it's critical.
		fmt.Fprintf(os.Stderr, "Error initializing security manager: %v\n", err)
	}
	s.securityMgr = sm

	if s.securityMgr != nil {
		s.upgrader.CheckOrigin = s.securityMgr.IsOriginAllowed
	} else {
		s.upgrader.CheckOrigin = func(r *http.Request) bool {
			return true // Default: allow all origins
		}
	}

	s.registry = handler.NewRegistry()
	if len(options.Handlers) > 0 {
		s.registry.AddHandlers(options.Handlers)
	}

	return s
}

// UpdateOptions applies the given options to the server.
func (s *Server) UpdateOptions(opts ...Option) {
	for _, opt := range opts {
		opt(s.opts)
	}
}

// Start launches the server and listens for incoming connections.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	hasRoot := false
	for _, path := range s.opts.Paths {
		pattern := path
		if path == "/" {
			hasRoot = true
			pattern = "/{$}" // Exact match for root in Go 1.22+
		}
		mux.HandleFunc(pattern, s.serveWS)
	}

	// Register UI and Status routes
	if s.opts.UIEnabled {
		// Register UI handler as a catch-all.
		// Specific routes like /api/*, /api/metrics, and exact WS paths (e.g., /{$}) will take precedence.
		mux.Handle("/", ui.Handler())
	} else if !hasRoot {
		mux.HandleFunc("/{$}", s.serveStatus)
	}

	// Register API routes
	mux.HandleFunc("GET /api/health", s.handleAPIHealth)

	if s.opts.APIEnabled {
		mux.HandleFunc("GET /api/status", s.handleAPIStatus)
		mux.HandleFunc("GET /api/clients", s.handleAPIClients)
		
		mux.HandleFunc("GET /api/handlers", s.handleListHandlers)
		mux.HandleFunc("POST /api/handlers", s.handleCreateHandler)
		mux.HandleFunc("GET /api/handlers/{name}", s.handleGetHandler)
		mux.HandleFunc("PUT /api/handlers/{name}", s.handleUpdateHandler)
		mux.HandleFunc("DELETE /api/handlers/{name}", s.handleDeleteHandler)

		mux.HandleFunc("GET /api/kv", s.handleListKV)
		mux.HandleFunc("GET /api/kv/{key}", s.handleGetKV)
		mux.HandleFunc("POST /api/kv/{key}", s.handleSetKV)
		mux.HandleFunc("DELETE /api/kv/{key}", s.handleDeleteKV)
	}

	if s.opts.MetricsEnabled {
		mux.Handle("/api/metrics", observability.Handler())
	}

	var handler http.Handler = mux
	if s.securityMgr != nil {
		handler = s.securityMgr.Middleware(mux)
		
		// Start security manager cleanup loop
		go func() {
			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					s.securityMgr.Cleanup()
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.opts.Port),
		Handler:      handler,
		ReadTimeout:  s.opts.ReadTimeout,
		WriteTimeout: s.opts.WriteTimeout,
		IdleTimeout:  s.opts.IdleTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		if s.opts.TLSEnabled {
			if s.opts.Verbose {
				s.logf("Starting TLS server on %s\n", s.httpSrv.Addr)
			}
			// Verify cert and key files
			if _, err := os.Stat(s.opts.CertFile); err != nil {
				errChan <- fmt.Errorf("certificate file error: %w", err)
				return
			}
			if _, err := os.Stat(s.opts.KeyFile); err != nil {
				errChan <- fmt.Errorf("key file error: %w", err)
				return
			}
			if err := s.httpSrv.ListenAndServeTLS(s.opts.CertFile, s.opts.KeyFile); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		} else {
			if s.opts.Verbose {
				s.logf("Starting server on %s\n", s.httpSrv.Addr)
			}
			if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}
	}()

	select {
	case <-ctx.Done():
		return s.Stop()
	case err := <-errChan:
		return err
	}
}

// Stop gracefully shuts down the server and all active connections.
func (s *Server) Stop() error {
	if s.httpSrv == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP server
	err := s.httpSrv.Shutdown(ctx)

	// Close all active WebSocket connections
	s.mu.Lock()
	for id, conn := range s.connections {
		if s.opts.Verbose {
			s.logf("Closing connection %s\n", id)
		}
		_ = conn.Close()
	}
	s.mu.Unlock()

	// Wait for all connections to finish cleaning up
	s.wg.Wait()

	return err
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") != "websocket" {
		if s.opts.Verbose {
			s.logf("[http] connection received: %s %s from %s (non-websocket)\n", r.Method, r.URL.Path, r.RemoteAddr)
		}
		if s.opts.UIEnabled && (r.URL.Path == "/" || r.URL.Path == "" || strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/favicon.svg" || r.URL.Path == "/icons.svg") {
			ui.Handler().ServeHTTP(w, r)
			return
		}
		s.serveStatus(w, r)
		return
	}

	if s.opts.Verbose {
		s.logf("[http] attempting websocket upgrade for %s from %s\n", r.URL.Path, r.RemoteAddr)
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if s.opts.Verbose {
			s.errorf("[http] upgrade error: %v\n", err)
		}
		return
	}

	if s.opts.Verbose {
		s.logf("[ws] upgrade successful for %s from %s\n", r.URL.Path, r.RemoteAddr)
	}

	// Create ws.Connection wrapper
	// We need to pass DialOptions to NewConnection even though this is server-side
	wsOpts := &ws.DialOptions{
		Verbose: s.opts.Verbose,
		// TODO: Map other server options to wsOpts if needed
	}
	
	wsConn := ws.NewConnection(conn, r.URL.String(), nil, wsOpts)
	
	s.mu.Lock()
	s.connections[wsConn.ID] = wsConn
	s.mu.Unlock()

	observability.ActiveConnections.Inc()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.connections, wsConn.ID)
			s.mu.Unlock()
			observability.ActiveConnections.Dec()
		}()

		// Track messages for metrics
		msgCh := wsConn.Subscribe()
		go func() {
			for {
				select {
				case msg, ok := <-msgCh:
					if !ok {
						return
					}
					if msg.Metadata.Direction == "received" {
						observability.MessagesReceived.Inc()
					} else {
						observability.MessagesSent.Inc()
					}
				case <-wsConn.Done():
					return
				}
			}
		}()
		defer wsConn.Unsubscribe(msgCh)

		// Start message loops
		wsConn.Start()

		// Initialize dispatcher
		dispatcher := handler.NewDispatcher(
			s.registry,
			wsConn,
			s.opts.TemplateEngine,
			s.opts.Verbose,
			s.opts.Variables,
			nil, // sessionVars
			s.opts.Sandbox,
			s.opts.Allowlist,
			s, // Server implements ServerStatProvider
		)
			
			// Setup logging/error handlers for dispatcher
			dispatcher.Log = func(f string, a ...interface{}) {
				s.logf(f, a...)
			}
			dispatcher.Error = func(f string, a ...interface{}) {
				s.errorf(f, a...)
			}

			dispatcher.Start(context.Background())
			dispatcher.HandleConnect()
			defer dispatcher.HandleDisconnect() // Handle disconnect when read loop exits

		// Wait for connection to close
		<-wsConn.Done()
	}()
}
func (s *Server) serveStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	connCount := len(s.connections)
	s.mu.Unlock()

	uptime := time.Since(s.startTime).Round(time.Second)

	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		pathsJSON, _ := json.Marshal(s.opts.Paths)
		fmt.Fprintf(w, `{"status": "running", "uptime": "%s", "connections": %d, "paths": %s}`,
			uptime, connCount, string(pathsJSON))
		return
	}

	scheme := "ws"
	statusColor := "#27ae60"
	if r.TLS != nil || s.opts.TLSEnabled {
		scheme = "wss"
		statusColor = "#2980b9" // Blue for secure
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>xwebs server</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f4f7f6; }
        .card { background: white; padding: 30px; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; margin-top: 0; }
        .status { display: inline-block; padding: 4px 12px; border-radius: 20px; background: %s; color: white; font-size: 0.8em; font-weight: bold; }
        ul { padding-left: 20px; }
        code { background: #eee; padding: 2px 5px; border-radius: 4px; }
        footer { margin-top: 40px; font-size: 0.8em; color: #7f8c8d; text-align: center; }
    </style>
</head>
<body>
    <div class="card">
        <h1>xwebs <span class="status">RUNNING</span></h1>
        <p>WebSocket server is active and accepting connections.</p>
        <hr>
        <p><strong>Uptime:</strong> %s</p>
        <p><strong>Active Connections:</strong> %d</p>
        <p><strong>WebSocket Paths:</strong></p>
        <ul>
`, statusColor, uptime, connCount)

	for _, path := range s.opts.Paths {
		fmt.Fprintf(w, "            <li><code>%s</code></li>\n", path)
	}

	fmt.Fprintf(w, `        </ul>
        <p>Connect using <code>%s://%s%s</code></p>
    </div>
    <footer>Powered by xwebs</footer>
</body>
</html>
`, scheme, r.Host, s.opts.Paths[0])
}

// GetClientCount returns the number of active connections.
func (s *Server) GetClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.connections)
}

// GetUptime returns the server uptime.
func (s *Server) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// GetClients returns a list of active connection metadata.
func (s *Server) GetClients() []template.ClientInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	clients := make([]template.ClientInfo, 0, len(s.connections))
	for _, conn := range s.connections {
		uptime := time.Since(conn.ConnectedAt())
		clients = append(clients, template.ClientInfo{
			ID:          conn.ID,
			RemoteAddr:  conn.RemoteAddr(),
			ConnectedAt: conn.ConnectedAt(),
			Uptime:      uptime,
			UptimeStr:   template.FormatUptime(uptime),
			MsgsIn:      conn.MsgsIn(),
			MsgsOut:     conn.MsgsOut(),
		})
	}
	return clients
}

// GetClient returns metadata for a specific client by ID.
func (s *Server) GetClient(id string) (template.ClientInfo, bool) {
	s.mu.Lock()
	conn, ok := s.connections[id]
	s.mu.Unlock()

	if !ok {
		return template.ClientInfo{}, false
	}

	uptime := time.Since(conn.ConnectedAt())
	return template.ClientInfo{
		ID:          conn.ID,
		RemoteAddr:  conn.RemoteAddr(),
		ConnectedAt: conn.ConnectedAt(),
		Uptime:      uptime,
		UptimeStr:   template.FormatUptime(uptime),
		MsgsIn:      conn.MsgsIn(),
		MsgsOut:     conn.MsgsOut(),
	}, true
}

// Broadcast sends a message to all connected clients.
func (s *Server) Broadcast(msg *ws.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		_ = conn.Write(msg)
	}
	return nil
}

// Kick disconnects a specific client by ID with an optional close code and reason.
func (s *Server) Kick(id string, code int, reason string) error {
	s.mu.Lock()
	conn, ok := s.connections[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("client %s not found", id)
	}

	if code == 0 {
		code = websocket.CloseGoingAway
	}
	if reason == "" {
		reason = "Kicked by admin"
	}

	return conn.CloseWithCode(code, reason)
}

// Send sends a message to a specific client by ID.
func (s *Server) Send(id string, msg *ws.Message) error {
	s.mu.Lock()
	conn, ok := s.connections[id]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("client %s not found", id)
	}

	return conn.Write(msg)
}

// GetStatus returns the current server status.
func (s *Server) GetStatus() string {
	return "running"
}

// GetTemplateEngine returns the server's template engine.
func (s *Server) GetTemplateEngine() *template.Engine {
	return s.opts.TemplateEngine
}

// AddHandler adds a new message handler to the server at runtime.
func (s *Server) AddHandler(h handler.Handler) error {
	return s.registry.Add(h)
}

// UpdateHandler replaces an existing message handler at runtime.
func (s *Server) UpdateHandler(h handler.Handler) error {
	return s.registry.UpdateHandler(h)
}

// DeleteHandler removes a message handler from the server at runtime.
func (s *Server) DeleteHandler(name string) error {
	return s.registry.Delete(name)
}

// RenameHandler renames a message handler at runtime.
func (s *Server) RenameHandler(oldName, newName string) error {
	return s.registry.RenameHandler(oldName, newName)
}

// GetHandlers returns all currently registered handlers.
func (s *Server) GetHandlers() []handler.Handler {
	return s.registry.Handlers()
}

// ReloadHandlers replaces all handlers in the registry.
func (s *Server) ReloadHandlers(handlers []handler.Handler, variables map[string]interface{}) {
	s.registry.ReplaceHandlers(handlers)
	s.opts.Handlers = handlers
	s.opts.Variables = variables
}

// EnableHandler enables a handler by name.
func (s *Server) EnableHandler(name string) error {
	return s.registry.EnableHandler(name)
}

// DisableHandler disables a handler by name.
func (s *Server) DisableHandler(name string) error {
	return s.registry.DisableHandler(name)
}

// GetHandlerStats returns statistics for a handler.
func (s *Server) GetHandlerStats(name string) (uint64, time.Duration, uint64, bool) {
	return s.registry.GetStats(name)
}

// IsHandlerDisabled returns true if the handler is disabled.
func (s *Server) IsHandlerDisabled(name string) bool {
	return s.registry.IsDisabled(name)
}

func (s *Server) logf(format string, a ...interface{}) {
	if s.opts.Logger != nil {
		s.opts.Logger.Printf(format, a...)
	} else {
		fmt.Printf(format, a...)
	}
}

func (s *Server) errorf(format string, a ...interface{}) {
	if s.opts.Logger != nil {
		s.opts.Logger.Errorf(format, a...)
	} else {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}
