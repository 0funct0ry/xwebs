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

	// Bind address
	BindAddr string

	APIEnabled     bool
	MetricsEnabled bool
	UIEnabled      bool

	// Server timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// TLS configuration
	TLSEnabled bool
	CertFile   string
	KeyFile    string

	// Security configuration
	AllowedOrigins []string
	AllowIPs       []string
	DenyIPs        []string
	RateLimit      string

	// Logger for server events
	Logger Logger

	// Static file serving
	StaticServeDir      string
	StaticServeFile     string
	StaticServePath     string
	StaticServePort     int
	StaticServeAddr     string
	StaticGenerate      bool
	StaticGenerateStyle string
	SSEStreams          []handler.SSEStreamConfig

	RedisManager handler.RedisManager
	MQTTManager  handler.MQTTManager
	NATSManager  handler.NATSManager
	KafkaManager handler.KafkaManager
	SQLiteManager handler.SQLiteManager
	OllamaURL    string
}

// Logger defines the interface for server logging.
type Logger interface {
	Printf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

// Option is a functional option for configuring the server.
type Option func(*Options)

// DefaultOptions returns the default server configuration.
func DefaultOptions() *Options {
	return &Options{
		BindAddr:     "localhost",
		Port:         8080,
		Paths:        []string{"/"},
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		APIEnabled:   true,
	}
}

// WithBindAddr sets the listening bind address.
func WithBindAddr(addr string) Option {
	return func(o *Options) {
		o.BindAddr = addr
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

// WithAllowedOrigins sets the allowed origins for WebSocket connections.
func WithAllowedOrigins(origins []string) Option {
	return func(o *Options) {
		o.AllowedOrigins = origins
	}
}

// WithAllowIPs sets the allowed IP addresses or CIDR ranges.
func WithAllowIPs(ips []string) Option {
	return func(o *Options) {
		o.AllowIPs = ips
	}
}

// WithDenyIPs sets the denied IP addresses or CIDR ranges.
func WithDenyIPs(ips []string) Option {
	return func(o *Options) {
		o.DenyIPs = ips
	}
}

// WithRateLimit sets the rate limit configuration.
func WithRateLimit(rateLimit string) Option {
	return func(o *Options) {
		o.RateLimit = rateLimit
	}
}

// WithUI enables or disables the web UI.
func WithUI(enabled bool) Option {
	return func(o *Options) {
		o.UIEnabled = enabled
	}
}

// WithLogger sets the server logger.
func WithLogger(logger Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithStaticServeDir sets the directory to serve static files from.
func WithStaticServeDir(dir string) Option {
	return func(o *Options) {
		o.StaticServeDir = dir
	}
}

// WithStaticServeFile sets the single file to serve static content from.
func WithStaticServeFile(file string) Option {
	return func(o *Options) {
		o.StaticServeFile = file
	}
}

// WithStaticServePath sets the URL path prefix for static content.
func WithStaticServePath(path string) Option {
	return func(o *Options) {
		o.StaticServePath = path
	}
}

// WithStaticServePort sets the port to listen on for static content.
func WithStaticServePort(port int) Option {
	return func(o *Options) {
		o.StaticServePort = port
	}
}

// WithStaticServeAddr sets the bind address to listen on for static content.
func WithStaticServeAddr(addr string) Option {
	return func(o *Options) {
		o.StaticServeAddr = addr
	}
}

// WithStaticGenerate sets whether to generate a high-quality HTML client if the file doesn't exist.
func WithStaticGenerate(generate bool) Option {
	return func(o *Options) {
		o.StaticGenerate = generate
	}
}

// WithStaticGenerateStyle sets the style for the generated HTML client.
func WithStaticGenerateStyle(style string) Option {
	return func(o *Options) {
		o.StaticGenerateStyle = style
	}
}

// WithRedisManager sets the Redis manager.
func WithRedisManager(m handler.RedisManager) Option {
	return func(o *Options) {
		o.RedisManager = m
	}
}

// WithMQTTManager sets the MQTT manager.
func WithMQTTManager(m handler.MQTTManager) Option {
	return func(o *Options) {
		o.MQTTManager = m
	}
}

// WithSSEStreams sets the SSE stream configurations.
func WithSSEStreams(streams []handler.SSEStreamConfig) Option {
	return func(o *Options) {
		o.SSEStreams = streams
	}
}

// WithNATSManager sets the NATS manager.
func WithNATSManager(m handler.NATSManager) Option {
	return func(o *Options) {
		o.NATSManager = m
	}
}

// WithKafkaManager sets the Kafka manager.
func WithKafkaManager(m handler.KafkaManager) Option {
	return func(o *Options) {
		o.KafkaManager = m
	}
}

// WithOllamaURL sets the default Ollama API URL.
func WithOllamaURL(url string) Option {
	return func(o *Options) {
		o.OllamaURL = url
	}
}

// WithSQLiteManager sets the SQLite manager.
func WithSQLiteManager(m handler.SQLiteManager) Option {
	return func(o *Options) {
		o.SQLiteManager = m
	}
}
