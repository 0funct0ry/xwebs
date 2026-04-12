package server

import (
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
)

// Options defines the configuration for the WebSocket server.
type Options struct {
	Port           int
	Paths          []string
	Handlers       []handler.Handler
	Variables      map[string]interface{}
	TemplateEngine *template.Engine
	Verbose        bool
	Sandbox        bool
	Allowlist      []string
	
	APIEnabled     bool
	MetricsEnabled bool

	// Server timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// TLS configuration
	TLSEnabled bool
	CertFile   string
	KeyFile    string
}

// Option is a functional option for configuring the server.
type Option func(*Options)

// DefaultOptions returns the default server configuration.
func DefaultOptions() *Options {
	return &Options{
		Port:         8080,
		Paths:        []string{"/"},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		APIEnabled:   true,
	}
}

// WithPort sets the listening port.
func WithPort(port int) Option {
	return func(o *Options) {
		o.Port = port
	}
}

// WithPaths sets the WebSocket paths.
func WithPaths(paths []string) Option {
	return func(o *Options) {
		if len(paths) > 0 {
			o.Paths = paths
		}
	}
}

// WithHandlers sets the message handlers.
func WithHandlers(handlers []handler.Handler) Option {
	return func(o *Options) {
		o.Handlers = handlers
	}
}

// WithVariables sets the global variables for handlers.
func WithVariables(vars map[string]interface{}) Option {
	return func(o *Options) {
		o.Variables = vars
	}
}

// WithTemplateEngine sets the template engine.
func WithTemplateEngine(engine *template.Engine) Option {
	return func(o *Options) {
		o.TemplateEngine = engine
	}
}

// WithVerbose enables or disables verbose logging.
func WithVerbose(verbose bool) Option {
	return func(o *Options) {
		o.Verbose = verbose
	}
}

// WithSandbox sets the sandbox mode for handlers.
func WithSandbox(sandbox bool) Option {
	return func(o *Options) {
		o.Sandbox = sandbox
	}
}

// WithAllowlist sets the allowed shell commands for handlers.
func WithAllowlist(allowlist []string) Option {
	return func(o *Options) {
		o.Allowlist = allowlist
	}
}

// WithTLS enables or disables TLS.
func WithTLS(enabled bool) Option {
	return func(o *Options) {
		o.TLSEnabled = enabled
	}
}

// WithCertFile sets the path to the certificate file.
func WithCertFile(certFile string) Option {
	return func(o *Options) {
		o.CertFile = certFile
	}
}

// WithKeyFile sets the path to the private key file.
func WithKeyFile(keyFile string) Option {
	return func(o *Options) {
		o.KeyFile = keyFile
	}
}

// WithAPI enables or disables the REST API.
func WithAPI(enabled bool) Option {
	return func(o *Options) {
		o.APIEnabled = enabled
	}
}

// WithMetrics enables or disables the metrics endpoint.
func WithMetrics(enabled bool) Option {
	return func(o *Options) {
		o.MetricsEnabled = enabled
	}
}
