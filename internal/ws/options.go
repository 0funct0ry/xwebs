package ws

import (
	"crypto/tls"
	"net/http"
	"time"
)

// DialOptions configuration for the WebSocket dialer.
type DialOptions struct {
	Headers      http.Header
	Subprotocols []string
	TLSConfig    *tls.Config
	CACert       string
	ClientCert   string
	ClientKey    string
	ProxyURL        string
	ReadBufferSize  int
	WriteBufferSize int
	PingInterval    time.Duration
	PongWait        time.Duration
	Verbose         bool
}

// DialOption is a functional option for the Dial function.
type DialOption func(*DialOptions)

// WithHeaders sets custom HTTP headers for the handshake.
func WithHeaders(headers http.Header) DialOption {
	return func(o *DialOptions) {
		if o.Headers == nil {
			o.Headers = make(http.Header)
		}
		for k, v := range headers {
			o.Headers[k] = v
		}
	}
}

// WithSubprotocols sets suggested subprotocols for the negotiation and selection.
func WithSubprotocols(protocols ...string) DialOption {
	return func(o *DialOptions) {
		o.Subprotocols = append(o.Subprotocols, protocols...)
	}
}

// WithInsecureSkipVerify sets the TLS configuration to skip verification.
func WithInsecureSkipVerify(insecure bool) DialOption {
	return func(o *DialOptions) {
		if o.TLSConfig == nil {
			o.TLSConfig = &tls.Config{}
		}
		o.TLSConfig.InsecureSkipVerify = insecure
	}
}

// WithCACert sets the CA certificate for server verification.
func WithCACert(caPath string) DialOption {
	return func(o *DialOptions) {
		o.CACert = caPath
	}
}

// WithClientCert sets the client certificate and key for mTLS.
func WithClientCert(certPath, keyPath string) DialOption {
	return func(o *DialOptions) {
		o.ClientCert = certPath
		o.ClientKey = keyPath
	}
}

// WithProxy sets the proxy URL for the connection.
func WithProxy(proxyURL string) DialOption {
	return func(o *DialOptions) {
		o.ProxyURL = proxyURL
	}
}
// WithReadBufferSize sets the buffer size for the read channel.
func WithReadBufferSize(size int) DialOption {
	return func(o *DialOptions) {
		o.ReadBufferSize = size
	}
}

// WithWriteBufferSize sets the buffer size for the write channel.
func WithWriteBufferSize(size int) DialOption {
	return func(o *DialOptions) {
		o.WriteBufferSize = size
	}
}

// WithPingInterval sets the interval for automatic ping messages.
func WithPingInterval(interval time.Duration) DialOption {
	return func(o *DialOptions) {
		o.PingInterval = interval
	}
}

// WithPongWait sets the wait time for a pong response from the server.
func WithPongWait(wait time.Duration) DialOption {
	return func(o *DialOptions) {
		o.PongWait = wait
	}
}

// WithVerbose enables verbose logging in the WebSocket engine.
func WithVerbose(verbose bool) DialOption {
	return func(o *DialOptions) {
		o.Verbose = verbose
	}
}
