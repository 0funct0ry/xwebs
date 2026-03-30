# xwebs

[![CI](https://github.com/0funct0ry/xwebs/actions/workflows/ci.yml/badge.svg)](https://github.com/0funct0ry/xwebs/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.23.0-blue.svg)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A CLI tool for WebSocket-based development with shell integration, Go templates, and an optional React web UI.

## Why xwebs?

Every WebSocket tool does one thing: connect and send messages. That's the equivalent of having `telnet` in a world that already has `curl`, `jq`, `awk`, and shell pipelines. WebSockets are the backbone of real-time systems — chat, dashboards, IoT, trading, CI/CD — yet the developer tooling around them is stuck in the "raw socket" era.

**xwebs flips the model.** Instead of treating WebSocket messages as dumb payloads, it treats them as **events that trigger shell pipelines**, with full Go template interpolation, pattern matching, an interactive REPL, a server mode, a relay/proxy mode, and an optional web UI. Think of it as `curl` + `netcat` + `jq` + `bash` — but for WebSockets.

## Features

### Available Now (v0.1.0-alpha)
- **WebSocket Engine** — Bidirectional message flow with goroutines/channels, Proxy support (HTTP/SOCKS5), full TLS support (custom CAs, mTLS, insecure mode), and automatic Ping/Pong keepalive
- **Configuration Profiles** — Switch between named settings (e.g., `--profile debug`)
- **Aliases & Bookmarks** — Map short names to long WebSocket URLs, headers, and TLS settings
- **Shell Completion** — Native completion for Bash, Zsh, Fish, and PowerShell
- **Version Info** — Detailed build information with `xwebs version`
- **Makefile Integration** — Standardized `build`, `test`, `lint`, and `install` targets
- **CI/CD** — Automated testing and building via GitHub Actions

### On the Roadmap (Planned)
- **Client Mode** — Full interactive REPL for WebSocket communication
- **Template Engine** — Rich Go template FuncMap with `jq`, `base64`, `crypto`, and more
- **Server Mode** — WebSocket server with handler dispatch and administration REPL
- **Handler Pipeline** — Bind message patterns to shell commands and actions
- **Relay & Broadcast** — MITM proxy and pub/sub fan-out modes
- **Mock & Replay** — Scenario-driven testing and session recording
- **Web UI** — React-based dashboard for visual message inspection and Compose

## Installation

### Via Go Install

If you have Go installed, you can install the latest version directly:

```bash
go install github.com/0funct0ry/xwebs@latest
```

### Binary Downloads (Planned)

In the future, pre-built binaries for Linux, macOS, and Windows will be available on the [Releases](https://github.com/0funct0ry/xwebs/releases) page.

### From Source

For development or to build the latest version from source:

```bash
git clone https://github.com/0funct0ry/xwebs.git
cd xwebs
make build
# To install locally to $GOPATH/bin
make install
```

### Build for Multiple Platforms

```bash
make build-all
```

## Quick Start

### Check Version

Verify your installation and see build details:

```bash
xwebs version
```

### WebSocket Connection

The `connect` command performs a WebSocket handshake, injects custom headers, and negotiates subprotocols.

```bash
# Direct URL
xwebs connect wss://echo.websocket.org

# With custom subprotocols
xwebs connect wss://echo.websocket.org --subprotocol v1.xwebs,mqtt

# Using an alias/bookmark (defined in config)
xwebs connect staging

### TLS Configuration

`xwebs` provides full support for secure WebSocket connections (`wss://`).

```bash
# Skip TLS verification (useful for local development with self-signed certs)
xwebs connect wss://localhost:8443 --insecure

# Use a custom CA certificate to validate the server
xwebs connect wss://api.internal.com --ca ./internal-ca.crt

# Mutual TLS (mTLS) with client certificate and key
xwebs connect wss://secure.gateway.com --cert ./client.crt --key ./client.key
```

### Proxy Configuration

`xwebs` supports routing WebSocket connections through HTTP CONNECT and SOCKS5 proxies.

```bash
# HTTP Proxy
xwebs connect ws://echo.websocket.org --proxy http://localhost:8080

# SOCKS5 Proxy
xwebs connect ws://echo.websocket.org --proxy socks5://localhost:1080

# Proxy with authentication
xwebs connect ws://echo.websocket.org --proxy socks5://user:password@proxy.internal:1080
```

### Keepalive Configuration

`xwebs` can automatically send ping frames to keep connections alive and detect unhealthy connections.

```bash
# Send ping every 10 seconds, timeout if no pong within 30 seconds
xwebs connect wss://api.example.com --ping-interval 10s --pong-wait 30s

# Disable automatic pings (default is 30s interval)
xwebs connect wss://api.example.com --ping-interval 0s
```

Keepalive settings can be defined in bookmarks:

```yaml
bookmarks:
  long-lived:
    url: "wss://events.example.com"
    ping-interval: 30s
    pong-wait: 60s
```

Proxy settings can also be defined in bookmarks:

```yaml
bookmarks:
  internal-proxy:
    url: "wss://api.internal.com"
    proxy: "http://squid.corp.local:3128"
```

### Reconnection Configuration

`xwebs` can automatically reconnect to the server if the connection is lost unexpectedly. Reconnection uses an exponential backoff strategy to avoid overwhelming the server.

```bash
# Enable reconnection with default parameters (1s initial backoff, 30s max, unlimited attempts)
xwebs connect wss://api.example.com --reconnect

# Custom backoff and limit to 10 attempts
xwebs connect wss://api.example.com --reconnect --reconnect-backoff 2s --reconnect-max 60s --reconnect-attempts 10
```

Reconnection settings can be defined in bookmarks:

```yaml
bookmarks:
  unstable-server:
    url: "wss://unstable.example.com"
    reconnect: true
    reconnect-backoff: 1s
    reconnect-max: 30s
    reconnect-attempts: 5
```

Currently, `connect` establishes the connection and reports handshake details. Full interactive REPL support is coming in EPIC 04.

### Generate Completion

Enable shell completion for a better CLI experience:

```bash
# Example for Zsh (add to ~/.zshrc)
source <(xwebs completion zsh)
```

### Use configuration file

xwebs automatically loads configuration from:
- `~/.xwebs.yaml` (user home directory)
- `.xwebs.yaml` (current directory)

Example configuration:

```yaml
verbose: false
quiet: false
color: auto
log-level: info
log-format: text
```

### Environment Variables

All configuration values can be overridden with environment variables prefixed with `XWEBS_`:

```bash
XWEBS_VERBOSE=true xwebs connect wss://example.com
XWEBS_LOG_LEVEL=debug xwebs serve --port 8080
```

### Command-line Flags

```
Usage: xwebs [flags]

Global Flags:
  -c, --config string      config file path (default searches ~/.xwebs.yaml then .xwebs.yaml)
  -v, --verbose            enable verbose output
  -q, --quiet              suppress all output except errors
      --proxy string       proxy URL (http, https, socks5)
      --color string      color output mode: auto, on, off (default "auto")
      --log-level string   logging level: debug, info, warn, error (default "info")
      --log-format string  log format: text, json (default "text")
  -h, --help               help for xwebs
```

## Configuration

### Config File Locations

xwebs searches for configuration files in the following order (first found wins):

1. Custom path specified via `-c, --config` flag
2. `.xwebs.yaml` in current working directory
3. `~/.xwebs.yaml` (user home directory)

### Environment Variable Mapping

| Flag           | Environment Variable |
|----------------|----------------------|
| `--verbose`    | `XWEBS_VERBOSE`      |
| `--quiet`      | `XWEBS_QUIET`        |
| `--color`      | `XWEBS_COLOR`        |
| `--log-level`  | `XWEBS_LOG_LEVEL`    |
| `--log-format` | `XWEBS_LOG_FORMAT`   |
| `--profile`    | `XWEBS_PROFILE`      |

### Named Profiles

Profiles allow you to group settings and apply them as an overlay to the base configuration. Use the `--profile` flag to specify a profile defined in your config file.

Example configuration with profiles:

```yaml
log-level: info
verbose: false

profiles:
  debug:
    log-level: debug
    verbose: true
  prod:
    log-level: error
    verbose: false
```

Usage:
```bash
xwebs --profile debug
```

### Precedence

Configuration values are applied in the following order (later takes precedence):

1. Default values
2. Config file base values (`~/.xwebs.yaml` or `.xwebs.yaml`)
3. **Named Profile values** (if `--profile` is specified)
4. Environment variables (`XWEBS_*`)
5. Command-line flags

### Aliases and Bookmarks

Aliases and bookmarks allow you to define short names for frequently used WebSocket endpoints. Bookmarks also support pre-configured HTTP headers.

Example configuration:

```yaml
aliases:
  echo: "wss://echo.websocket.org"
  local: "ws://localhost:8080"

bookmarks:
  staging:
    url: "wss://api.staging.example.com"
    headers:
      X-API-Key: "secret-abc-123"
      Authorization: "Bearer your-token-here"
  secure-prod:
    url: "wss://api.prod.example.com"
    insecure: false
    ca: "/path/to/ca.crt"
    cert: "/path/to/client.crt"
    key: "/path/to/client.key"
```

Usage:
```bash
xwebs connect staging
```

### Build Information

```bash
xwebs version
```

Displays the version, git commit hash, build date, and Go version.

### Shell Completion

Generate shell completion scripts for Bash, Zsh, Fish, or PowerShell.

```bash
# Bash
source <(xwebs completion bash)

# Zsh
xwebs completion zsh > "${fpath[1]}/_xwebs"

# Fish
xwebs completion fish > ~/.config/fish/completions/xwebs.fish

# PowerShell
xwebs completion powershell | Out-String | Invoke-Expression
```

## Examples

See the `examples/` directory for sample configuration files and usage examples.

## Development

The project includes a `Makefile` with standard development targets:

- `make` - Show help and available targets
- `make build` - Build the `xwebs` binary for the current platform
- `make build-prod` - Build optimized binary with production flags
- `make build-all` - Cross-compile for Linux, Darwin, and Windows
- `make test` - Run all tests with verbose output
- `make lint` - Run `golangci-lint` (falls back to `go vet`)
- `make install` - Install the binary to `$GOPATH/bin`
- `make clean` - Remove build artifacts
- `make ci` - Run `fmt`, `vet`, `test`, and `build` in sequence

Usage:
```bash
# Build the project
make build

# Run tests
make test

# Install globally
make install
```




## CI Pipeline

The project uses GitHub Actions for Continuous Integration. Every push and pull request triggers a workflow that:

1. **Tests** across multiple Go versions (1.22.x, 1.23.x).
2. **Lints** the codebase using `golangci-lint`.
3. **Builds** the binary to ensure compilation succeeds.
4. **Uploads** the built binary as a workflow artifact for validation.

You can view the latest CI status [here](https://github.com/0funct0ry/xwebs/actions).

## License

MIT License - see [LICENSE](./LICENSE) for details.
