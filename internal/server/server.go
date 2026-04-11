package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/ws"
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
	wg          sync.WaitGroup
}

// New creates a new WebSocket server with the given options.
func New(opts ...Option) *Server {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	s := &Server{
		opts: options,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Default: allow all origins
			},
		},
		connections: make(map[string]*ws.Connection),
	}

	if len(options.Handlers) > 0 {
		s.registry = handler.NewRegistry()
		s.registry.AddHandlers(options.Handlers)
	}

	return s
}

// Start launches the server and listens for incoming connections.
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	for _, path := range s.opts.Paths {
		mux.HandleFunc(path, s.serveWS)
	}

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.opts.Port),
		Handler:      mux,
		ReadTimeout:  s.opts.ReadTimeout,
		WriteTimeout: s.opts.WriteTimeout,
		IdleTimeout:  s.opts.IdleTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		if s.opts.Verbose {
			fmt.Printf("Starting server on %s\n", s.httpSrv.Addr)
		}
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
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
			fmt.Printf("Closing connection %s\n", id)
		}
		_ = conn.Close()
	}
	s.mu.Unlock()

	// Wait for all connections to finish cleaning up
	s.wg.Wait()

	return err
}

func (s *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		if s.opts.Verbose {
			fmt.Printf("Upgrade error: %v\n", err)
		}
		return
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

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.connections, wsConn.ID)
			s.mu.Unlock()
		}()

		// Start message loops
		wsConn.Start()

		// Initialize dispatcher if registry exists
		if s.registry != nil {
			dispatcher := handler.NewDispatcher(
				s.registry,
				wsConn,
				s.opts.TemplateEngine,
				s.opts.Verbose,
				s.opts.Variables,
				nil, // sessionVars
				s.opts.Sandbox,
				s.opts.Allowlist,
			)
			
			// Setup logging/error handlers for dispatcher if needed
			if s.opts.Verbose {
				dispatcher.Log = func(f string, a ...interface{}) {
					fmt.Printf("[handler] "+f+"\n", a...)
				}
				dispatcher.Error = func(f string, a ...interface{}) {
					fmt.Printf("[handler-error] "+f+"\n", a...)
				}
			}

			dispatcher.Start(context.Background())
			dispatcher.HandleConnect()
			defer dispatcher.HandleDisconnect() // Handle disconnect when read loop exits
		}

		// Wait for connection to close
		<-wsConn.Done()
	}()
}
