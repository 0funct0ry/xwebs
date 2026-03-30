package ws

import (
	"crypto/tls"
	"net/http"
)

// DialOptions configuration for the WebSocket dialer.
type DialOptions struct {
	Headers      http.Header
	Subprotocols []string
	TLSConfig    *tls.Config
	CACert       string
	ClientCert   string
	ClientKey    string
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
