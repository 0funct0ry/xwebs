---
title: CLI Reference
description: Complete command-line reference for all xwebs commands and flags.
---

# CLI Reference

```
xwebs — WebSocket Swiss Army Knife

Usage:
  xwebs <command> [flags]

Commands:
  connect      Connect to a WebSocket server
  serve        Start a WebSocket server
  relay        Proxy/MITM between client and server
  broadcast    Fan-out pub/sub server
  mock         Start a mock server from scenario files
  bench        Load test a WebSocket endpoint
  replay       Replay a recorded session
  diff         Compare responses from two endpoints
  completion   Generate shell completion scripts
  version      Print version information
  help         Help about any command
```

---

## `xwebs connect`

Connect to a WebSocket server in interactive (REPL) or non-interactive (pipeline) mode.

```
xwebs connect [url] [flags]
```

### Synopsis

```bash
# Interactive REPL
xwebs connect wss://echo.websocket.org

# Non-interactive: pipe in, get response, exit
echo '{"type":"ping"}' | xwebs connect wss://api.example.com --once

# With auth headers
xwebs connect wss://api.example.com \
  --header "Authorization: Bearer {{env \"TOKEN\"}}" \
  --header "X-Request-ID: {{uuid}}"
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--header` | stringArray | — | HTTP header to include (can repeat) |
| `--subprotocol` | stringArray | — | WebSocket subprotocol(s) to negotiate |
| `--cert` | string | — | TLS client certificate file (PEM) |
| `--key` | string | — | TLS client key file (PEM) |
| `--ca` | string | — | CA certificate file for server verification |
| `--insecure` | bool | false | Skip TLS certificate verification |
| `--compress` | bool | false | Enable permessage-deflate compression |
| `--max-message-size` | string | `16MB` | Maximum incoming message size |
| `--ping-interval` | duration | `30s` | Keepalive ping interval |
| `--reconnect` | bool | true | Automatically reconnect on disconnect |
| `--reconnect-max` | duration | `5m` | Maximum reconnect backoff |
| `--interactive` | bool | auto | Force interactive REPL (even without TTY) |
| `--keymap` | string | `emacs` | REPL keymap: `emacs` or `vi` |
| `--history-size` | int | `10000` | Maximum history entries |
| `--proxy` | string | — | Proxy URL (`http://`, `https://`, `socks5://`) |
| `-v, --verbose` | bool | false | Verbose output |
| `-q, --quiet` | bool | false | Suppress non-essential output |
| `--color` | string | `auto` | Color mode: `auto`, `on`, `off` |
| `--log-level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | string | `text` | Log format: `text` or `json` |
| `-c, --config` | string | `~/.xwebs.yaml` | Config file path |
| `--profile` | string | — | Named profile from config |
| `--handlers` | string | — | Handler config file (YAML) |
| `--on` | stringArray | — | Inline handler: `'<match> :: <action>'` |
| `--respond` | string | — | Default response template for `--on` handlers |
| `--no-shell-func` | bool | false | Disable shell/env/fileRead template functions |
| `--sandbox` | bool | false | Enable shell command allowlisting |
| `--allowlist` | strings | — | Allowed shell commands (requires `--sandbox`) |

**Non-interactive flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--once` | bool | false | Exit after receiving the first response |
| `--input` | string | — | Send messages from file (one per line) |
| `--send` | string | — | Send this message on connect |
| `--expect` | int | — | Exit after receiving N responses |
| `--until` | string | — | Exit when this template evaluates truthy |
| `--output` | string | — | Write responses to file |
| `--jsonl` | bool | false | Output one JSON message per line |
| `--script` | string | — | Execute a `.xwebs` script file |
| `--watch` | string | — | Re-send file content on change |
| `--timeout` | duration | — | Exit after this duration |
| `--exit-on` | string | — | Template producing integer exit code |

---

## `xwebs serve`

Start a WebSocket server with configurable message handlers.

```
xwebs serve [flags]
```

### Synopsis

```bash
# Basic server
xwebs serve --port 8080

# With handler config
xwebs serve --port 8080 --handlers handlers.yaml

# With TLS
xwebs serve --port 8443 --tls --cert server.crt --key server.key

# With Web UI
xwebs serve --port 8080 --ui

# Interactive REPL for live administration
xwebs serve --port 8080 --handlers handlers.yaml --interactive

# Multiple paths
xwebs serve --port 8080 --path /ws --path /events
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-p, --port` | int | `8080` | Port to listen on |
| `--host` | string | `0.0.0.0` | Host/interface to bind to |
| `--path` | stringArray | `/ws` | WebSocket endpoint path(s) |
| `--tls` | bool | false | Enable TLS |
| `--cert` | string | — | TLS certificate file (PEM) |
| `--key` | string | — | TLS key file (PEM) |
| `--ui` | bool | false | Enable web UI at `/` |
| `--metrics` | bool | false | Expose Prometheus metrics at `/api/metrics` |
| `--interactive` | bool | auto | Force server REPL |
| `--handler-timeout` | duration | `30s` | Default handler execution timeout |
| `--max-message-size` | string | `16MB` | Maximum incoming message size |
| `--ping-interval` | duration | `30s` | Keepalive ping interval |
| `--allowed-origins` | strings | — | CORS allowed origins (comma-separated) |
| `--rate-limit` | string | — | Global rate limit (`N/s`, `N/m`) |
| `--allow-ip` | strings | — | IP allowlist (comma-separated CIDRs) |
| `--deny-ip` | strings | — | IP denylist (comma-separated CIDRs) |
| `--handlers` | string | — | Handler config file (YAML) |
| `--on` | stringArray | — | Inline handler |
| `--respond` | string | — | Default response template |
| `--no-shell-func` | bool | false | Disable shell template functions |
| `--sandbox` | bool | false | Enable shell command allowlisting |
| `--allowlist` | strings | — | Allowed shell commands |
| `-v, --verbose` | bool | false | Verbose output |
| `-q, --quiet` | bool | false | Suppress non-essential output |
| `--log-level` | string | `info` | Log level |
| `--log-format` | string | `text` | Log format |
| `-c, --config` | string | `~/.xwebs.yaml` | Config file |
| `--profile` | string | — | Named profile |

---

## `xwebs relay`

Sit between a client and an upstream server — inspect and optionally transform messages in both directions.

```
xwebs relay --listen <addr> --upstream <url> [flags]
```

### Synopsis

```bash
# Basic MITM proxy
xwebs relay --listen :9090 --upstream wss://api.example.com

# With transformation
xwebs relay --listen :9090 --upstream wss://api.example.com \
  --client-to-server '{{.Message | prettyJSON}}' \
  --server-to-client '{{.Message | jq ".data"}}'

# Log all traffic
xwebs relay --listen :9090 --upstream wss://api.example.com \
  --log relay.jsonl
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--listen` | string | `:9090` | Local address to listen on |
| `--upstream` | string | — | Upstream WebSocket URL |
| `--client-to-server` | string | — | Transform template for client→server messages |
| `--server-to-client` | string | — | Transform template for server→client messages |
| `--log` | string | — | Log all traffic to JSONL file |

---

## `xwebs broadcast`

Fan-out server — all connected clients receive all messages.

```
xwebs broadcast [flags]
```

### Synopsis

```bash
xwebs broadcast --port 8080

# With topic support
xwebs broadcast --port 8080 --topics
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | `8080` | Port to listen on |
| `--topics` | bool | false | Enable topic-based subscription |

---

## `xwebs mock`

Start a scripted mock server from a scenario file.

```
xwebs mock --port <port> --scenario <file> [flags]
```

### Synopsis

```bash
xwebs mock --port 8080 --scenario test/auth-flow.yaml
```

### Scenario File Format

```yaml
scenarios:
  - name: auth-flow
    steps:
      - expect:
          jq: '.type == "auth"'
        respond: '{"type":"auth_ok","session":"{{uuid}}"}'
        delay: 100ms
      - expect:
          jq: '.type == "subscribe"'
        respond: '{"type":"subscribed","channel":"{{.Message | jq ".channel"}}"}'
      - after: 1s
        send: '{"type":"event","data":"mock-event-1"}'
```

---

## `xwebs bench`

Load test a WebSocket endpoint with concurrent connections.

```
xwebs bench <url> [flags]
```

### Synopsis

```bash
# 100 connections, 10 msg/s each, for 60 seconds
xwebs bench wss://api.example.com \
  --connections 100 \
  --rate 10 \
  --duration 60s \
  --message '{"type":"ping","id":"{{counter "req"}}"}'
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--connections` | int | `10` | Number of concurrent connections |
| `--rate` | int | `1` | Messages per second per connection |
| `--duration` | duration | `10s` | Test duration |
| `--message` | string | — | Message template (evaluated per send) |
| `--output` | string | — | Write results to file |

**Output:** latency percentiles (p50, p95, p99), throughput (msg/s), error rate, connection stats.

---

## `xwebs replay`

Replay a previously recorded session against a server or as a mock.

```
xwebs replay <file> [flags]
```

### Synopsis

```bash
# Replay against staging
xwebs replay session.jsonl --target wss://staging.api.example.com

# Serve recorded responses as a mock
xwebs replay session.jsonl --serve --port 8080
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--target` | string | — | Target WebSocket URL |
| `--serve` | bool | false | Serve as a mock server |
| `--port` | int | `8080` | Port for mock server |
| `--speed` | float | `1.0` | Playback speed multiplier |

---

## `xwebs diff`

Compare responses from two WebSocket endpoints for the same input.

```
xwebs diff <url1> <url2> [flags]
```

### Synopsis

```bash
xwebs diff wss://v1.api.example.com wss://v2.api.example.com \
  --input messages.txt
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--input` | string | — | Messages file (one per line) |
| `--format` | string | `text` | Output format: `text` or `json` |

---

## `xwebs completion`

Generate shell completion scripts.

```
xwebs completion <shell>
```

Supported shells: `bash`, `zsh`, `fish`, `powershell`.

```bash
# Bash
xwebs completion bash > /usr/local/etc/bash_completion.d/xwebs

# Zsh
xwebs completion zsh > "${fpath[1]}/_xwebs"

# Fish
xwebs completion fish > ~/.config/fish/completions/xwebs.fish
```

---

## `xwebs version`

Print version, commit, and build information.

```bash
xwebs version
# xwebs v1.0.0 (commit: abc1234, built: 2024-01-15T10:00:00Z)
```

---

## Global Config File

`~/.xwebs.yaml` (or `.xwebs.yaml` in the current directory):

```yaml
defaults:
  format: json
  color: auto
  timestamps: true
  history_size: 10000
  keymap: emacs
  handler_timeout: 30s

aliases:
  prod: "wss://prod-api.example.com/ws"
  staging: "wss://staging-api.example.com/ws"

bookmarks:
  - name: Production API
    url: wss://prod-api.example.com/ws
    headers:
      Authorization: "Bearer {{env \"PROD_TOKEN\"}}"

profiles:
  debug:
    verbose: true
    log-level: debug
  ci:
    color: false
    quiet: true
```

## Environment Variables

Every flag has an `XWEBS_` equivalent:

```bash
XWEBS_PORT=8080
XWEBS_LOG_LEVEL=debug
XWEBS_HANDLERS=handlers.yaml
XWEBS_FORMAT=json
XWEBS_UI=true
```
