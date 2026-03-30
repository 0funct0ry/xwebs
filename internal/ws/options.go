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
	PingInterval       time.Duration
	PongWait           time.Duration
	Reconnect          bool
	ReconnectBackoff   time.Duration
	ReconnectMax       time.Duration
	ReconnectAttempts  int
	Verbose            bool
	MaxMessageSize     int64
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

// WithReconnect enables automatic reconnection.
func WithReconnect(reconnect bool) DialOption {
	return func(o *DialOptions) {
		o.Reconnect = reconnect
	}
}

// WithReconnectBackoff sets the initial backoff duration for reconnection.
func WithReconnectBackoff(backoff time.Duration) DialOption {
	return func(o *DialOptions) {
		o.ReconnectBackoff = backoff
	}
}

// WithReconnectMax sets the maximum backoff duration for reconnection.
func WithReconnectMax(max time.Duration) DialOption {
	return func(o *DialOptions) {
		o.ReconnectMax = max
	}
}

// WithReconnectAttempts sets the maximum number of reconnection attempts.
func WithReconnectAttempts(attempts int) DialOption {
	return func(o *DialOptions) {
		o.ReconnectAttempts = attempts
	}
}

// WithMaxMessageSize sets the maximum message size for incoming and outgoing messages.
func WithMaxMessageSize(size int64) DialOption {
	return func(o *DialOptions) {
		o.MaxMessageSize = size
	}
}
