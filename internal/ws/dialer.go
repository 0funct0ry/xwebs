package ws

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

// DefaultDialer is the default WebSocket dialer.
type Dialer struct {
	Options *DialOptions
}

// NewDialer creates a new WebSocket dialer with the given options.
func NewDialer(options *DialOptions) *Dialer {
	return &Dialer{
		Options: options,
	}
}

// Dial connects to the given WebSocket URL with the configured options.
func Dial(ctx context.Context, urlStr string, opts ...DialOption) (*Connection, error) {
	dOpts := &DialOptions{
		Headers: make(http.Header),
	}
	for _, opt := range opts {
		opt(dOpts)
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", urlStr, err)
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return nil, fmt.Errorf("invalid scheme %q; only ws:// and wss:// are supported", u.Scheme)
	}

	dialer := websocket.Dialer{
		Subprotocols:     dOpts.Subprotocols,
		TLSClientConfig: dOpts.TLSConfig,
	}

	conn, resp, err := dialer.DialContext(ctx, urlStr, dOpts.Headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket handshake failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("failed to dial %q: %w", urlStr, err)
	}

	return NewConnection(conn, urlStr, resp), nil
}

// ParseURL parses and validates a WebSocket URL.
func ParseURL(urlStr string) (*url.URL, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	if !strings.HasPrefix(u.Scheme, "ws") {
		return nil, fmt.Errorf("invalid scheme: must be ws:// or wss://")
	}

	return u, nil
}
