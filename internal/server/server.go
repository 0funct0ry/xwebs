package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/kv"
	"github.com/0funct0ry/xwebs/internal/observability"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/0funct0ry/xwebs/ui"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type serverState int32

const (
	serverRunning serverState = iota
	serverDraining
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
	topics      *TopicStore
	wg          sync.WaitGroup
	startTime   time.Time
	securityMgr *SecurityManager
	staticMgr   *StaticManager
	redisMgr    handler.RedisManager
	mqttMgr     handler.MQTTManager
	natsMgr     handler.NATSManager
	sseManager  *SSEManager
	httpMocks   map[string]template.HTTPMockResponse
	httpMocksMu sync.RWMutex

	state     serverState
	paused    atomic.Bool
	pauseCond *sync.Cond

	sourceMu      sync.Mutex
	sourceCancels map[string]context.CancelFunc
}

// New creates a new WebSocket server with the given options.
func New(opts ...Option) (*Server, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	s := &Server{
		opts:        options,
		upgrader:    websocket.Upgrader{},
		connections: make(map[string]*ws.Connection),
		kvStore:     kv.NewStore(),
		topics:      newTopicStore(),
		startTime:   time.Now(),
		state:       serverRunning,
		staticMgr:   NewStaticManager(options.Logger),
		redisMgr:    options.RedisManager,
		mqttMgr:     options.MQTTManager,
		natsMgr:     options.NATSManager,
		httpMocks:   make(map[string]template.HTTPMockResponse),
	}

	if s.mqttMgr == nil {
		s.mqttMgr = handler.NewMQTTManager()
	}
	if s.natsMgr == nil {
		s.natsMgr = handler.NewNATSManager()
	}

	var logf func(string, ...interface{})
	if options.Logger != nil {
		logf = options.Logger.Printf
	}
	s.sseManager = NewSSEManager(options.SSEStreams, options.Verbose, logf)
	s.sourceCancels = make(map[string]context.CancelFunc)
	s.pauseCond = sync.NewCond(&s.mu)

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

	s.registry = handler.NewRegistry(handler.ServerMode)
	if len(options.Handlers) > 0 {
		if err := s.registry.AddHandlers(options.Handlers); err != nil {
			return nil, err
		}
	}

	return s, nil
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

	for _, path := range s.opts.Paths {
		pattern := path
		if path == "/" {
			pattern = "/{$}" // Exact match for root in Go 1.22+
		}
		mux.HandleFunc(pattern, s.serveWS)
	}

	// Register UI and Status routes
	// Note: Specific routes like /api/* and /sse/* take precedence over the root handler.
	mux.Handle("/", http.HandlerFunc(s.handleHTTPMock))

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

	// Register SSE routes
	mux.HandleFunc("GET /sse/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		s.sseManager.HandleSSE(name)(w, r)
	})

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

	// Start initial static serving
	// Start initial static serving
	if s.opts.StaticServeDir != "" || s.opts.StaticServeFile != "" || s.opts.StaticGenerate {
		port := s.opts.StaticServePort
		if port == 0 {
			port = 9090
		}
		path := s.opts.StaticServePath
		if path == "" {
			path = "/"
		}

		root := s.opts.StaticServeDir
		serveIsFile := false
		if root == "" {
			root = s.opts.StaticServeFile
			serveIsFile = true
		}
		if root == "" && s.opts.StaticGenerate {
			root = "index.html"
			serveIsFile = true
		}

		// Handle generation logic
		if s.opts.StaticGenerate {
			if _, err := os.Stat(root); os.IsNotExist(err) {
				wsURL := s.GetURL(s.opts.Paths[0])
				if err := s.staticMgr.GenerateMinimalHTML(root, wsURL, s.opts.StaticGenerateStyle); err != nil {
					return fmt.Errorf("generating minimal HTML: %w", err)
				}
				s.logf("✓ Generated minimal HTML at %s\n", root)
			}
		}

		if err := s.staticMgr.Start(StaticConfig{
			Port:  port,
			Root:  root,
			Path:  path,
			IsDir: !serveIsFile,
		}); err != nil {
			return fmt.Errorf("starting initial static server: %w", err)
		}
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

	// Start source handlers (e.g. redis-subscribe)
	s.startSourceHandlers(ctx)

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

	// Close all managed static servers
	s.staticMgr.StopAll()

	// Stop all source handlers
	s.stopSourceHandlers()

	// Wait for all connections to finish cleaning up
	s.wg.Wait()

	return err
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	if s.IsDraining() {
		if s.opts.Verbose {
			s.logf("[http] rejecting connection from %s: server is draining\n", r.RemoteAddr)
		}
		http.Error(w, "Server is draining", http.StatusServiceUnavailable)
		return
	}

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
		observability.IncrementTotalErrors()
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
	observability.IncrementTotalConnections()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			// Clean up all topic subscriptions for this connection.
			s.topics.UnsubscribeAll(wsConn.ID)
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
						observability.IncrementMessagesReceived()
					} else {
						observability.MessagesSent.Inc()
						observability.IncrementMessagesSent()
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
			s,        // Server implements ServerStatProvider
			s.topics, // Server.topics implements TopicManager
			s,        // Server implements KVManager
			s.redisMgr,
			s.mqttMgr,
			s.natsMgr,
			s.opts.OllamaURL,
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

// GetTopics returns metadata for all active topics.
func (s *Server) GetTopics() []template.TopicInfo {
	return s.topics.GetTopics()
}

// GetTopic returns metadata for a single topic by name.
func (s *Server) GetTopic(name string) (template.TopicInfo, bool) {
	return s.topics.GetTopic(name)
}

// PublishToTopic fans out msg to all subscribers of the named topic.
func (s *Server) PublishToTopic(topic string, msg *ws.Message) (int, error) {
	return s.topics.Publish(topic, msg)
}

// PublishSticky stores msg as the retained value for topic and fans it out to current subscribers.
func (s *Server) PublishSticky(topic string, msg *ws.Message) (int, error) {
	return s.topics.PublishSticky(topic, msg)
}

// ClearRetained removes the retained message for a topic.
func (s *Server) ClearRetained(topic string) {
	s.topics.ClearRetained(topic)
}

// SubscribeClientToTopic manually subscribes a connected client to a topic.
// It looks up the live connection by clientID and registers it with the TopicStore.
func (s *Server) SubscribeClientToTopic(clientID, topic string) error {
	s.mu.Lock()
	conn, ok := s.connections[clientID]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("client %q not found", clientID)
	}
	s.topics.Subscribe(clientID, conn, topic)
	return nil
}

// UnsubscribeClientFromTopic removes a client from a specific topic.
// Returns the number of remaining subscribers.
func (s *Server) UnsubscribeClientFromTopic(clientID, topic string) (int, error) {
	s.mu.Lock()
	_, ok := s.connections[clientID]
	s.mu.Unlock()
	if !ok {
		return 0, fmt.Errorf("client %q not found", clientID)
	}
	remaining := s.topics.Unsubscribe(clientID, topic)
	return remaining, nil
}

// UnsubscribeClientFromAllTopics removes a client from every topic it is subscribed to.
// Returns the list of topic names from which the client was removed.
func (s *Server) UnsubscribeClientFromAllTopics(clientID string) ([]string, error) {
	s.mu.Lock()
	_, ok := s.connections[clientID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("client %q not found", clientID)
	}
	affected := s.topics.UnsubscribeAll(clientID)
	return affected, nil
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

// Broadcast sends a message to all connected clients except those listed in excludeIDs.
// Returns the number of clients the message was successfully delivered to.
func (s *Server) Broadcast(msg *ws.Message, excludeIDs ...string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	excluded := make(map[string]bool)
	for _, id := range excludeIDs {
		excluded[id] = true
	}

	count := 0
	for id, conn := range s.connections {
		if excluded[id] {
			continue
		}
		if err := conn.Write(msg); err == nil {
			count++
		}
	}
	return count
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

// SendToSSE delivers an event to a named SSE stream.
func (s *Server) SendToSSE(stream, event, data, id string) error {
	if s.sseManager == nil {
		return fmt.Errorf("SSE manager not initialized")
	}
	return s.sseManager.SendToSSE(stream, event, data, id)
}

// UpdateSSEStreamConfig updates the configuration of an SSE stream.
func (s *Server) UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error {
	if s.sseManager == nil {
		return fmt.Errorf("SSE manager not initialized")
	}
	return s.sseManager.UpdateStreamConfig(stream, onNoConsumers, bufferSize)
}

// RegisterHTTPMock registers a canned HTTP response at a specific path.
func (s *Server) RegisterHTTPMock(path string, mock template.HTTPMockResponse) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	s.httpMocksMu.Lock()
	defer s.httpMocksMu.Unlock()
	s.httpMocks[path] = mock
	return nil
}

func (s *Server) handleHTTPMock(w http.ResponseWriter, r *http.Request) {
	s.httpMocksMu.RLock()
	mock, ok := s.httpMocks[r.URL.Path]
	s.httpMocksMu.RUnlock()

	if ok {
		if s.opts.Verbose {
			s.logf("[http] serving mock response for %s %s\n", r.Method, r.URL.Path)
		}
		for k, v := range mock.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(mock.Status)
		_, _ = w.Write([]byte(mock.Body))
		return
	}

	// Fallback to UI or Status
	if s.opts.UIEnabled {
		// UI handler handles assets and fallback to index.html for React router
		ui.Handler().ServeHTTP(w, r)
		return
	}

	// Default: show status on root or 404
	if r.URL.Path == "/" || r.URL.Path == "" {
		s.serveStatus(w, r)
		return
	}

	http.NotFound(w, r)
}

// ListSSEStreams returns metadata for all SSE streams.
func (s *Server) ListSSEStreams() []SSEStreamInfo {
	if s.sseManager == nil {
		return nil
	}
	return s.sseManager.ListStreams()
}

// GetSSEStreamInfo returns metadata for a specific SSE stream.
func (s *Server) GetSSEStreamInfo(name string) (SSEStreamInfo, bool) {
	if s.sseManager == nil {
		return SSEStreamInfo{}, false
	}
	return s.sseManager.GetStreamInfo(name)
}

// ClearSSEBuffer clears the buffer of an SSE stream.
func (s *Server) ClearSSEBuffer(name string) error {
	if s.sseManager == nil {
		return fmt.Errorf("SSE manager not initialized")
	}
	return s.sseManager.ClearBuffer(name)
}

// GetStatus returns the current server status.
func (s *Server) GetStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == serverDraining {
		return "draining"
	}
	if s.paused.Load() {
		return "paused"
	}
	return "running"
}

// Drain stops accepting new connections.
func (s *Server) Drain() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = serverDraining
}

// IsDraining returns true if the server is in draining mode.
func (s *Server) IsDraining() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == serverDraining
}

// Pause suspends message processing.
func (s *Server) Pause() {
	s.paused.Store(true)
}

// Resume resumes message processing and flushes buffers.
func (s *Server) Resume() {
	s.paused.Store(false)
	s.pauseCond.Broadcast()
}

// IsPaused returns true if message processing is paused.
func (s *Server) IsPaused() bool {
	return s.paused.Load()
}

// WaitIfPaused blocks if the server is currently paused.
func (s *Server) WaitIfPaused() {
	if !s.paused.Load() {
		return
	}
	s.mu.Lock()
	for s.paused.Load() {
		s.pauseCond.Wait()
	}
	s.mu.Unlock()
}

// GetTemplateEngine returns the server's template engine.
func (s *Server) GetTemplateEngine() *template.Engine {
	return s.opts.TemplateEngine
}

// AddHandler adds a new message handler to the server at runtime.
// AddHandler registers a new message handler at runtime and starts it if it's a source.
func (s *Server) AddHandler(h handler.Handler) error {
	if err := s.registry.Add(h); err != nil {
		return err
	}
	s.startSourceHandler(h, context.Background())
	return nil
}

// UpdateHandler replaces an existing message handler at runtime.
func (s *Server) UpdateHandler(h handler.Handler) error {
	s.stopSourceHandler(h.Name)
	if err := s.registry.UpdateHandler(h); err != nil {
		return err
	}
	s.startSourceHandler(h, context.Background())
	return nil
}

// DeleteHandler removes a message handler from the server at runtime.
func (s *Server) DeleteHandler(name string) error {
	s.stopSourceHandler(name)
	return s.registry.Delete(name)
}

// RenameHandler renames a message handler at runtime.
func (s *Server) RenameHandler(oldName, newName string) error {
	s.sourceMu.Lock()
	if cancel, ok := s.sourceCancels[oldName]; ok {
		s.sourceCancels[newName] = cancel
		delete(s.sourceCancels, oldName)
	}
	s.sourceMu.Unlock()
	return s.registry.RenameHandler(oldName, newName)
}

// GetHandlers returns all currently registered handlers.
func (s *Server) GetHandlers() []handler.Handler {
	return s.registry.Handlers()
}

// GetVariables returns the current global variables used by handlers.
func (s *Server) GetVariables() map[string]interface{} {
	return s.opts.Variables
}

// ReloadHandlers replaces all handlers in the registry.
func (s *Server) ReloadHandlers(handlers []handler.Handler, variables map[string]interface{}) error {
	// Stop existing source handlers before replacing
	s.stopSourceHandlers()

	if err := s.registry.ReplaceHandlers(handlers); err != nil {
		return err
	}
	s.opts.Handlers = handlers
	s.opts.Variables = variables

	// Start new source handlers
	// We use background context as the server is already running
	s.startSourceHandlers(context.Background())

	return nil
}

// EnableHandler enables a handler by name.
func (s *Server) EnableHandler(name string) error {
	if err := s.registry.EnableHandler(name); err != nil {
		return err
	}
	if h, ok := s.registry.GetHandler(name); ok {
		s.startSourceHandler(h, context.Background())
	}
	return nil
}

// DisableHandler disables a handler by name.
func (s *Server) DisableHandler(name string) error {
	s.stopSourceHandler(name)
	return s.registry.DisableHandler(name)
}

// ResetSequence clears sequence indices for a handler.
func (s *Server) ResetSequence(name string) {
	s.registry.ResetSequence(name)
}

// ListKV returns all entries in the key-value store.
func (s *Server) ListKV() map[string]interface{} {
	if s.kvStore == nil {
		return make(map[string]interface{})
	}
	return s.kvStore.List()
}

// GetKV retrieves a value from the key-value store.
func (s *Server) GetKV(key string) (interface{}, bool) {
	if s.kvStore == nil {
		return nil, false
	}
	return s.kvStore.Get(key)
}

// SetKV stores a value in the key-value store.
func (s *Server) SetKV(key string, val interface{}, ttl time.Duration) {
	if s.kvStore == nil {
		s.kvStore = kv.NewStore()
	}
	s.kvStore.Set(key, val, ttl)
}

// DeleteKV removes a key from the key-value store.
func (s *Server) DeleteKV(key string) {
	if s.kvStore == nil {
		return
	}
	s.kvStore.Delete(key)
}

// GetHandlerStats returns statistics for a handler.
func (s *Server) GetHandlerStats(name string) (uint64, time.Duration, uint64, bool) {
	return s.registry.GetStats(name)
}

// GetGlobalStats returns global server statistics.
func (s *Server) GetGlobalStats() observability.GlobalStats {
	return observability.GetGlobalStats()
}

// GetRegistryStats returns global handler execution stats.
func (s *Server) GetRegistryStats() (uint64, uint64) {
	return s.registry.GetGlobalStats()
}

// GetSlowLog returns the slowest handler executions.
func (s *Server) GetSlowLog(limit int) []handler.SlowLogEntry {
	return s.registry.GetSlowLog(limit)
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

// GetPort returns the server's listening port.
func (s *Server) GetPort() int {
	return s.opts.Port
}

// GetPaths returns the server's WebSocket paths.
func (s *Server) GetPaths() []string {
	return s.opts.Paths
}

// GetURL returns a full WebSocket URL for a given path.
func (s *Server) GetURL(path string) string {
	scheme := "ws"
	if s.opts.TLSEnabled {
		scheme = "wss"
	}
	return fmt.Sprintf("%s://localhost:%d%s", scheme, s.opts.Port, path)
}

// StartStaticServe starts a new static file server on the given port.
func (s *Server) StartStaticServe(port int, root string, path string, isFile bool, generate bool, generateStyle string) error {
	if generate {
		wsURL := s.GetURL(s.opts.Paths[0])
		if err := s.staticMgr.GenerateMinimalHTML(root, wsURL, generateStyle); err != nil {
			return err
		}
	}

	return s.staticMgr.Start(StaticConfig{
		Port:  port,
		Root:  root,
		Path:  path,
		IsDir: !isFile,
	})
}

func (s *Server) startSourceHandlers(ctx context.Context) {
	for _, h := range s.registry.Handlers() {
		s.startSourceHandler(h, ctx)
	}
}

func (s *Server) startSourceHandler(h handler.Handler, ctx context.Context) {
	if (h.Builtin != "redis-subscribe" && h.Builtin != "mqtt-subscribe" && h.Builtin != "nats-subscribe") || s.registry.IsDisabled(h.Name) {
		return
	}

	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()

	// Already running?
	if _, ok := s.sourceCancels[h.Name]; ok {
		return
	}

	// Capture handler for closure
	handlerCopy := h
	sourceCtx, cancel := context.WithCancel(ctx)
	s.sourceCancels[h.Name] = cancel

	if h.Builtin == "redis-subscribe" {
		if s.redisMgr == nil {
			// Default to localhost if not configured
			mgr, err := handler.NewRedisManager("redis://localhost:6379")
			if err != nil {
				s.errorf("  [server] error initializing default redis for %q: %v\n", h.Name, err)
				cancel()
				delete(s.sourceCancels, h.Name)
				return
			}
			s.redisMgr = mgr
			s.logf("  [server] initialized default Redis: redis://localhost:6379\n")
		}
		go s.runRedisSubscribe(sourceCtx, &handlerCopy)
	} else if h.Builtin == "mqtt-subscribe" {
		go s.runMQTTSubscribe(sourceCtx, &handlerCopy)
	} else if h.Builtin == "nats-subscribe" {
		go s.runNATSSubscribe(sourceCtx, &handlerCopy)
	}
}

func (s *Server) stopSourceHandlers() {
	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()

	for name, cancel := range s.sourceCancels {
		cancel()
		delete(s.sourceCancels, name)
	}
}

func (s *Server) stopSourceHandler(name string) {
	s.sourceMu.Lock()
	defer s.sourceMu.Unlock()

	if cancel, ok := s.sourceCancels[name]; ok {
		cancel()
		delete(s.sourceCancels, name)
	}
}

func (s *Server) runRedisSubscribe(ctx context.Context, h *handler.Handler) {
	if s.opts.Verbose {
		s.logf("  [source:%s] starting redis-subscribe on %q\n", h.Name, h.Channel)
	}

	baseInterval := 2 * time.Second
	if h.ReconnectInterval != "" {
		if d, err := time.ParseDuration(h.ReconnectInterval); err == nil {
			baseInterval = d
		}
	}

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// (Re)connect and subscribe
		var pubsub *redis.PubSub
		isPattern := strings.ContainsAny(h.Channel, "*?")
		if isPattern {
			pubsub = s.redisMgr.PSubscribe(ctx, h.Channel)
		} else {
			pubsub = s.redisMgr.Subscribe(ctx, h.Channel)
		}

		if pubsub == nil {
			s.errorf("  [source:%s] failed to subscribe to %q (redis manager not initialized)\n", h.Name, h.Channel)
			return
		}

		// Wait for subscription confirmation
		_, err := pubsub.Receive(ctx)
		if err != nil {
			s.errorf("  [source:%s] subscription error for %q: %v\n", h.Name, h.Channel, err)
			
			// Trigger on_error if configured (only on first failure of a connection attempt)
			if attempt == 0 && h.OnErrorMsg != "" {
				s.broadcastSourceError(h)
			}

			// Backoff and retry
			attempt++
			wait := baseInterval * time.Duration(1<<(uint(attempt-1)))
			if wait > 30*time.Second {
				wait = 30 * time.Second
			}
			
			select {
			case <-ctx.Done():
				pubsub.Close()
				return
			case <-time.After(wait):
				pubsub.Close()
				continue
			}
		}

		if s.opts.Verbose {
			s.logf("  [source:%s] subscribed to %q\n", h.Name, h.Channel)
		}
		attempt = 0 // Reset on success

		// Message loop
	loop:
		for {
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				// Context cancellation is not an error we want to log as a disconnect
				if ctx.Err() != nil {
					pubsub.Close()
					return
				}
				break loop
			}
			s.handleRedisMessage(h, msg)
		}

		pubsub.Close()
		s.errorf("  [source:%s] redis connection lost for %q, reconnecting...\n", h.Name, h.Channel)
		// Trigger on_error on disconnect
		if h.OnErrorMsg != "" {
			s.broadcastSourceError(h)
		}

		// Exponential backoff
		attempt++
		wait := baseInterval * time.Duration(1<<(uint(attempt-1)))
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			continue
		}
	}
}

func (s *Server) handleRedisMessage(h *handler.Handler, msg *redis.Message) {
	if s.opts.Verbose {
		s.logf("  [source:%s] received redis message on %q: %s\n", h.Name, msg.Channel, msg.Payload)
	}

	tmplCtx := template.NewContext()
	tmplCtx.Message = msg.Payload
	tmplCtx.Channel = msg.Channel
	tmplCtx.Source = "redis"
	tmplCtx.Vars["Message"] = msg.Payload
	tmplCtx.Vars["Source"] = "redis"
	tmplCtx.Vars["Channel"] = msg.Channel
	
	// Add server context
	uptime := s.GetUptime()
	tmplCtx.Server = &template.ServerContext{
		ClientCount:     s.GetClientCount(),
		Clients:         s.GetClients(),
		Uptime:          uptime,
		UptimeFormatted: template.FormatUptime(uptime),
	}

	// 1. Transform message if respond: is set
	payload := msg.Payload
	if h.Respond != "" {
		res, err := s.opts.TemplateEngine.Execute("respond", h.Respond, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering respond template: %v\n", h.Name, err)
			return
		}
		payload = res
	}

	wsMsg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(payload),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 2. Deliver message
	if h.Topic != "" {
		topic, err := s.opts.TemplateEngine.Execute("topic", h.Topic, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering topic template: %v\n", h.Name, err)
			return
		}
		topic = strings.TrimSpace(topic)
		if topic != "" {
			delivered, _ := s.topics.Publish(topic, wsMsg)
			if s.opts.Verbose {
				s.logf("  [source:%s] delivered to topic %q (%d clients)\n", h.Name, topic, delivered)
			}
		}
	} else if h.Target != "" {
		target, err := s.opts.TemplateEngine.Execute("target", h.Target, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering target template: %v\n", h.Name, err)
			return
		}
		target = strings.TrimSpace(target)
		if target != "" {
			if err := s.Send(target, wsMsg); err != nil && s.opts.Verbose {
				s.errorf("  [source:%s] failed to send to client %q: %v\n", h.Name, target, err)
			}
		}
	} else {
		// Broadcast to all
		count := s.Broadcast(wsMsg)
		if s.opts.Verbose {
			s.logf("  [source:%s] broadcasted to %d clients\n", h.Name, count)
		}
	}
}

func (s *Server) runMQTTSubscribe(ctx context.Context, h *handler.Handler) {
	if s.opts.Verbose {
		s.logf("  [source:%s] starting mqtt-subscribe on %s topic %q\n", h.Name, h.BrokerURL, h.Topic)
	}

	baseInterval := 2 * time.Second
	if h.ReconnectInterval != "" {
		if d, err := time.ParseDuration(h.ReconnectInterval); err == nil {
			baseInterval = d
		}
	}

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		qos := byte(0)
		if h.QoS != "" {
			// Evaluate QoS with an empty context (since we don't have a message yet)
			res, err := s.opts.TemplateEngine.Execute("qos", h.QoS, template.NewContext())
			if err == nil {
				if val, err := strconv.Atoi(strings.TrimSpace(res)); err == nil {
					qos = byte(val)
				}
			}
		}

		unsubscribe, err := s.mqttMgr.Subscribe(h.BrokerURL, h.Topic, qos, func(topic string, payload []byte) {
			s.handleMQTTMessage(h, topic, payload)
		})

		if err == nil {
			// Subscription is active. Wait for context cancellation.
			<-ctx.Done()
			unsubscribe()
			return
		}

		s.errorf("  [source:%s] MQTT subscription error for %s: %v, retrying...\n", h.Name, h.BrokerURL, err)
		if h.OnErrorMsg != "" {
			s.broadcastSourceError(h)
		}

		// Exponential backoff
		attempt++
		wait := baseInterval * time.Duration(1<<(uint(attempt-1)))
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			continue
		}
	}
}

func (s *Server) handleMQTTMessage(h *handler.Handler, topic string, payload []byte) {
	if s.opts.Verbose {
		s.logf("  [source:%s] received MQTT message on %q: %s\n", h.Name, topic, string(payload))
	}

	tmplCtx := template.NewContext()
	tmplCtx.Message = string(payload)
	tmplCtx.MessageBytes = payload
	tmplCtx.MqttTopic = topic
	tmplCtx.Source = "mqtt"
	tmplCtx.Vars["Message"] = string(payload)
	tmplCtx.Vars["Source"] = "mqtt"
	tmplCtx.Vars["MqttTopic"] = topic

	// Add server context
	uptime := s.GetUptime()
	tmplCtx.Server = &template.ServerContext{
		ClientCount:     s.GetClientCount(),
		Clients:         s.GetClients(),
		Uptime:          uptime,
		UptimeFormatted: template.FormatUptime(uptime),
	}

	// 1. Transform message if respond: is set
	processedPayload := string(payload)
	if h.Respond != "" {
		res, err := s.opts.TemplateEngine.Execute("respond", h.Respond, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering respond template: %v\n", h.Name, err)
			return
		}
		processedPayload = res
	}

	wsMsg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(processedPayload),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 2. Deliver message
	if h.Topic != "" && h.Builtin != "mqtt-subscribe" {
		// This branch handles cases where a target topic is specified for the WS broadcast
		// But for mqtt-subscribe, h.Topic is the SOURCE topic.
		// So we only use h.Topic as a TARGET if it's explicitly intended.
		// Wait, in redis-subscribe, h.Topic is used as TARGET topic if present.
		// But h.Channel is the source.
		// For mqtt-subscribe, h.Topic is BOTH the source and potentially the target.
		// To avoid confusion, let's say if h.Topic is set, we broadcast to that WS topic.
		
		topicName, err := s.opts.TemplateEngine.Execute("topic", h.Topic, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering topic template: %v\n", h.Name, err)
			return
		}
		topicName = strings.TrimSpace(topicName)
		if topicName != "" {
			delivered, _ := s.topics.Publish(topicName, wsMsg)
			if s.opts.Verbose {
				s.logf("  [source:%s] delivered to topic %q (%d clients)\n", h.Name, topicName, delivered)
			}
		}
	} else if h.Target != "" {
		target, err := s.opts.TemplateEngine.Execute("target", h.Target, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering target template: %v\n", h.Name, err)
			return
		}
		target = strings.TrimSpace(target)
		if target != "" {
			if err := s.Send(target, wsMsg); err != nil && s.opts.Verbose {
				s.errorf("  [source:%s] failed to send to client %q: %v\n", h.Name, target, err)
			}
		}
	} else {
		// Broadcast to all
		count := s.Broadcast(wsMsg)
		if s.opts.Verbose {
			s.logf("  [source:%s] broadcasted to %d clients\n", h.Name, count)
		}
	}
}

func (s *Server) runNATSSubscribe(ctx context.Context, h *handler.Handler) {
	if s.opts.Verbose {
		s.logf("  [source:%s] starting nats-subscribe on %s subject %q\n", h.Name, h.NatsURL, h.Subject)
	}

	baseInterval := 2 * time.Second
	if h.ReconnectInterval != "" {
		if d, err := time.ParseDuration(h.ReconnectInterval); err == nil {
			baseInterval = d
		}
	}

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		unsubscribe, err := s.natsMgr.Subscribe(h.NatsURL, h.Subject, func(subject string, payload []byte) {
			s.handleNATSMessage(h, subject, payload)
		})

		if err == nil {
			// Subscription is active. Wait for context cancellation.
			if s.opts.Verbose {
				s.logf("  [source:%s] nats-subscribe active on %s\n", h.Name, h.Subject)
			}
			<-ctx.Done()
			unsubscribe()
			return
		}

		s.errorf("  [source:%s] NATS subscription error for %s: %v, retrying...\n", h.Name, h.NatsURL, err)
		if h.OnErrorMsg != "" {
			s.broadcastSourceError(h)
		}

		// Exponential backoff
		attempt++
		wait := baseInterval * time.Duration(1<<(uint(attempt-1)))
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			continue
		}
	}
}

func (s *Server) handleNATSMessage(h *handler.Handler, subject string, payload []byte) {
	if s.opts.Verbose {
		s.logf("  [source:%s] received NATS message on %q: %s\n", h.Name, subject, string(payload))
	}

	tmplCtx := template.NewContext()
	tmplCtx.Message = string(payload)
	tmplCtx.MessageBytes = payload
	tmplCtx.NatsSubject = subject
	tmplCtx.Source = "nats"
	tmplCtx.Vars["Message"] = string(payload)
	tmplCtx.Vars["Source"] = "nats"
	tmplCtx.Vars["NatsSubject"] = subject

	// Add server context
	uptime := s.GetUptime()
	tmplCtx.Server = &template.ServerContext{
		ClientCount:     s.GetClientCount(),
		Clients:         s.GetClients(),
		Uptime:          uptime,
		UptimeFormatted: template.FormatUptime(uptime),
	}

	// 1. Transform message if respond: is set
	processedPayload := string(payload)
	if h.Respond != "" {
		res, err := s.opts.TemplateEngine.Execute("respond", h.Respond, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering respond template: %v\n", h.Name, err)
			return
		}
		processedPayload = res
	}

	wsMsg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(processedPayload),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 2. Deliver message
	if h.Topic != "" && h.Builtin != "nats-subscribe" {
		topicName, err := s.opts.TemplateEngine.Execute("topic", h.Topic, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering topic template: %v\n", h.Name, err)
			return
		}
		topicName = strings.TrimSpace(topicName)
		if topicName != "" {
			delivered, _ := s.topics.Publish(topicName, wsMsg)
			if s.opts.Verbose {
				s.logf("  [source:%s] delivered to topic %q (%d clients)\n", h.Name, topicName, delivered)
			}
		}
	} else if h.Target != "" {
		target, err := s.opts.TemplateEngine.Execute("target", h.Target, tmplCtx)
		if err != nil {
			s.errorf("  [source:%s] error rendering target template: %v\n", h.Name, err)
			return
		}
		target = strings.TrimSpace(target)
		if target != "" {
			if err := s.Send(target, wsMsg); err != nil && s.opts.Verbose {
				s.errorf("  [source:%s] failed to send to client %q: %v\n", h.Name, target, err)
			}
		}
	} else {
		// Broadcast to all
		count := s.Broadcast(wsMsg)
		if s.opts.Verbose {
			s.logf("  [source:%s] broadcasted to %d clients\n", h.Name, count)
		}
	}
}

func (s *Server) broadcastSourceError(h *handler.Handler) {
	tmplCtx := template.NewContext()
	tmplCtx.Source = "redis"
	tmplCtx.Channel = h.Channel
	tmplCtx.Vars["Source"] = "redis"
	tmplCtx.Vars["Channel"] = h.Channel

	res, err := s.opts.TemplateEngine.Execute("on_error", h.OnErrorMsg, tmplCtx)
	if err != nil {
		s.errorf("  [source:%s] error rendering on_error template: %v\n", h.Name, err)
		return
	}

	if res == "" {
		return
	}

	wsMsg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(res),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}
	s.Broadcast(wsMsg)
}

// StopStaticServe stops the static server on the given port.
func (s *Server) StopStaticServe(port int) error {
	return s.staticMgr.Stop(port)
}

// GetStaticConfigs returns all active static server configurations.
func (s *Server) GetStaticConfigs() []map[string]interface{} {
	configs := s.staticMgr.GetConfigs()
	maps := make([]map[string]interface{}, 0, len(configs))
	for _, c := range configs {
		maps = append(maps, map[string]interface{}{
			"port":     c.Port,
			"root":     c.Root,
			"path":     c.Path,
			"isDir":    c.IsDir,
			"requests": c.Requests,
		})
	}
	return maps
}

// GenerateMinimalHTML creates a boilerplate HTML client.
func (s *Server) GenerateMinimalHTML(targetPath string, wsURL string, style string) error {
	return s.staticMgr.GenerateMinimalHTML(targetPath, wsURL, style)
}
func (s *Server) GetAvailableStyles() []string {
	return GetAvailableStyles()
}
