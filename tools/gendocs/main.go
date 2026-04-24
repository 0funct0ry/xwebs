// gendocs generates Docusaurus MDX reference pages for xwebs CLI commands.
// Since the cobra root command is unexported, this tool generates docs from
// a manually-maintained command manifest. Run: go run tools/gendocs/main.go --out docs-site/docs/reference/
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var outDir string
	flag.StringVar(&outDir, "out", "docs-site/docs/reference", "Output directory for generated MDX files")
	flag.Parse()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for _, cmd := range commands {
		if err := writeDoc(cmd, outDir); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", cmd.Name, err)
			os.Exit(1)
		}
	}
	fmt.Printf("Generated %d command docs in %s\n", len(commands), outDir)
}

type Flag struct {
	Name    string
	Type    string
	Default string
	Desc    string
}

type Command struct {
	Name     string
	Short    string
	Long     string
	Use      string
	Flags    []Flag
	Examples []string
}

func writeDoc(c Command, dir string) error {
	outFile := filepath.Join(dir, "cmd-"+c.Name+".md")
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	generated := time.Now().Format("2006-01-02")

	// Front matter
	fmt.Fprintf(f, "---\ntitle: \"xwebs %s\"\ndescription: \"%s\"\ngenerated: \"%s\"\n---\n\n",
		c.Name, c.Short, generated)

	// Header
	fmt.Fprintf(f, "# xwebs %s\n\n%s\n\n---\n\n", c.Name, c.Long)

	// Synopsis
	fmt.Fprintf(f, "## Synopsis\n\n```bash\n%s\n```\n\n", c.Use)

	// Flags
	if len(c.Flags) > 0 {
		fmt.Fprintf(f, "## Flags\n\n| Flag | Type | Default | Description |\n|------|------|---------|-------------|\n")
		for _, fl := range c.Flags {
			def := fl.Default
			if def == "" {
				def = "—"
			}
			fmt.Fprintf(f, "| `--%s` | %s | %s | %s |\n", fl.Name, fl.Type, def, fl.Desc)
		}
		fmt.Fprintln(f)
	}

	// Examples
	if len(c.Examples) > 0 {
		fmt.Fprintf(f, "## Examples\n\n```bash\n%s\n```\n\n", strings.Join(c.Examples, "\n\n"))
	}

	return nil
}

var commands = []Command{
	{
		Name:  "connect",
		Short: "Connect to a WebSocket server",
		Long:  "Connect to a remote WebSocket server in interactive (REPL) or non-interactive (pipeline) mode. In interactive mode a readline-based REPL is started with tab completion, persistent history, and syntax highlighting. In non-interactive mode (piped stdin or --once/--script flags) xwebs reads from stdin, sends messages, and writes responses to stdout.",
		Use:   "xwebs connect <url> [flags]",
		Flags: []Flag{
			{"header", "stringArray", "", "HTTP header to include in the handshake (repeatable)"},
			{"subprotocol", "stringArray", "", "WebSocket subprotocol(s) to negotiate"},
			{"cert", "string", "", "TLS client certificate file (PEM)"},
			{"key", "string", "", "TLS client key file (PEM)"},
			{"ca", "string", "", "CA certificate file for server verification"},
			{"insecure", "bool", "false", "Skip TLS certificate verification"},
			{"compress", "bool", "false", "Enable permessage-deflate compression"},
			{"max-message-size", "string", "16MB", "Maximum incoming message size"},
			{"ping-interval", "duration", "30s", "Keepalive ping interval"},
			{"reconnect", "bool", "true", "Automatically reconnect on disconnect"},
			{"reconnect-max", "duration", "5m", "Maximum reconnect backoff duration"},
			{"interactive", "bool", "auto", "Force interactive REPL (even without TTY)"},
			{"keymap", "string", "emacs", "REPL keymap: emacs or vi"},
			{"history-size", "int", "10000", "Maximum REPL history entries"},
			{"proxy", "string", "", "Proxy URL (http://, https://, socks5://)"},
			{"verbose", "bool", "false", "Verbose output"},
			{"quiet", "bool", "false", "Suppress non-essential output"},
			{"color", "string", "auto", "Color mode: auto, on, off"},
			{"log-level", "string", "info", "Log level: debug, info, warn, error"},
			{"log-format", "string", "text", "Log format: text or json"},
			{"config", "string", "~/.xwebs.yaml", "Config file path"},
			{"profile", "string", "", "Named profile from config file"},
			{"handlers", "string", "", "Handler config file (YAML)"},
			{"on", "stringArray", "", "Inline handler: '<match> :: <action>'"},
			{"respond", "string", "", "Default response template for --on handlers"},
			{"no-shell-func", "bool", "false", "Disable shell/env/fileRead template functions"},
			{"sandbox", "bool", "false", "Enable shell command allowlisting"},
			{"allowlist", "strings", "", "Allowed shell commands (requires --sandbox)"},
			{"once", "bool", "false", "Exit after receiving the first response"},
			{"input", "string", "", "Send messages from file (one per line)"},
			{"send", "string", "", "Send this message on connect"},
			{"expect", "int", "", "Exit after receiving N responses"},
			{"until", "string", "", "Exit when this template evaluates truthy"},
			{"output", "string", "", "Write responses to file"},
			{"jsonl", "bool", "false", "Output one JSON message per line"},
			{"script", "string", "", "Execute a .xwebs script file"},
			{"watch", "string", "", "Re-send file content on change"},
			{"timeout", "duration", "", "Exit after this duration"},
			{"exit-on", "string", "", "Template producing integer exit code"},
		},
		Examples: []string{
			`# Interactive REPL
xwebs connect wss://echo.websocket.org`,
			`# Non-interactive: send once and exit
echo '{"type":"ping"}' | xwebs connect wss://api.example.com --once`,
			`# With auth header (template expanded)
xwebs connect wss://api.example.com \
  --header "Authorization: Bearer {{env \"TOKEN\"}}"`,
			`# Pipe JSON responses to jq
xwebs connect wss://stream.example.com --jsonl | jq '.price'`,
			`# With inline handler
xwebs connect wss://api.example.com \
  --on '.type == "error" :: run:notify-send "Error" "{{.Message}}"'`,
		},
	},
	{
		Name:  "serve",
		Short: "Start a WebSocket server",
		Long:  "Start a WebSocket server that responds to incoming messages using configurable handler pipelines. Handlers are defined in a YAML file (--handlers) or inline with --on flags. In interactive mode (--interactive), a server REPL is started for live administration.",
		Use:   "xwebs serve [flags]",
		Flags: []Flag{
			{"port", "int", "8080", "Port to listen on"},
			{"host", "string", "0.0.0.0", "Host/interface to bind"},
			{"path", "stringArray", "/ws", "WebSocket endpoint path(s)"},
			{"tls", "bool", "false", "Enable TLS"},
			{"cert", "string", "", "TLS certificate file (PEM)"},
			{"key", "string", "", "TLS key file (PEM)"},
			{"ui", "bool", "false", "Enable web UI at /"},
			{"metrics", "bool", "false", "Expose Prometheus metrics at /api/metrics"},
			{"interactive", "bool", "auto", "Force server REPL"},
			{"handler-timeout", "duration", "30s", "Default handler execution timeout"},
			{"max-message-size", "string", "16MB", "Maximum incoming message size"},
			{"allowed-origins", "strings", "", "CORS allowed origins"},
			{"rate-limit", "string", "", "Global rate limit (N/s or N/m)"},
			{"allow-ip", "strings", "", "IP allowlist (CIDR)"},
			{"deny-ip", "strings", "", "IP denylist (CIDR)"},
			{"handlers", "string", "", "Handler config file (YAML)"},
			{"on", "stringArray", "", "Inline handler: '<match> :: <action>'"},
			{"respond", "string", "", "Default response template for --on handlers"},
			{"no-shell-func", "bool", "false", "Disable shell template functions"},
			{"sandbox", "bool", "false", "Enable shell command allowlisting"},
			{"allowlist", "strings", "", "Allowed shell commands"},
			{"verbose", "bool", "false", "Verbose output"},
			{"quiet", "bool", "false", "Suppress non-essential output"},
			{"log-level", "string", "info", "Log level"},
			{"log-format", "string", "text", "Log format: text or json"},
			{"config", "string", "~/.xwebs.yaml", "Config file"},
			{"profile", "string", "", "Named profile"},
		},
		Examples: []string{
			`# Basic server
xwebs serve --port 8080`,
			`# With handler config
xwebs serve --port 8080 --handlers handlers.yaml`,
			`# Inline ping/pong
xwebs serve --port 8080 --on 'ping :: respond:pong'`,
			`# With TLS
xwebs serve --port 8443 --tls --cert server.crt --key server.key`,
			`# Interactive REPL for live administration
xwebs serve --port 8080 --handlers handlers.yaml --interactive`,
		},
	},
	{
		Name:  "relay",
		Short: "Proxy/MITM WebSocket traffic between a client and upstream server",
		Long:  "Sit between a client and an upstream WebSocket server. Inspect, log, and optionally transform messages in both directions. Useful for debugging production protocols or recording sessions for replay.",
		Use:   "xwebs relay --listen <addr> --upstream <url> [flags]",
		Flags: []Flag{
			{"listen", "string", ":9090", "Local address to listen on"},
			{"upstream", "string", "", "Upstream WebSocket URL (required)"},
			{"client-to-server", "string", "", "Transform template for client→server messages"},
			{"server-to-client", "string", "", "Transform template for server→client messages"},
			{"log", "string", "", "Log all traffic to JSONL file"},
		},
		Examples: []string{
			`xwebs relay --listen :9090 --upstream wss://api.example.com`,
			`xwebs relay --listen :9090 --upstream wss://api.example.com \
  --server-to-client '{{.Message | prettyJSON}}' \
  --log relay.jsonl`,
		},
	},
	{
		Name:  "broadcast",
		Short: "Fan-out pub/sub WebSocket server",
		Long:  "Start a simple broadcast server where every message received from any client is forwarded to all connected clients. With --topics enabled, clients can subscribe to named channels.",
		Use:   "xwebs broadcast [flags]",
		Flags: []Flag{
			{"port", "int", "8080", "Port to listen on"},
			{"topics", "bool", "false", "Enable topic-based subscription"},
		},
		Examples: []string{
			`xwebs broadcast --port 8080`,
			`xwebs broadcast --port 8080 --topics`,
		},
	},
	{
		Name:  "mock",
		Short: "Start a scripted mock WebSocket server",
		Long:  "Start a mock server that responds to clients according to a scenario file. Scenarios define an ordered sequence of expect/respond steps, simulating a real server's behavior without running actual backend code.",
		Use:   "xwebs mock --port <port> --scenario <file> [flags]",
		Flags: []Flag{
			{"port", "int", "8080", "Port to listen on"},
			{"scenario", "string", "", "Scenario YAML file (required)"},
		},
		Examples: []string{
			`xwebs mock --port 8080 --scenario test/auth-flow.yaml`,
		},
	},
	{
		Name:  "bench",
		Short: "Load test a WebSocket endpoint",
		Long:  "Open N concurrent connections and send messages at a configured rate for a specified duration. Reports latency percentiles, throughput, and error rates.",
		Use:   "xwebs bench <url> [flags]",
		Flags: []Flag{
			{"connections", "int", "10", "Number of concurrent connections"},
			{"rate", "int", "1", "Messages per second per connection"},
			{"duration", "duration", "10s", "Test duration"},
			{"message", "string", "", "Message template (evaluated per send)"},
			{"output", "string", "", "Write results to file"},
		},
		Examples: []string{
			`xwebs bench wss://api.example.com \
  --connections 100 \
  --rate 10 \
  --duration 60s \
  --message '{"type":"ping","id":"{{counter "req"}}"}'`,
		},
	},
	{
		Name:  "replay",
		Short: "Replay a recorded xwebs session",
		Long:  "Replay a JSONL session file recorded with xwebs connect --record. Can replay against a live server (--target) or serve the recorded responses as a mock server (--serve).",
		Use:   "xwebs replay <file> [flags]",
		Flags: []Flag{
			{"target", "string", "", "Target WebSocket URL to replay against"},
			{"serve", "bool", "false", "Serve recorded responses as a mock server"},
			{"port", "int", "8080", "Port for mock server mode"},
			{"speed", "float", "1.0", "Playback speed multiplier"},
		},
		Examples: []string{
			`xwebs replay session.jsonl --target wss://staging.api.example.com`,
			`xwebs replay session.jsonl --serve --port 8080`,
		},
	},
	{
		Name:  "diff",
		Short: "Compare responses from two WebSocket endpoints",
		Long:  "Send the same messages to two WebSocket servers and compare their responses. Useful for verifying API compatibility between versions.",
		Use:   "xwebs diff <url1> <url2> [flags]",
		Flags: []Flag{
			{"input", "string", "", "Messages file (one per line)"},
			{"format", "string", "text", "Output format: text or json"},
		},
		Examples: []string{
			`xwebs diff wss://v1.api.example.com wss://v2.api.example.com --input messages.txt`,
		},
	},
}
