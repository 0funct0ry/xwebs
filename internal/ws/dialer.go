package ws

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"
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

	if u.Scheme == "wss" {
		if dOpts.TLSConfig == nil {
			dOpts.TLSConfig = &tls.Config{}
		}

		// Load custom CA if provided
		if dOpts.CACert != "" {
			caCert, err := os.ReadFile(dOpts.CACert)
			if err != nil {
				return nil, fmt.Errorf("reading CA certificate %q: %w", dOpts.CACert, err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate %q: invalid PEM", dOpts.CACert)
			}
			dOpts.TLSConfig.RootCAs = caCertPool
		}

		// Load client certificates if provided
		if dOpts.ClientCert != "" && dOpts.ClientKey != "" {
			cert, err := tls.LoadX509KeyPair(dOpts.ClientCert, dOpts.ClientKey)
			if err != nil {
				return nil, fmt.Errorf("loading client key pair: %w", err)
			}
			dOpts.TLSConfig.Certificates = []tls.Certificate{cert}
		} else if dOpts.ClientCert != "" || dOpts.ClientKey != "" {
			return nil, fmt.Errorf("both client certificate and key must be provided for mTLS")
		}
	}

	dialer := websocket.Dialer{
		Subprotocols:    dOpts.Subprotocols,
		TLSClientConfig: dOpts.TLSConfig,
	}

	if dOpts.ProxyURL != "" {
		proxyURL, err := url.Parse(dOpts.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("parsing proxy URL %q: %w", dOpts.ProxyURL, err)
		}

		switch proxyURL.Scheme {
		case "http", "https":
			dialer.Proxy = http.ProxyURL(proxyURL)
		case "socks5":
			pDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("creating SOCKS5 dialer: %w", err)
			}
			dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				if cd, ok := pDialer.(proxy.ContextDialer); ok {
					return cd.DialContext(ctx, network, addr)
				}
				return pDialer.Dial(network, addr)
			}
		default:
			return nil, fmt.Errorf("unsupported proxy scheme %q", proxyURL.Scheme)
		}
	}

	conn, resp, err := dialer.DialContext(ctx, urlStr, dOpts.Headers)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket handshake failed with status %d: %w", resp.StatusCode, err)
		}
		// Wrap TLS errors for better clarity
		if strings.Contains(err.Error(), "certificate") || strings.Contains(err.Error(), "tls:") {
			return nil, fmt.Errorf("TLS verification failed: %w", err)
		}
		return nil, fmt.Errorf("failed to dial %q: %w", urlStr, err)
	}

	return NewConnection(conn, urlStr, resp, dOpts), nil
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
