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
- **CLI Foundation** — Robust command structure with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper)
- **Configuration Profiles** — Switch between named settings (e.g., `--profile debug`)
- **Aliases & Bookmarks** — Map short names to long WebSocket URLs and headers
- **Shell Completion** — Native completion for Bash, Zsh, Fish, and PowerShell
- **Version Info** — Detailed build information with `xwebs version`
- **Makefile Integration** — Standardized `build`, `test`, `lint`, and `install` targets
- **CI/CD** — Automated testing and building via GitHub Actions

### On the Roadmap (Planned)
- **Client Mode** — Full interactive REPL for WebSocket communication
- **WebSocket Engine** — TLS support, proxies, auto-reconnect, and frame handling
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

### Resolve Connection Details

While the full interactive client is coming in EPIC 04, you can currently use `connect` to resolve aliases and bookmarks from your configuration:

```bash
# Resolve a bookmark
xwebs connect staging

# Resolve a raw URL
xwebs connect wss://echo.websocket.org
```

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
