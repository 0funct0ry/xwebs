package relay

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/gorilla/websocket"
)

// Relay handles bi-directional WebSocket proxying.
type Relay struct {
	opts      *Options
	httpSrv   *http.Server
	upgrader  websocket.Upgrader
	startTime time.Time
}

// Options defines configuration for the Relay server.
type Options struct {
	BindAddr     string
	Port         int
	UpstreamURL  string
	Paths        []string
	Verbose      bool
	Quiet        bool

	// TLS configuration for the listener
	TLSEnabled bool
	CertFile   string
	KeyFile    string

	// Dialer options for the upstream connection
	DialOpts []ws.DialOption

	// Logger for relay events
	Logger Logger
}

// Logger defines the interface for relay logging.
type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// New creates a new Relay instance.
func New(opts *Options) *Relay {
	if opts.BindAddr == "" {
		opts.BindAddr = "localhost"
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
	if len(opts.Paths) == 0 {
		opts.Paths = []string{"/"}
	}

	r := &Relay{
		opts:      opts,
		startTime: time.Now(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Default: allow all origins for relay
			},
		},
	}

	return r
}

// Start launches the relay server.
func (r *Relay) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	for _, path := range r.opts.Paths {
		pattern := path
		if path == "/" {
			pattern = "/{$}" // Exact match for root in Go 1.22+
		}
		mux.HandleFunc(pattern, r.serveRelay)
	}

	r.httpSrv = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", r.opts.BindAddr, r.opts.Port),
		Handler: mux,
	}

	errChan := make(chan error, 1)
	go func() {
		if r.opts.TLSEnabled {
			if r.opts.Verbose {
				r.logf("Starting TLS relay on %s -> %s\n", r.httpSrv.Addr, r.opts.UpstreamURL)
			}
			if err := r.httpSrv.ListenAndServeTLS(r.opts.CertFile, r.opts.KeyFile); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		} else {
			if r.opts.Verbose {
				r.logf("Starting relay on %s -> %s\n", r.httpSrv.Addr, r.opts.UpstreamURL)
			}
			if err := r.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}
	}()

	select {
	case <-ctx.Done():
		return r.Stop()
	case err := <-errChan:
		return err
	}
}

// Stop gracefully shuts down the relay server.
func (r *Relay) Stop() error {
	if r.httpSrv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.httpSrv.Shutdown(ctx)
}

func (r *Relay) serveRelay(w http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Upgrade") != "websocket" {
		http.Error(w, "Not a WebSocket handshake", http.StatusBadRequest)
		return
	}

	// Dial upstream
	if r.opts.Verbose {
		r.logf("[relay] incoming connection from %s, dialing upstream %s\n", req.RemoteAddr, r.opts.UpstreamURL)
	}

	// For the upstream dial, we might want to pass headers from the incoming request
	// But usually relay means "start a fresh session" or "mirror".
	// The US mentioned standard flags like --header, which are usually for the upstream.
	
	upstreamConn, err := ws.Dial(req.Context(), r.opts.UpstreamURL, r.opts.DialOpts...)
	if err != nil {
		r.errorf("[relay] failed to connect to upstream %s: %v\n", r.opts.UpstreamURL, err)
		http.Error(w, "Failed to connect to upstream", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	// Upgrade client connection
	clientConnRaw, err := r.upgrader.Upgrade(w, req, nil)
	if err != nil {
		r.errorf("[relay] failed to upgrade client connection: %v\n", err)
		return
	}
	
	// Wrap client connection
	clientConn := ws.NewConnection(clientConnRaw, req.URL.String(), nil, &ws.DialOptions{
		Verbose: r.opts.Verbose,
	})
	clientConn.Start()
	defer clientConn.Close()

	if r.opts.Verbose {
		r.logf("[relay] piping %s <-> %s\n", req.RemoteAddr, r.opts.UpstreamURL)
	}

	// Pipe messages
	errChan := make(chan error, 2)

	// Client to Upstream
	go func() {
		msgCh := clientConn.Subscribe()
		defer clientConn.Unsubscribe(msgCh)
		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					errChan <- nil
					return
				}
				if msg.Metadata.Direction == "received" {
					if r.opts.Verbose {
						r.logf("[relay] CLIENT -> UPSTREAM: %d bytes (%s)\n", len(msg.Data), msg.Metadata.ID)
					}
					if err := upstreamConn.Write(msg); err != nil {
						errChan <- fmt.Errorf("writing to upstream: %w", err)
						return
					}
				}
			case <-clientConn.Done():
				errChan <- nil
				return
			case <-upstreamConn.Done():
				errChan <- nil
				return
			}
		}
	}()

	// Upstream to Client
	go func() {
		msgCh := upstreamConn.Subscribe()
		defer upstreamConn.Unsubscribe(msgCh)
		for {
			select {
			case msg, ok := <-msgCh:
				if !ok {
					errChan <- nil
					return
				}
				if msg.Metadata.Direction == "received" {
					if r.opts.Verbose {
						r.logf("[relay] UPSTREAM -> CLIENT: %d bytes (%s)\n", len(msg.Data), msg.Metadata.ID)
					}
					if err := clientConn.Write(msg); err != nil {
						errChan <- fmt.Errorf("writing to client: %w", err)
						return
					}
				}
			case <-upstreamConn.Done():
				errChan <- nil
				return
			case <-clientConn.Done():
				errChan <- nil
				return
			}
		}
	}()

	// Wait for any side to close or error
	err = <-errChan
	if err != nil && r.opts.Verbose {
		r.errorf("[relay] connection closed with error: %v\n", err)
	} else if r.opts.Verbose {
		r.logf("[relay] connection closed normally\n")
	}
}

func (r *Relay) logf(format string, v ...interface{}) {
	if r.opts.Logger != nil {
		r.opts.Logger.Printf(format, v...)
	} else {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}

func (r *Relay) errorf(format string, v ...interface{}) {
	if r.opts.Logger != nil {
		r.opts.Logger.Errorf(format, v...)
	} else {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}
