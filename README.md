# xwebs

A CLI tool in Go for WebSocket-based development with shell integration, Go templates, and an optional React web UI.

## Why xwebs?

Every WebSocket tool does one thing: connect and send messages. That's the equivalent of having `telnet` in a world that already has `curl`, `jq`, `awk`, and shell pipelines. WebSockets are the backbone of real-time systems — chat, dashboards, IoT, trading, CI/CD — yet the developer tooling around them is stuck in the "raw socket" era.

**xwebs flips the model.** Instead of treating WebSocket messages as dumb payloads, it treats them as **events that trigger shell pipelines**, with full Go template interpolation, pattern matching, an interactive REPL, a server mode, a relay/proxy mode, and an optional web UI. Think of it as `curl` + `netcat` + `jq` + `bash` — but for WebSockets.

## Features

- **Client Mode** — Connect to WebSocket servers, send/receive messages, automate interactions
- **Server Mode** — Spin up WebSocket handlers, serve a web UI, expose metrics
- **Relay Mode** — Man-in-the-middle inspection and transformation
- **Broadcast Mode** — Pub/sub fan-out for multi-client testing
- **Mock Mode** — Scenario-driven fake servers for integration tests
- **Bench Mode** — Load testing with concurrent connections and rate control
- **Replay Mode** — Record and replay sessions deterministically
- **Interactive REPL** — Tab completion, syntax highlighting, command history
- **Go Templates** — Dynamic message payloads, handler configs, and expressions
- **Shell Integration** — Execute shell commands in response to WebSocket events

## Installation

### From Source

```bash
git clone https://github.com/0funct0ry/xwebs.git
cd xwebs
go install
```

### Build from Source

```bash
# Build for current platform
make build

# Cross-compile for multiple platforms
make build-all
```

## Quick Start

### Connect to a WebSocket server

```bash
xwebs connect wss://echo.websocket.org
```

### Start a WebSocket server

```bash
xwebs serve --port 8080
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

### Building

```bash
make build        # Build the binary
make test         # Run tests
make lint         # Run linters
make install      # Install to $GOPATH/bin
```




## License

MIT License - see [LICENSE](./LICENSE) for details.
