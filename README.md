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
- **WebSocket Engine** — Bidirectional message flow with goroutines/channels
- **Advanced TLS support** (custom CAs, mTLS, insecure mode)
- **Proxy support** (HTTP & SOCKS5)
- **Graceful close** with custom codes and reasons
- **Automatic reconnection** with exponential backoff
- **Keepalive** (ping/pong) with configurable intervals
- **Message fragmentation** control
- **Per-message-deflate compression**
- **Custom Headers & Authentication** — repeatable `--header` flags, `--token` (Bearer), and `--auth` (Basic) with full **Go template support**
- **Template Engine** — Rich Go template FuncMap with `now`, `jq`, `base64`, `crypto`, and more (core engine implemented)
- **Configuration Profiles** — Switch between named settings (e.g., `--profile debug`)
- **Aliases & Bookmarks** — Map short names to long WebSocket URLs, headers, and TLS settings
- **Shell Completion** — Native completion for Bash, Zsh, Fish, and PowerShell
- **Version Info** — Detailed build information with `xwebs version`
- **Makefile Integration** — Standardized `build`, `test`, `lint`, and `install` targets
- **CI/CD** — Automated testing and building via GitHub Actions
- **Client Mode & REPL** — Robust, shared REPL framework with command history, tab completion, and multi-mode support
- **Core REPL Commands** — Built-in commands for session management (`:set`, `:vars`), connection status (`:status`), and WebSocket operations (`:ping`, `:pong`, `:close`, `:send`, `:sendb`, `:sendj`, `:sendt`)
- **Ping/Pong Observability** — Send and receive WebSocket control frames (ping/pong) with text or binary payloads, visible in the REPL session
- **Connection Management** — Dynamic `:connect` and `:reconnect` within the active REPL session
- **Output Formatting & Filtering** — Flexible display options including JSON pretty-printing, hex dumps, and `jq` or Regex message filters
- **Automation & Scripting** — Multi-step automation with `:source`, `:alias`, `:wait`, and `:assert` commands, plus a `--script` flag for non-interactive execution
- **Observability & Testing** — High-fidelity JSONL logging, session recording/replay, and scenario-based mocking with `gojq` matching
- **Message Handlers** — Declarative YAML-based message matching and action execution with lifecycle support
- **RTT & Latency Tracking** — Real-time performance metrics for round-trip time, accessible via `.LastLatencyMs` in templates
- **Real-time Syntax Highlighting** — Visual feedback for JSON and Go template expressions as typed in the REPL
- **Shell Direct Execution** — Run arbitrary shell commands with `:! <command>` directly from the REPL, supporting both captured output and interactive modes
- **Server Mode** — WebSocket server with multiple paths, handler support, graceful shutdown, and an interactive admin console for managing server state, connections, and handlers directly from the terminal.
- **Server Administration REPL** — Interactive admin console for managing server state, connections, handlers, and pub-sub topics directly from the terminal.
- **Pub-Sub Topics** — Internal pub-sub bus: clients subscribe to named topics via handler-dispatched builtins; operators publish and inspect subscriptions from the REPL with `:topics`, `:topic`, `:publish`, `:subscribe`, and `:unsubscribe`.

### On the Roadmap (Planned)
- **Web UI** — React-based dashboard for visual server monitoring and management (embedded via `go:embed`) (Planned)
- **Relay & Broadcast** — MITM proxy and fan-out relay modes (Planned)

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

The `connect` command performs a WebSocket handshake, injects custom headers, and negotiates subprotocols. Once connected, it enters a basic interactive mode where you can type messages to send and see incoming messages from the server.

```bash
# Direct URL
xwebs connect wss://echo.websocket.org
```

### Interactive REPL Mode

When running in a terminal (TTY), `xwebs connect` enters a rich interactive REPL mode. Unlike basic "line-at-a-time" interfaces, this REPL supports powerful built-in commands starting with a colon `:`.

**Common REPL Commands:**

| Command          | Description                                          |
|------------------|------------------------------------------------------|
| `:help`          | List all available commands                          |
| `:hedit [-n <n>]`| Edit a previous command in your $EDITOR              |
| `:status`        | Show detailed connection metadata                    |
| `:send <text>`   | Send a text message (default for bare text)          |
| `:sendb <hex>`   | Send binary data (hex or `base64:`)                  |
| `:sendj <json>`  | Send validated JSON message                          |
| `:sendt <tmpl>`  | Send rendered Go template                            |
| `:ping [p]`      | Send a ping frame (text or binary prefix)            |
| `:pong [p]`      | Send a pong frame (text or binary prefix)            |
| `:connect <url>` | Connect to a new URL in the same session             |
| `:reconnect`     | Force a reconnection to the current URL              |
| `:close [c][r]`  | Send a graceful close frame                          |
| `:disconnect`    | Disconnect from the current server                   |
| `:set <k> <v>`   | Set a session variable (Client mode only)            |
| `:get <k>`       | Get a session variable (Client mode only)            |
| `:vars`          | List session variables (Client mode only)             |
| `:env`           | List all environment variables                       |
| `:pwd [var]`     | Show current working directory (optional save to var)|
| `:cd [path\|-]`  | Change the current working directory                 |
| `:ls [-l] [path]`| List directory contents (with optional detailed view)|
| `:mkdir [-p] <d>`| Create a new directory (with optional parent creation)|
| `:cat <file>`    | Display file contents with syntax highlighting       |
| `:edit <file>`   | Open a file in your $EDITOR and offer reload         |
| `:write <f> <c>` | Save templated content or session data to a file     |
| `:history [flags]`| Display/manage command history (search, filter, export) |
| `:bench <n> <m>` | Benchmark latency for N iterations                   |
| `:flood <msg>`   | Stress test server with high-rate messages           |
| `:watch`         | Monitor connection statistics in real-time           |
| `:exit`, `:quit` | Disconnect and quit the application (or `Ctrl+D`)    |
| `:clear`         | Clear the terminal screen                            |
| `:! [-i] <cmd>`  | Execute a shell command (use `-i` for interactive)   |
| `:shell`, `:sh`  | Switch to a full interactive shell session (use `--yes` to skip prompt)|
| `:format <type>` | Set display format: `json`, `raw`, `hex`, `template` |
| `:filter <expr>` | Set a display filter (`.jq`, `/regex/`, or `off`)    |
| `:quiet`         | Toggle non-message output suppression                |
| `:verbose`       | Toggle frame-level metadata display                  |
| `:timestamps`    | Toggle ISO 8601 message timestamps                   |
| `:color <mode>`  | Set coloring mode: `on`, `off`, `auto`               |
| `:shortcuts`, `:keys` | List active keyboard bindings: `:shortcuts [-d]`     |
| `:source <f>`    | Execute a `.xwebs` script file                       |
| `:alias <n> <c>` | Create a command alias with positional args          |
| `:wait <dur>`    | Pause execution (e.g., `1s`, `500ms`)                |
| `:assert <ex>`   | Validate state with template expressions             |
| `:log <file>`    | Log traffic to JSONL (one object per line)           |
| `:record <f>`    | Start relative-time session recording                |
| `:replay <f>`    | Play back a session recording with timing            |
| `:mock <f>`      | Load YAML-based mock scenario                        |
| `:handlers`      | List all loaded handlers in priority order           |
| `:handler (add\|delete\|edit\|save)` | Manage handlers directly from the REPL (see details below) |
| `:prompt set <t>`| Customize the REPL prompt with Go templates          |

**Keyboard Shortcuts:**

The REPL supports customizable keyboard shortcuts for frequent operations. These are defined in your `.xwebs.yaml` configuration file.

| Shortcut | Default Command |
|----------|-----------------|
| `Ctrl+O` | `:shell --yes`  |

You can add your own shortcuts to perform commands or insert common prefixes:

```yaml
# ~/.xwebs.yaml
repl:
  shortcuts:
    "Ctrl+A": ":status"
    "Ctrl+K": ":clear"
    "Ctrl+Q": ":quit"
    "Ctrl+S": ":send "   # Note the trailing space; cursor is positioned at the end
```

Run `:shortcuts` inside the REPL to see all active custom bindings, or `:shortcuts -d` to see the default `readline` key bindings.

**Multi-line Input:**

For complex payloads or templates, you can use the backslash `\` as a line continuation character. The REPL will switch to a continuation prompt (`... `) and preserve your indentation until the message is complete.

```text
> { \
...   "event": "update", \
...   "data": { \
...     "id": 1 \
...   } \
... }
< {"status":"received"}
```

**Heredoc Input:**

For large payloads or scripts, you can use `<<EOF` style heredoc. The REPL will switch to a continuation prompt until the delimiter is matched on a line by itself.

```text
> :write config.json <<EOF
{
  "user": "{{.Vars.user}}",
  "token": "{{.Env.AUTH_TOKEN}}"
}
EOF
Written 56 bytes to config.json
```

**Advanced Sending Examples:**

```text
# Sending Binary (Hex)
> :sendb 48656c6c6f
< Hello

# Sending JSON (Validated)
> :sendj {"event":"login","id":123}
< {"status":"ok"}

# Sending Templates (Dynamic)
> :set user alice
> :sendt hello {{ .Session.user }}
< hello alice
```

**Interactive Session Example:**

```text
Connecting to: wss://echo.websocket.org
Successfully connected to wss://echo.websocket.org

> :status
Connection Status:
  URL:            wss://echo.websocket.org
  Subprotocol:    
  Compression:    false
  Closed:         false

> Hello, xwebs!
< Hello, xwebs!

> :exit
```

**Non-Interactive & Automation Mode:**

`xwebs` features a powerful non-interactive mode designed for scripts, CI/CD pipelines, and automated testing. You can send messages, wait for specific responses, and exit based on conditions without ever entering the REPL.

**Automation Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--send <msg>` | Send a message upon connection (repeatable) | `--send '{"type":"ping"}'` |
| `--input <file>` | Send content from a file upon connection | `--input request.json` |
| `--expect <pat>` | Wait for a message matching Regex or JQ (repeatable) | `--expect '/ready/'` |
| `--until <pat>` | Exit as soon as a message matches this pattern | `--until '.status == "success"'` |
| `--once` | Exit after the first message is received | `--once` |
| `--jsonl` | Output all traffic as machine-readable JSONL | `--jsonl` |
| `--output <file>` | Redirect formatted output to a file | `--output results.log` |
| `--timeout <dur>` | Set a global timeout for the entire session | `--timeout 30s` |
| `--watch <pat>` | Keep connection open and print only matches | `--watch '.event == "trade"'` |
| `--interactive`, `-i` | Force interactive REPL mode | `--interactive` |
| `--no-interact`, `-I` | Disable interactive REPL mode | `--no-interact` |

**Examples:**

```bash
# Send a login message and wait for a success response
xwebs connect wss://api.example.com --send '{"type":"login"}' --expect '.status == "success"' --once

# Pipe a file to a connection and capture the first response
cat request.json | xwebs connect wss://echo.websocket.org --once

# Monitor a stream for specific events and exit on a match
xwebs connect wss://stream.example.com --until '.type == "shutdown"'

# Health check with timeout and exit code
xwebs connect wss://api.example.com --expect '/ready/' --timeout 5s || exit 1
```

**Unix Pipelines:**

`xwebs` is designed to be a "good citizen" in Unix pipelines. When `stdout` is redirected or piped (non-TTY):
- **Clean Output**: Automatic suppression of direction indicators (⬆/⬇), timestamps, and status messages on `stdout`.
- **Metadata Redirection**: Handshake details and informational logs are automatically redirected to `stderr`.
- **Exclusive Data Stream**: Only received messages are written to `stdout` by default (sent messages are suppressed unless `--verbose` is used).

```bash
# Seamless integration with jq
xwebs connect wss://api.example.com --once | jq .status

# Process a stream of events
xwebs connect wss://stream.example.com | grep "ERROR" | tee errors.log
```

### Server Mode

The `serve` command transforms `xwebs` into a WebSocket server. You can host multiple endpoints, load handlers to automate responses, and manage the server lifecycle gracefully.

**Features:**
- **Multi-Path Routing**: Listen on one or more paths using repeatable `--path` flags (e.g., `--path /ws --path /events`).
- **Path Normalization**: Paths are automatically normalized to ensure they start with a leading slash `/`.
- **Strict Routing**: Attempting to connect to an unconfigured path results in a `404 Not Found` response, ensuring clean API boundaries.
- **Handler Integration**: Load declarative YAML handlers to build reactive services.
- **Dynamic Variable Injection**: Pass variables to handlers via configuration.
- **Graceful Shutdown**: Close all active connections cleanly on termination.
- **HTTP Status Page**: Clean landing page for standard browser requests showing uptime and active WebSocket paths.
- **Embedded Web UI**: Modern React dashboard available when started with the `--ui` flag.
- **Interactive Admin REPL**: Manage clients and server state in real-time using the `--interactive` flag.
- **Developer Observability**: Detailed logging of connection handshakes and upgrades when `--verbose` is enabled.

**Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--port`, `-p` | Port to listen on (default: `8080`) | `--port 9000` |
| `--path` | WebSocket path(s) to listen on (repeatable, default: `/`) | `--path /ws --path /events` |
| `--handlers` | Path to handler configuration YAML | `--handlers echo.yaml` |
| `--on` | Define a quick inline handler (`pattern :: run:cmd`) | `--on "ping :: respond:pong"` |
| `--on-match` | Define a full inline JSON handler | `--on-match '{"match":...}'` |
| `--respond` | Default response template for inline handlers | `--respond "OK"` |
| `--tls` | Enable TLS (HTTPS/WSS) | `--tls` |
| `--cert` | Path to certificate file | `--cert server.crt` |
| `--key` | Path to private key file | `--key server.key` |
| `--allowed-origins` | Allowed origins for WebSocket connections | `--allowed-origins https://example.com` |
| `--allow-ip` | Allowed IP addresses or CIDR ranges | `--allow-ip 127.0.0.1 --allow-ip 192.168.1.0/24` |
| `--deny-ip` | Denied IP addresses or CIDR ranges | `--deny-ip 1.2.3.4` |
| `--rate-limit` | Rate limit (per-client[,global]) | `--rate-limit 10/s` or `--rate-limit 10/s,100/s` |
| `--ui` | Enable the embedded React web UI at `/` | `--ui` |
| `--interactive`, `-i` | Enable interactive admin REPL | `--interactive` |
| `--no-interact`, `-I` | Disable interactive admin REPL | `--no-interact` |

**TLS Support (HTTPS/WSS):**
xwebs can host secure servers using standard TLS certificates. This is essential for production environments or when the backend requires `wss://`.

```bash
# Start a secure server using provided certificates
xwebs serve --tls --cert certs/server.crt --key certs/server.key --port 8443

# Connect to the secure server (using --insecure for self-signed certs)
xwebs connect wss://localhost:8443/ --insecure
```

When TLS is enabled:
- The server listens for HTTPS and WSS connections.
- The interactive status page indicates the secure status with a blue "RUNNING" pill.
- Recommended connection strings automatically use the `wss://` scheme.

**Security Controls:**
xwebs provides powerful built-in security controls to protect your server from unauthorized access and abuse.

- **IP Filtering**: Restriction via `--allow-ip` and `--deny-ip` supporting individual IPs and CIDR ranges.
- **Origin Restriction**: Ensure only trusted web applications can connect via `--allowed-origins`.
- **Rate Limiting**: Protect against DoS attacks with global and per-client rate limits using the token bucket algorithm. Use formats like `10/s`, `100/m`, or `5/h`.

```bash
# Start a server restricted to local network and specific origin, with rate limiting
xwebs serve --allow-ip 192.168.1.0/24 --allowed-origins https://myapp.example.com --rate-limit 5/s,50/s
```

**Examples:**

```bash
# Start a basic echo server on port 8080 (listens on /)
xwebs serve

# Start a server on port 9000 with multiple paths
xwebs serve --port 9000 --path /ws --path /events

# Paths are normalized automatically (ws2 becomes /ws2)
xwebs serve --path /ws1 --path ws2

# Start a server with custom handlers
xwebs serve --handlers examples/echo.yaml --verbose
```

**HTTP Status & Diagnostics:**
When the server is running, any standard HTTP (non-WebSocket) request to a registered path or the root `/` will return a status page. This page provides:
- Server status (RUNNING)
- Uptime duration
- Count of active WebSocket connections
- List of registered WebSocket paths

For machine-to-machine monitoring, you can request JSON by setting the `Accept: application/json` header:
```bash
curl -H "Accept: application/json" http://localhost:8080/
```

**Verbose Logging:**
When running with `--verbose`, the server logs detailed information about every connection attempt:
- `[http] connection received: GET /ws from [::1]:61234 (non-websocket)`
- `[http] attempting websocket upgrade for /ws from [::1]:61235`
- `[ws] upgrade successful for /ws from [::1]:61235`

**Graceful Shutdown:**
When the server receives a termination signal (e.g., `Ctrl+C`), it:
1. Stops accepting new connections.
2. Closes all active WebSocket sessions with a normal closure code (1000).
3. Waits for handlers to complete their current step before exiting.

**REST API:**
When the server is running, you can interact with it via a built-in REST API. This is useful for programmatic management and monitoring.

- **Status**: `GET /api/status` — Returns server uptime, connections, and paths.
- **Health**: `GET /api/health` — Returns basic health status, uptime, and component status (always available).
- **Clients**: `GET /api/clients` — Returns a list of active connection metadata.
- **Handlers (CRUD)**:
  - `GET /api/handlers` — List all loaded handlers.
  - `POST /api/handlers` — Create a new handler from JSON.
  - `PUT /api/handlers/{name}` — Update an existing handler.
  - `DELETE /api/handlers/{name}` — Delete a handler.
- **Key-Value Store**:
  - `GET /api/kv` — List all keys and values.
  - `GET /api/kv/{key}` — Get the value for a specific key.
  - `POST /api/kv/{key}` — Set a value for a specific key.
  - `DELETE /api/kv/{key}` — Delete a key.
- **Metrics**: `GET /api/metrics` — Returns Prometheus-compatible metrics (when enabled).

**API Flags:**

| Flag | Description | Example |
|------|-------------|---------|
| `--api` | Enable the REST API (default: `true`) | `--api=false` |
| `--metrics` | Enable Prometheus metrics (default: `false`) | `--metrics` |

**Examples:**

```bash
# Get server status
curl http://localhost:8080/api/status

# List connected clients
curl http://localhost:8080/api/clients

# Create a dynamic handler
curl -X POST -d '{"name": "api-pong", "match": {"pattern": "ping"}, "respond": "pong"}' http://localhost:8080/api/handlers

# Set a value in the KV store
curl -X POST -d '"my-value"' http://localhost:8080/api/kv/my-key

# Delete a key
curl -X DELETE http://localhost:8080/api/kv/my-key
```

**Server REPL Commands:**

When started with `--interactive` (or `-i`), the server provides a dedicated set of administrative commands:

| Command | Description |
|---------|-------------|
| `:status` | Show server status, uptime, and client count |
| `:drain` | Gracefully stop accepting connections and wait for existing ones to close |
| `:pause` | Temporarily pause message processing (incoming messages will be buffered) |
| `:resume` | Resume normal message processing and flush buffered messages |
| `:clients` | List all connected clients with ID, address, uptime, and message counts |
| `:client <id>` | Show detailed metadata for a specific client |
| `:send [flags] <id> <msg>` | Send message to specific client (`-j` JSON, `-t` Template, `-b` Binary) |
| `:broadcast [flags] <msg>` | Send message to all connected clients (`-j`, `-t`, `-b`) |
| `:kick <id> [c] [r]`| Disconnect a client with optional close code and reason |
| `:handlers` | List all registered server-side handlers with execution statistics (Matches, Latency, Errors) |
| `:handler (add\|delete\|edit\|rename\|save\|<name>)` | Manage handlers: `add <flags>` (see flags below), `delete <name>`, `edit [name]`, `rename <old> <new>`, `save [file] [--force\|-f]`, or show details for `<name>` |
| `:kv (list\|get\|set\|del)` | Manage the server-side key-value store. Subcommands: `list` (or `ls`), `get <key>`, `set <key> [-t\|-j] <val>`, `del <key>` |
| `:reload` | Hot-reload the handler configuration file and variables from disk without restarting the server |
| `:enable <name>` | Enable a previously disabled handler at runtime |
| `:disable <name>`| Disable a handler at runtime to stop it from matching incoming messages |
| `:stats` | Show global server observability statistics (connections, messages, handler hits) |
| `:slow [n]` | Show the top `n` slowest handler executions (default 10) |

> [!NOTE]
> Session-variable commands (`:set`, `:get`, `:vars`) are not available in server mode as there is no single client session context. Use [Key-Value Store commands](#key-value-store-commands) (e.g., `:kv set`, `:kv get`) for shared administrative state.

**`:handler add` flags:**

| Flag | Short | Description |
|------|-------|-------------|
| `--name <name>` | `-n` | Unique handler name (auto-generated if omitted) |
| `--match <pattern>` | `-m` | **(required)** Match pattern |
| `--match-type <type>` | `-t` | Match type: `glob`, `regex`, `jq`, `text`, etc. |
| `--priority <n>` | `-p` | Execution priority (higher runs first) |
| `--run <cmd>` | `-r` | Shell command to execute on match |
| `--respond <tmpl>` | `-R` | Response template sent back to the client |
| `--builtin <name>` | `-B` | Builtin action: `subscribe`, `unsubscribe`, or `publish` |
| `--topic <template>` | | Topic name template for builtin actions (required when `--builtin` is set) |
| `--exclusive` | `-e` | Stop further handler matching after this one fires |
| `--sequential` | `-s` | Run handler actions sequentially (default: concurrent) |
| `--rate-limit <limit>` | `-l` | Per-handler rate limit (e.g. `10/s`) |
| `--debounce <duration>` | `-d` | Debounce window (e.g. `500ms`) |

Example — add pub-sub handlers directly from the REPL:

```text
xwebs> :handler add -n handle-subscribe -m 'sub:*' -B subscribe --topic '{{.Message | trimPrefix "sub:"}}' -R 'subscribed:{{.Message | trimPrefix "sub:"}}'
xwebs> :handler add -n handle-unsubscribe -m 'unsub:*' -B unsubscribe --topic '{{.Message | trimPrefix "unsub:"}}' -R 'unsubscribed:{{.Message | trimPrefix "unsub:"}}'
xwebs> :handler add -n echo -m '*' -R 'echo:{{.Message}}'
```

**Topic / Pub-Sub Commands:**

| Command | Description |
|---------|-------------|
| `:topics` | List all active topics with subscriber counts and last-activity time |
| `:topic <name>` | Show per-subscriber detail: connection ID, remote address, time subscribed, messages sent |
| `:publish [-t] [--allow-empty] <topic> <msg>` | Fan-out a message to all subscribers. `-t` expands the message as a Go template first. `--allow-empty` sends even when no subscribers are present |
| `:subscribe <client-id> <topic>` | Manually subscribe a connected client to a topic from the REPL (creates topic if needed) |
| `:unsubscribe <client-id> <topic>` | Remove a client from a specific topic without disconnecting them |
| `:unsubscribe <client-id> --all` | Remove a client from every topic it is currently subscribed to |

Topics are created automatically when the first client subscribes and removed when the last subscriber leaves.  All topic commands support tab completion for both topic names and client IDs from live server state.

### Builtin Actions

`xwebs` includes several built-in actions that perform specialized tasks without requiring an external shell. These actions are formally defined via a registry, ensuring consistent behavior and **strict load-time validation**.

| Builtin | Scope | Description |
|---------|-------|-------------|
| `noop` | Shared | Does nothing; useful for testing or as a placeholder. |
| `subscribe` | Server | Subscribes the client connection to a pub/sub topic. |
| `unsubscribe` | Server | Unsubscribes the client from a pub/sub topic. |
| `publish` | Server | Broadcasts a message to all subscribers of a topic. |
| `kv-set` | Server | Sets a value in the shared server key-value store. |
| `kv-get` | Server | Retrieves a value from the KV store into `.KvValue`. |
| `kv-del` | Server | Deletes a key from the KV store. |

**Validation Features:**
- **Unknown Builtins**: Using an unknown builtin name in handler configuration causes an immediate startup error.
- **Mode Scoping**: Builtins are scoped to specific modes (Server/Client/Shared). For example, using `subscribe` in a client-mode handler will result in a validation error: `builtin "subscribe" is only available in server mode`.
- **Parameter Validation**: Each builtin validates its required configuration fields (e.g., `topic`, `key`) when the configuration is loaded, preventing runtime failures.

#### Pub-Sub Builtins (Server Only)

Clients subscribe to topics by sending messages that match a handler with `builtin: subscribe`. The `topic:` field is a Go template that resolves the topic name from the message:

```yaml
handlers:
  # Prefix-style: client sends "sub:trades"
  - name: handle-subscribe
    match: 'sub:*'
    builtin: subscribe
    topic: '{{.Message | trimPrefix "sub:"}}'
    respond: 'subscribed:{{.Message | trimPrefix "sub:"}}'

  - name: handle-unsubscribe
    match: 'unsub:*'
    builtin: unsubscribe
    topic: '{{.Message | trimPrefix "unsub:"}}'
    respond: 'unsubscribed:{{.Message | trimPrefix "unsub:"}}'

  # JSON-style: client sends {"type":"subscribe","channel":"trades"}
  - name: static-file
    match:
      jq: '.type == "download"'
    builtin: file-send                   # Send file as binary frame
    path: '{{.Message | jq ".path"}}'

  # Key-Value Store Examples
  - name: track-price
    match:
      jq: '.type == "price"'
    builtin: kv-set
    key: 'last:price:{{.Message | jq ".symbol"}}'
    value: '{{.Message | jq ".price"}}'
    respond: '{"ack":true}'

  - name: get-price
    match:
      jq: '.type == "get-price"'
    builtin: kv-get
    key: 'last:price:{{.Message | jq ".symbol"}}'
    respond: '{"symbol":"{{.Message | jq ".symbol"}}","price":"{{.KvValue}}"}'

  - name: delete-price
    match:
      jq: '.type == "del-price"'
    builtin: kv-del
    key: 'last:price:{{.Message | jq ".symbol"}}'
    respond: '{"deleted":true}'
```

Topic pub-sub example session:

```text
xwebs> :topics
TOPIC    SUBSCRIBERS  LAST ACTIVE
trades   2            5s ago
news     1            30s ago

2 topics, 3 total subscriptions

xwebs> :topic trades
Topic: trades
Subscribers: 2
  ID        REMOTE ADDR         SUBSCRIBED  MESSAGES SENT
  c-a1b2c3  192.168.1.10:51234  2m ago      44
  c-d4e5f6  10.0.0.5:60812      45s ago     12

xwebs> :publish trades {"symbol":"BTC","price":62000}
✓ Published to "trades" → 2 clients

xwebs> :publish -t trades {"symbol":"ETH","ts":"{{now | formatTime "RFC3339"}}"}
✓ Published to "trades" → 2 clients

xwebs> :subscribe c-d4e5f6 news
✓ c-d4e5f6 (10.0.0.5:60812) subscribed to "news"

xwebs> :unsubscribe c-a1b2c3 trades
✓ c-a1b2c3 removed from "trades" (1 subscriber remains)

xwebs> :unsubscribe c-d4e5f6 --all
✓ c-d4e5f6 removed from 2 topics: news, trades
```

Example administrative session:
```text
⚡XWEBS> :clients

Active Clients (1):
ID              Remote Address       Uptime       IN     OUT    Connected At
--------------------------------------------------------------------------------
conn-12345678   [::1]:64438          1m 12s       10     5      11:43:04

⚡XWEBS> :client conn-12345678
Client Information: conn-12345678
  Remote Address: [::1]:64438
  Connected At:   2026-04-13 11:43:04 (1m 12s ago)
  Messages IN:    10
  Messages OUT:   5

⚡XWEBS> :send conn-12345678 System update starting soon.
✓ Message sent to client conn-12345678

⚡XWEBS> :kick conn-12345678 1000 Maintenance
✓ Client conn-12345678 kicked
```

### Message Handlers

`xwebs` allows you to define declarative message handlers in a YAML configuration file. This is useful for building reactive WebSocket clients and servers without writing custom Go code.

**Key Features:**
- **Priority-Based Execution**: Handlers can be assigned a `priority` (higher numbers execute first). Handlers with the same priority run in the order they appear in the file.
- **Concurrency Control**: By default, multiple instances of the same handler can run in parallel if they match separate messages. Setting `concurrent: false` ensures that only one instance of a specific handler runs at a time, serializing execution for that handler name. This is crucial for preventing race conditions in stateful operations (e.g., updating a local file or global variable). Serialization is applied per-handler name and does not block unrelated handlers.
- **Match Conditions**: Match incoming messages by type (`text`, `json`, `regex`, `glob`, `jq`, `json_schema`, `template`) and pattern. You can also match by frame type using `binary: true` (for binary frames) or `binary: false` (for text frames). You can also use **shorthands** for concise matching: `match.regex: "pattern"`, `match.jq: "query"`, `match.json_path: "path"`, `match.json_schema: "path/to/schema.json"`, or `match.template: "expression"`. The `glob` matcher converts `*` and `?` to logical regexes, intuitively supporting full and substring matching across newlines and slashes. The `template` matcher evaluates a Go template and matches if the result is truthy (non-empty, non-false, non-zero).
- **Composite Matchers**: Combine multiple conditions using logical AND (`all`) and OR (`any`).
  - `match.all: [...]` requires **all** listed sub-matchers to match.
  - `match.any: [...]` requires **at least one** listed sub-matcher to match.
  - Composite matchers can be nested to create complex logic (e.g., `(A AND (B OR C))`).
- **Actions**: Trigger operations when a message matches or a lifecycle event occurs.
  - `shell`: Executes a shell command using `sh -c`.
    - `run`: The command to execute (preferred over `command`). Supports Go templates.
    - `command`: Fallback field for the command string.
    - `timeout`: Maximum execution time (e.g. `5s`, `1m`). Defaults to `30s`.
    - `env`: Map of environment variables for the command.
    - `silent`: If `false` (default), command output is logged to the terminal.
  - `send`: Sends a message back to the server.
    - `message`: The message content. Supports Go templates.
  - `log`: Appends a message to a file or stream.
    - `message`: The log entry. Supports Go templates.
    - `target`: Destination file path or `stdout`/`stderr`.
  - `builtin`: Executes a built-in `xwebs` command (REPL command).
- **Stdin Piping**: For `shell` actions, the raw incoming WebSocket message is automatically piped to the command's `stdin`.
- **Template Context**: Shell execution results are available to subsequent actions in the same handler via the `.Handler` object. For the `respond:` shorthand, these are also available as top-level variables:
  - `.Stdout` / `.Handler.Stdout`: Captured standard output.
  - `.Stderr` / `.Handler.Stderr`: Captured standard error.
  - `.ExitCode` / `.Handler.ExitCode`: Process exit code (0 for success, non-zero for failures, -1 for timeouts). Execution continues on shell failure to allow `respond:` to format the error context.
  - `.DurationMs` / `.Handler.Duration`: Time taken for execution (ms or parsed duration).
- **Pipeline Execution**: Define multi-step pipelines to chain shell operations and use intermediate results.
  - `pipeline`: A sequence of steps (`run` or `builtin`).
  - `as`: (Optional) Name the step to make its output and status available to later steps and the final `respond:` template via `{{.Steps.<name>.Stdout}}` and `{{.Steps.<name>.ExitCode}}`.
  - `ignore_error`: (Optional) If set to `true`, the pipeline will continue to execute even if the step's shell command returns a non-zero exit code. By default, pipelines stop execution if any step fails.
- **Lifecycle Events**: Bind actions to `on_connect`, `on_disconnect`, and `on_error` events. `on_connect` runs after a successful handshake; `on_disconnect` runs when the session terminates; `on_error` triggers on both connection failures (e.g. dial errors) and runtime handler execution failures. In `on_error` handlers, the specific error message is available via the `{{.Error}}` template variable.
- **Template Support**: All message and command fields support full Go templates with access to `.Msg`, `.Conn`, `.Vars`, `.Error`, etc.
- **Automatic Retries**: Configure handlers to automatically retry on failure with linear or exponential backoff.
  - `retry`:
    - `count`: Number of retry attempts (e.g., `3`).
    - `backoff`: Strategy, either `linear` or `exponential`. Defaults to `linear`.
    - `interval`: Initial wait duration (e.g., `1s`, `500ms`). Defaults to `1s`.
    - `max_interval`: Maximum wait time for exponential backoff (e.g., `30s`). Defaults to `30s`.
  - **Failure Definition**: A retry is triggered if a pipeline step fails (and `ignore_error` is false) or if a concise `run`/`builtin` command returns a non-zero exit code while `retry` is configured.
  - **Recovery**: If a retry succeeds, execution continues normally. If all retries fail, the handler terminates, and the optional `respond:` action is skipped for pipelines but executed for concise models (to match non-retry behavior).
- **Rate Limiting (Token Bucket)**: Prevent handler surges and protect system stability by limiting execution frequency.
  - `rate_limit`: A string specifying the maximum rate (e.g., `10/s`, `100/m`, `5/h`).
  - **Behavior**: Excess messages exceeding the rate limit are consistently **dropped** (not queued). In verbose mode, a warning is logged when a message is dropped.
  - **Independence**: Rate limits are applied independently per-handler name and are shared across all connections to protect global resources.
- **Debounce**: Consolidate rapid successive messages into a single execution.
  - `debounce`: A duration string (e.g., `500ms`, `1s`, `2s`).
  - **Behavior**: Implements **trailing-edge debounce**. The timer resets on each new matching message. Only the most recent message is processed when the quiet period ends.
  - **Scope**: Debounce timers are managed per-handler name and are global across all connections.
- **Exclusive Matching (Short-Circuiting)**: Setting `exclusive: true` on a handler ensures that if it matches, all subsequent handlers (lower priority) are skipped for that message. This is useful for high-priority "catch" handlers that should prevent a message from reaching more general-purpose handlers. Short-circuiting applies to both matching and execution, improving performance for complex handler sets.
- **Shell Sandboxing**: Restrict shell command execution to an explicit allowlist for enhanced security when handling untrusted messages or configurations.
  - `--sandbox`: Enables sandboxing mode.
  - `--allowlist`: Configures a comma-separated list of approved executables (e.g., `echo,ls,grep`).
  - **REPL Observability**: Use the `:handlers` command in the REPL to see the loaded handlers in their execution order.
- **Dynamic Handlers**: Manage handlers directly from the REPL without restarting or editing files using the `:handler` command.
  - **Add**: `:handler add --match <pattern> [flags]`
    - Supports short flags: `-m` (match), `-n` (name), `-t` (match-type), `-p` (priority), `-r` (run), `-R` (respond), `-B` (builtin), `--topic` (topic template), `-e` (exclusive), `-s` (sequential), `-l` (rate-limit), `-d` (debounce).
    - `-B, --builtin <name>` — set a builtin action (`subscribe`, `unsubscribe`, `publish`). Requires `--topic`.
    - `--topic <template>` — Go template that resolves the topic name for builtin actions (e.g. `'{{.Message | trimPrefix "sub:"}}'`).
    - If `--name` is omitted, a unique Docker-style name is generated (e.g., `distracted_lovelace`).
  - **Delete**: `:handler delete <name>`
  - **Rename**: `:handler rename <old-name> <new-name>`
  - **Edit**: `:handler edit [name]` — Opens the handler (or entire config if no name provided) in your `$EDITOR`.
  - **Save**: `:handler save [filename] [--force|-f]` — Persists all in-memory handlers and session variables to a YAML file. If `filename` is omitted, xwebs uses the path from `--handlers`; when that target already exists, REPL asks for overwrite confirmation unless forced.
  ```text
  > :handler add -m ping -R pong
  Handler "boring_wozniak" added successfully.
  > :handler edit boring_wozniak
  > :handler save handlers.yaml
  > :handler delete boring_wozniak
  ```



**Usage:**
```bash
# Load handlers from a YAML file
xwebs connect wss://echo.websocket.org --handlers examples/handlers/handlers.yaml
```

### Inline Handlers (CLI Flags)

For quick prototyping, dynamic responders, or simple automation, you can define handlers directly from the command line using `--on`, `--on-match`, and `--respond`. These work identically in both `connect` and `serve` modes.

#### Standardized Syntax (`--on`)
The `--on` flag uses a standardized `::` separator to define the relationship between a message pattern and its actions.

**Syntax:** `--on "<pattern> [:: <actions>]"`

- **Pattern Auto-Detection**: xwebs automatically detects the matcher type based on the expression:
    - `^` or `(` or `[` -> **Regex**
    - `.` or `$` -> **JQ** (JSON Query)
    - `*` or `?` -> **Glob**
    - Literal -> **Glob** (exact match)
- **Action Segments**: Multiple actions can be chained using the `::` separator:
    - `run:<cmd>` -> Execute a shell command.
    - `respond:<tmpl>` -> Send a response back.
    - `builtin:<cmd>` -> Execute a REPL command.
    - `timeout:<dur>` -> Set a custom timeout for the action.
    - `exclusive` -> Short-circuit subsequent handlers on match.

**Examples:**
```bash
# Basic glob echo (literal match)
xwebs connect wss://echo.websocket.org --on "ping :: respond:pong"

# Regex matcher with shared response template
xwebs connect wss://echo.websocket.org \
  --on "(?i)hello :: run:echo 'Greeter triggered'" \
  --respond "Hello! I am xwebs."
```

#### Important: Quoting and JQ Syntax
When using **JQ matchers** (expressions starting with `.`), you must follow JQ's string syntax rules:
1. **Double Quotes for JQ Strings**: JQ *requires* double quotes (`"`) for string literals. Single quotes (`'`) will cause a `parsing gojq query: unexpected token "'"` error.
2. **Shell Quoting**: To send double quotes to JQ from your terminal, you have two options:
   - **Outer Single Quotes**: Wrap the whole handler in single quotes and use raw double quotes inside: `' .type == "alert" :: ... '`
   - **Escaped Double Quotes**: Wrap the whole handler in double quotes and escape the inner double quotes: `" .type == \"alert\" :: ... "`

**Correct JQ Examples:**
```bash
# Preferred: Outer single quotes (cleanest)
xwebs connect URL --on '.type == "alert" :: run:echo Alert! :: respond:Acknowledged'

# Alternative: Escaped double quotes
xwebs connect URL --on ".type == \"alert\" :: run:echo Alert! :: respond:Acknowledged"
```

**Common Edge Cases:**
```bash
# JQ matching on JSON streams
xwebs connect wss://api.example.com --on '.event == "trade" :: run:./alert.sh'

# Multi-segment handlers
xwebs connect wss://echo.websocket.org \
  --on '.type == "ping" :: timeout:5s :: run:./process.sh :: respond:processed!'
```

#### JSON Handlers (`--on-match`)
For full control (retry logic, concurrency, debounce), you can define a complete handler using JSON syntax.

```bash
xwebs connect wss://echo.websocket.org \
  --on-match '{"name": "heavy", "match": {"pattern": "*"}, "concurrent": false, "run": "sleep 5"}'
```

#### Default Response (`--respond`)
Set a default response template for all matching inline handlers (those defined via `--on`) that do not have an explicit `respond:` segment.

```bash
xwebs serve --port 8080 --on "ping:echo pong" --respond "Auto-reply: {{.Stdout}}"
```


**Example `handlers.yaml`:**
```yaml
variables:
  log_file: "session.log"

handlers:
  - name: "priority_handler"
    priority: 100
    match:
      type: "text"
      pattern: "emergency"
    actions:
      - action: "log"
        message: "EMERGENCY MSG RECEIVED!"

  - name: "jq_matcher"
    match:
      jq: '.type == "release" and .env == "production"'
    actions:
      - action: "log"
        message: "Production release detected via JQ!"

  - name: "auto_ping"
    on_connect:
      - action: "shell"
        command: "echo 'Started session at {{now}}' > {{.Vars.log_file}}"
    match:
      type: "json"
      pattern: ".type == \"ping\""
    actions:
      - action: "send"
        message: '{"type": "pong", "ts": "{{nowUnix}}"}'

  - name: "json_path_matcher"
    match:
      json_path: "user.id"
      equals: 1001
    actions:
      - action: "log"
        message: "Matched user 1001 via JSONPath!"

  - name: "complex_template_matcher"
    match:
      template: '{{ contains "alert" .Message }}'
    actions:
      - action: log
        message: "ALERT DETECTED via template matching!"

    match:
      type: "text"
      pattern: "update_state"
    run: "./scripts/update_global_state.sh"
    respond: "State updated successfully"

  - name: "rate_limited_api"
    rate_limit: "1/s"
    match:
      type: "glob"
      pattern: "api/*"
    run: "./scripts/process_api.sh"
```

### Output Formatting & Filtering

`xwebs` provides advanced control over how messages are displayed in your terminal. This is purely a display concern and does not affect the data sent or received.

**Formatting Examples:**

```text
# Enable JSON pretty-printing for all incoming messages
> :format json

# Use a custom Go template for display
# Available: .Message, .MessageType, .Timestamp, .Direction, etc.
> :format template [{{ .Timestamp }}] {{ .Direction }} >> {{ .Message }}

# Toggle frame-level metadata (Opcode, Length, Compression)
> :verbose
```

**Filtering Examples:**

```text
# Show only messages where the 'event' field is 'update'
> :filter .event == "update"

# Show only messages containing the word "ERROR" (regex)
> :filter /ERROR/

# Clear the filter
> :filter off
```

**Non-Interactive Formatting:**

```bash
# Filter and format incoming messages in a pipe
xwebs connect wss://api.example.com --filter '.status == "healthy"' --format json
```

# With custom subprotocols
xwebs connect wss://echo.websocket.org --subprotocol v1.xwebs,mqtt

# Using an alias/bookmark (defined in config)
xwebs connect staging

# Automate a sequence of actions via script
xwebs connect wss://echo.websocket.org --script integration-test.xwebs

# Connect and log traffic immediately
```
xwebs connect wss://echo.websocket.org --log traffic.jsonl
```

### Observability & Testing

`xwebs` includes a complete suite of tools for monitoring, capturing, and simulating WebSocket traffic. 

#### JSONL Logging & Recording

- **Logging**: Capture every message and connection event in a machine-readable JSONL format. Logging captures raw, unformatted data regardless of display settings.
- **Recording**: Similar to logging, but captures relative timing, allowing for exact playback of test sessions.

```bash
# Start logging from the command line
xwebs connect wss://api.example.com --log session_debug.jsonl

# Record a session to use as a test fixture later
xwebs connect wss://echo.websocket.org --record my_recording.jsonl
```

#### Deterministic Replay

Play back any recorded session using the `:replay` command. `xwebs` automatically offsets session timing to start immediately and waits for a grace period at the end to capture responses.

```text
> :replay my_recording.jsonl
▶ Replaying messages from my_recording.jsonl...
  [142ms]  → {"action": "ping"}
⬇ {"status": "pong"}
✓ Replay complete. 1 sent, 1 received.
```

#### Scenario-Based Mocking

The mocking engine allows you to simulate complex server behaviors using YAML scenarios. Mocks support `gojq` pattern matching, delays, and server-initiated pushes (`after`).

```yaml
# examples/mocks/simple_greeting.yaml
scenarios:
  - name: "Greeting Mock"
    loop: true
    steps:
      - expect:
          jq: '.msg? | contains("hello")'
        respond: '{"status": "MOCKED", "reply": "Hello from xwebs!"}'
        delay: 500ms
```

To load a mock:
```text
> :mock examples/mocks/simple_greeting.yaml
✓ Mock loaded: examples/mocks/simple_greeting.yaml
```

### Automation & Scripting

`xwebs` features a powerful scripting subsystem that allows you to automate repetitive tasks, perform integration tests, and build repeatable WebSocket workflows.

#### Script Files (`.xwebs`)

A script file is a plain text file where each line is a REPL command. Lines starting with `#` are comments.

Example `health-check.xwebs`:
```text
# Login to the service
:sendj {"type": "login", "user": "test-bot"}

# Wait for the welcome message
:wait 500ms

# Assert that we are logged in and latency is low
:assert {{eq (.Last | jq ".status") "ok"}} "Login failed"
:assert {{lt .LastLatencyMs 100}} "High latency detected: {{.LastLatencyMs}}ms"

# Send a heartbeat and exit
:ping
:exit
```

#### Aliases with Arguments

Aliases support positional argument substitution (`$1`, `$2`, ..., `$@`), allowing you to create custom shorthand commands.

```text
# Define an alias
> :alias sendto :sendj {"to":"$1", "msg":"$2"}

# Use the alias
> :sendto "alice" "hello there"
# Expands to: :sendj {"to":"alice", "msg":"hello there"}
```

#### Assertions and Observability

The `:assert` command evaluates a Go template expression. If the expression evaluates to `false` or an empty string, the assertion fails. In `--script` mode, an assertion failure stops the script and causes `xwebs` to exit with code 1.

Templates have access to:
- `.Last` — The last received message (as a string, use `| jq` to parse)
- `.LastLatencyMs` — The round-trip time of the last message exchange in milliseconds
- `.Vars` — Session variables set via `:set`

### Custom Headers and Authentication

`xwebs` allows you to inject custom HTTP headers and authentication credentials into the WebSocket handshake. Values can be static or dynamic Go templates.

```bash
# Add multiple custom headers
xwebs connect wss://api.example.com -H "X-API-Key: 12345" -H "X-Client-ID: local-01"

# Use environment variables in headers via templates
xwebs connect wss://api.example.com -H "Authorization: Bearer {{ env \"AUTH_TOKEN\" }}"

# Dedicated flag for Bearer tokens
xwebs connect wss://api.example.com --token "my-secret-token"
xwebs connect wss://api.example.com --token "{{ .Env.AUTH_TOKEN }}"

# Dedicated flag for Basic Authentication (user:pass)
xwebs connect wss://api.example.com --auth "admin:password123"
xwebs connect wss://api.example.com --auth "admin:{{ .Env.PASS }}"
```

> [!TIP]
> When using templates in CLI flags, always wrap the value in single quotes to prevent your shell from interpreting the double curly braces `{{ }}`.

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
### Message Size Configuration

`xwebs` can enforce a maximum message size for both incoming and outgoing messages. This is useful for protecting against memory exhaustion from unexpectedly large payloads.

```bash
# Enforce a 1MB limit (1048576 bytes)
xwebs connect wss://api.example.com --max-message-size 1048576
```

Message size limits can also be defined in bookmarks:

```yaml
bookmarks:
  limited-service:
    url: "wss://api.example.com"
    max-message-size: 65536 # 64KB
```

### Compression Configuration

`xwebs` supports per-message-deflate compression (`RFC 7692`) to reduce bandwidth usage for large message payloads.

```bash
# Enable compression for the connection
xwebs connect wss://echo.websocket.org --compress
```

Compression settings can also be defined in bookmarks:

```yaml
bookmarks:
  compressed-service:
    url: "wss://api.example.com"
    compress: true
```

 
### Fragmentation Configuration
 
`xwebs` supports automatic message fragmentation for outgoing messages that exceed a specified frame size. This is useful for splitting large payloads into smaller frames to avoid overwhelming network buffers or the remote endpoint. Incoming fragmented messages are automatically reassembled.
 
```bash
 # Fragment outgoing messages into 10KB frames (10240 bytes)
 xwebs connect wss://api.example.com --max-frame-size 10240
```
 
Fragmentation settings can also be defined in bookmarks:
 
```yaml
 bookmarks:
   large-payloads:
     url: "wss://api.example.com"
     max-frame-size: 16384 # 16KB
```
 
### Template Engine

`xwebs` includes a powerful Go template engine that can be used in handlers, CLI flags, and the REPL to create dynamic messages and shell commands.

#### String Functions

The following string manipulation functions are available in templates:

| Function       | Description                             | Example                                           |
|----------------|-----------------------------------------|---------------------------------------------------|
| `upper`        | Converts string to uppercase            | `{{ .msg \| upper }}`                             |
| `lower`        | Converts string to lowercase            | `{{ .msg \| lower }}`                             |
| `trim`         | Trims leading/trailing whitespace       | `{{ .msg \| trim }}`                              |
| `replace`      | Replaces all occurrences of a string    | `{{ .msg \| replace "old" "new" }}`               |
| `split`        | Splits a string into a list             | `{{ .msg \| split " " }}`                         |
| `join`         | Joins a list into a string              | `{{ .list \| join "," }}`                         |
| `contains`     | Checks if a string contains a substring | `{{ if contains "error" .msg }}...{{ end }}`      |
| `regexMatch`   | Checks if a string matches a regex      | `{{ if regexMatch "^[0-9]+$" .msg }}...{{ end }}` |
| `regexFind`    | Finds the first regex match             | `{{ .msg  \| regexFind "[a-z]+" }}`               |
| `regexReplace` | Replaces regex matches                  | `{{ .msg  \| regexReplace "[0-9]+" "#" }}`        |
| `shellEscape`  | Escapes a string for safe shell use     | `{{ .msg  \| shellEscape }}`                      |
| `urlEncode`    | URL encodes a string                    | `{{ .msg  \| urlEncode }}`                        |
| `quote`        | Wraps a string in double quotes         | `{{ .msg  \| quote }}`                            |
| `truncate`     | Truncates a string with ellipsis        | `{{ .msg  \| truncate 10 }}`                      |
| `short`        | Truncates a string to 8 chars (no ...)  | `{{ .msg  \| short }}`                             |
| `red`          | Wraps string in ANSI red                | `{{ .msg \| red }}`                               |
| `green`        | Wraps string in ANSI green              | `{{ .msg \| green }}`                             |
| `yellow`       | Wraps string in ANSI yellow             | `{{ .msg \| yellow }}`                            |
| `blue`         | Wraps string in ANSI blue               | `{{ .msg \| blue }}`                              |
| `magenta`      | Wraps string in ANSI magenta            | `{{ .msg \| magenta }}`                           |
| `cyan`         | Wraps string in ANSI cyan               | `{{ .msg \| cyan }}`                              |
| `bold`         | Wraps string in ANSI bold               | `{{ .msg \| bold }}`                              |
| `dim`          | Wraps string in ANSI dim/faint          | `{{ .msg \| dim }}`                               |
| `underline`    | Wraps string in ANSI underline          | `{{ .msg \| underline }}`                         |
| `reset`        | Returns ANSI reset code                 | `{{ reset }}`                                     |
| `padLeft`      | Pads a string on the left               | `{{ .msg \| padLeft 10 }}`                        |
| `padRight`     | Pads a string on the right              | `{{ .msg \| padRight 10 }}`                       |
| `indent`       | Indents every line with spaces          | `{{ .msg \| indent 2 }}`                          |

#### JSON Functions

| Function       | Description                             | Example                                           |
|----------------|-----------------------------------------|---------------------------------------------------|
| `toJSON`       | Marshals a value to JSON string         | `{{ dict "a" 1 \| toJSON }}`                      |
| `fromJSON`     | Unmarshals a JSON string to a value     | `{{ (fromJSON .msg).key }}`                       |
| `prettyJSON`   | Marshals to indented JSON string        | `{{ .data \| prettyJSON }}`                       |
| `compactJSON`  | Removes whitespace from JSON string     | `{{ .msg \| compactJSON }}`                       |
| `isJSON`       | Checks if a string is valid JSON        | `{{ if isJSON .msg }}...{{ end }}`                |
| `jq`           | Executes a JQ query on a value          | `{{ jq ".foo[0].bar" .msg }}`                     |
| `jsonPath`     | Alias for `jq` for dot-path access      | `{{ jsonPath ".user.id" .msg }}`                  |
| `mergeJSON`    | Merges two JSON objects/strings         | `{{ mergeJSON .base .overlay }}`                  |
| `setJSON`      | Sets a field in a JSON object           | `{{ setJSON "key" "val" .obj }}`                  |
| `deleteJSON`   | Deletes a field from a JSON object      | `{{ deleteJSON "key" .obj }}`                     |

#### Connection & Session Context Functions

The following functions provide access to connection-level metadata and session state. These are particularly useful in custom REPL prompts (`:prompt`) and reactive handlers.

| Function         | Description                                    | Example                                    |
|------------------|------------------------------------------------|--------------------------------------------|
| `connID`         | Returns unique ID for the current connection   | `{{ connID }}`                             |
| `shortConnID`    | Returns first 8 chars of connection ID         | `{{ shortConnID }}`                        |
| `sessionID`      | Returns unique ID for the user session         | `{{ sessionID }}`                          |
| `clientIP`       | Returns the client's public IP address         | `{{ clientIP }}`                           |
| `remoteAddr`     | Returns the remote host:port                   | `{{ remoteAddr }}`                         |
| `localAddr`      | Returns the local host:port                    | `{{ localAddr }}`                          |
| `subprotocol`    | Returns the negotiated WebSocket subprotocol   | `{{ subprotocol }}`                        |
| `uptime`         | Returns duration since connection opened       | `{{ uptime }}`                             |
| `connectedSince` | Returns timestamp of connection start          | `{{ connectedSince \| date "15:04" }}`     |
| `msgsIn`         | Total messages received in current session     | `{{ msgsIn }}↓`                            |
| `msgsOut`        | Total messages sent in current session         | `{{ msgsOut }}↑`                           |
| `messageCount`   | Total messages (In + Out)                      | `{{ messageCount }}`                       |
| `mode`           | Current operating mode (client/server)         | `{{ mode }}`                               |
| `status`         | Connection state (connected, reconnecting, etc)| `{{ status }}`                             |
| `port`           | Current connection numeric port                | `{{ port }}`                               |
| `path`           | WebSocket endpoint path                        | `{{ path }}`                               |
| `tls`            | Security symbol (🔒 for TLS, empty for plain)   | `{{ tls }}`                               |
| `secure`         | Boolean indicator for TLS/SSL                  | `{{ secure }}`                             |
| `reconnectCount` | Current reconnection attempt number            | `{{ reconnectCount }}`                     |
| `lastMsgAgo`     | Time since last received message               | `{{ lastMsgAgo }} ago`                     |
| `lastSendAgo`    | Time since last sent message                   | `{{ lastSendAgo }} ago`                    |
| `rtt`            | Last measured Round Trip Time (via ping/pong)  | `{{ rtt }}`                                |
| `avgRtt`         | Moving average Round Trip Time                 | `{{ avgRtt }}`                             |
| `handlerHits`    | Total number of handler executions             | `{{ handlerHits }}`                        |
| `activeHandlers` | Number of currently executing handlers         | `{{ activeHandlers }}`                     |
| `sessionSet`     | Sets a session variable                        | `{{ sessionSet "key" "val" }}`             |
| `sessionGet`     | Gets a session variable                        | `{{ sessionGet "key" }}`                   |
| `sessionClear`   | Clears all session variables                  | `{{ sessionClear }}`                       |
| `clientCount`    | Returns current connected client count (server) | `{{ clientCount }}`                        |
| `serverUptime`   | Returns total server uptime duration           | `{{ serverUptime }}`                       |

#### Server Context Variables
When running in **Server Mode** (`xwebs serve`), handlers have access to global server state via several variables. These are available at the root level of all templates.

| Variable           | Description                                  | Example                                           |
|--------------------|----------------------------------------------|---------------------------------------------------|
| `{{.ClientCount}}` | Total number of currently connected clients   | `Connected: {{.ClientCount}} users`               |
| `{{.Clients}}`     | List of active client metadata objects       | `{{range .Clients}}{{.ID}} ({{.RemoteAddr}}){{end}}` |
| `{{.ServerUptime}}`| Server uptime as a Go `time.Duration`        | `Up for {{.ServerUptime}}`                        |
| `{{.ServerUptimeStr}}`| Formatted uptime string (e.g., `1h2m3s`)    | `Uptime: {{.ServerUptimeStr}}`                    |
| `{{.RemoteAddr}}`  | The remote address of the specific client     | `Your IP: {{.RemoteAddr}}`                        |

**Client Metadata Fields (`.Clients`):**
- `.ID`: Unique connection identifier (e.g., `conn-123456`).
- `.RemoteAddr`: The IP address and port of the client.
- `.ConnectedAt`: The timestamp when the client connected.
- `.Uptime`: Duration the client has been connected.
- `.UptimeStr`: Formatted connection duration.

#### Encoding Functions

| Function        | Description                             | Example                                           |
|-----------------|-----------------------------------------|---------------------------------------------------|
| `base64Encode`  | Base64 encodes a string                 | `{{ .msg \| base64Encode }}`                      |
| `base64Decode`  | Base64 decodes a string                 | `{{ .msg \| base64Decode }}`                      |
| `hexEncode`     | Hex encodes a string                    | `{{ .msg \| hexEncode }}`                         |
| `hexDecode`     | Hex decodes a string                    | `{{ .msg \| hexDecode }}`                         |
| `gzip`          | Gzip compresses a string                | `{{ .msg \| gzip }}`                              |
| `gunzip`        | Gunzip decompresses a string            | `{{ .msg \| gunzip }}`                            |

#### Crypto Functions

| Function      | Description                     | Example                                |
|---------------|---------------------------------|----------------------------------------|
| `md5`         | Calculates MD5 hash (hex)       | `{{ .msg \| md5 }}`                    |
| `sha256`      | Calculates SHA256 hash (hex)    | `{{ .msg \| sha256 }}`                 |
| `sha512`      | Calculates SHA512 hash (hex)    | `{{ .msg \| sha512 }}`                 |
| `hmacSHA256`  | Calculates HMAC-SHA256 (hex)    | `{{ hmacSHA256 "key" .msg }}`          |
| `jwt`         | Decodes JWT claims (unverified) | `{{ (jwt .token).sub }}`               |
| `randomBytes` | Generates N random bytes        | `{{ randomBytes 16 \| base64Encode }}` |

#### Time Functions

| Function        | Description                             | Example                                           |
|-----------------|-----------------------------------------|---------------------------------------------------|
| `now`           | Returns current time object             | `{{ now.Year }}`                                  |
| `time`          | Current time (HH:MM:SS)                 | `{{ time }}` -> `22:08:16`                        |
| `shortTime`     | Current time (HH:MM)                    | `{{ shortTime }}` -> `22:08`                      |
| `date`          | Current date (YYYY-MM-DD)               | `{{ date }}` -> `2026-04-09`                      |
| `isoTime`       | Current time in ISO 8601 / RFC3339      | `{{ isoTime }}`                                   |
| `weekday`       | Current day of the week                 | `{{ weekday }}` -> `Thursday`                     |
| `hour`          | Current hour (24h)                      | `{{ hour }}` -> `22`                              |
| `minute`        | Current minute                          | `{{ minute }}` -> `08`                            |
| `unix`          | Current Unix timestamp (seconds)        | `{{ unix }}`                                      |
| `unixMilli`     | Current Unix timestamp (milliseconds)   | `{{ unixMilli }}`                                 |
| `elapsed`       | Concise process uptime                  | `{{ elapsed }}` -> `1h2m3s`                       |
| `uptime`        | Process uptime (Duration object)        | `{{ uptime }}`                                    |
| `formatTime`    | Formats a time object or timestamp      | `{{ formatTime "2006-01-02" .t }}`                |
| `parseTime`     | Parses a time string                    | `{{ parseTime "2006-01-02" "2023-01-01" }}`       |
| `duration`      | Parses a duration string                | `{{ duration "1h30m" }}`                          |
| `since`         | Calculated duration since time          | `{{ since .start }}`                              |

#### Color and Style Functions

The template engine includes ANSI color and style helpers, perfect for `:prompt` customization or formatted logging.

| Function       | Description                             | Example                                           |
|----------------|-----------------------------------------|---------------------------------------------------|
| `color`        | Wrap text in a specific color/code      | `{{ color "red" "text" }}` or `{{ color 31 "t" }}`|
| `red`, `green` | Shorthand for common colors             | `{{ red "error" }}`, `{{ green "success" }}`      |
| `yellow`, `blue`| (Also: `black`, `magenta`, `cyan`, `white`, `grey`, `dim`) | `{{ yellow "warning" }}`                  |
| `bold`, `italic`| Shorthand for text styles               | `{{ bold "strong" }}`, `{{ italic "focus" }}`     |
| `underline`    | (Also: `faint`, `inverse`)              | `{{ underline "link" }}`                          |
| `reset`        | Returns the ANSI reset code             | `{{ reset }}`                                     |

#### Prompt Customization

You can dynamically change the REPL prompt using the `:prompt set` command. The prompt supports all Go template functions and has access to connection context.

**Available Prompt Variables:**
- `.Host` — The remote host name (e.g., `echo.websocket.org`)
- `.ConnectionID` — The unique ID for the current session
- `.Vars` — Access to all session variables set via `:set`
- `.Time` — Access to standard time functions

**Examples:**
```text
# Simple colored host
> :prompt set "{{green .Host}} >> "

# Complex context-aware prompt
> :set user bob
> :prompt set "{{bold .ConnectionID}} @ {{cyan .Vars.user}} > "

# Time-stamped Prompt
> :prompt set "{{grey (formatTime \"15:04:05\" now)}} {{green .Host}} $ "

# Identity & Role (Context Aware)
> :set role admin
> :prompt set "{{bold (upper .Vars.role)}}[{{cyan .Vars.user}}] > "

# Minimalist Lambda Style
> :prompt set "{{magenta \"λ\"}} {{faint .ConnectionID}} » "

# Status Indicator Style
> :set status ready
> :prompt set "{{green \"●\"}} {{.Host}} {{yellow .Vars.status}} ❯ "

# Branded Console
> :prompt set "[xwebs:{{blue .ConnectionID}}] {{italic \"active\"}} → "
```


#### Math Functions

| Function | Description                             | Example                               |
|----------|-----------------------------------------|---------------------------------------|
| `add`    | Adds two numbers                        | `{{ add 1 2 }}`                       |
| `sub`    | Subtracts two numbers                   | `{{ sub 10 5 }}`                      |
| `mul`    | Multiplies two numbers                  | `{{ mul 2 3 }}`                       |
| `div`    | Divides two numbers                     | `{{ div 10 2 }}`                      |
| `mod`    | Modulo of two integers                  | `{{ mod 10 3 }}`                      |
| `max`    | Max of two numbers                      | `{{ max 5 10 }}`                      |
| `min`    | Min of two numbers                      | `{{ min 5 10 }}`                      |
| `seq`    | Generates a sequence of integers        | `{{ range seq 1 5 }}{{.}}{{ end }}`   |
| `random` | Generates a random integer in [min,max) | `{{ random 1 100 }}`                  |

#### System Functions

| Function       | Description                     | Example                                   |
|----------------|---------------------------------|-------------------------------------------|
| `env`          | Returns an environment variable | `{{ env "HOME" }}`                        |
| `shell`        | Executes a shell command        | `{{ shell "ls -l" }}`                     |
| `hostname`     | Returns the system hostname     | `{{ hostname }}`                          |
| `user`         | Returns the current username    | `{{ user }}`                              |
| `home`         | Returns user home directory     | `{{ home }}`                              |
| `pid`          | Returns the process ID          | `{{ pid }}`                               |
| `tty`          | Returns session TTY             | `{{ tty }}`                               |
| `xwebsVersion` | Returns xwebs version           | `{{ xwebsVersion }}`                      |
| `cpuUsage`     | CPU utilization percentage      | `{{ cpuUsage }}`                          |
| `memUsage`     | System memory usage             | `{{ memUsage }}`                          |
| `diskUsage`    | Root disk usage                 | `{{ diskUsage }}`                         |
| `fileRead`     | Reads a file's content          | `{{ fileRead "config.json" }}`            |
| `fileExists`   | Checks if a file exists         | `{{ if fileExists "a.txt" }}...{{ end }}` |

> [!NOTE]
> These functions can be disabled for security using the `--no-shell-func` global flag. See [Template Sandboxing](#template-sandboxing) for details.

#### Template Sandboxing

For security-sensitive environments or when running untrusted configurations, you can enable template sandboxing using the `--no-shell-func` flag (or `XWEBS_NO_SHELL_FUNC=true` environment variable). 

When enabled, the following functions are restricted and will return an error if called:
- `env`, `shell`, `fileRead`, `fileExists`, `glob`, `hostname`, `pid`, `cwd`, `tempFile`
- `user`, `home`, `tty`, `cpuUsage`, `memUsage`, `diskUsage`, `xwebsVersion`

Safe functions (string manipulation, JSON processing, math, time, encoding, cryptography, etc.) continue to work normally in sandbox mode.

#### ID Functions

| Function  | Description                             | Example                               |
|-----------|-----------------------------------------|---------------------------------------|
| `uuid`    | Generates a UUID v4                     | `{{ uuid }}`                          |
| `ulid`    | Generates a ULID                        | `{{ ulid }}`                          |
| `nanoid`  | Generates a NanoID                      | `{{ nanoid }}`                        |
| `shortid` | Generates a ShortID                     | `{{ shortid }}`                       |
| `counter` | Returns/increments a named counter      | `{{ counter "msg_id" }}`              |
| `reqCounter`| Increments/returns request counter     | `{{ reqCounter }}`                    |
| `msgCounter`| Increments/returns message counter     | `{{ msgCounter }}`                    |
| `errorCount`| Increments/returns error counter       | `{{ errorCount }}`                    |
| `seq`       | Increments/returns generic sequence    | `{{ seq }}`                           |

#### Fake Data (Faker) Functions

The template engine includes basic fake data generation helpers, useful for creating realistic test payloads and messages.

| Function        | Description                             | Example                               |
|-----------------|-----------------------------------------|---------------------------------------|
| `fakeName`      | Generates a full name                   | `{{ fakeName }}`                      |
| `fakeFirstName` | Generates a first name                  | `{{ fakeFirstName }}`                 |
| `fakeLastName`  | Generates a last name                   | `{{ fakeLastName }}`                  |
| `fakeEmail`     | Generates a fake email address          | `{{ fakeEmail }}`                     |
| `fakeUsername`  | Generates a fake username               | `{{ fakeUsername }}`                  |
| `fakePhone`     | Generates a fake phone number           | `{{ fakePhone }}`                     |
| `fakeCompany`   | Generates a fake company name           | `{{ fakeCompany }}`                   |
| `fakeCompanyCatchPhrase`| Generates a fake catch phrase  | `{{ fakeCompanyCatchPhrase }}`        |
| `fakeProductName`| Generates a fake product name          | `{{ fakeProductName }}`               |
| `fakeProductCategory`| Generates a random product category| `{{ fakeProductCategory }}`           |
| `fakeColor`     | Generates a random color name           | `{{ fakeColor }}`                     |
| `fakePrice`     | Generates a fake price (optional range) | `{{ fakePrice 10.0 100.0 }}`          |
| `fakeAmount`    | Generates a fake amount (optional range)| `{{ fakeAmount 10.0 100.0 }}`         |
| `fakeCurrency`  | Generates a fake currency code          | `{{ fakeCurrency }}`                  |
| `fakeCreditCard`| Generates a fake credit card number     | `{{ fakeCreditCard }}`                |
| `fakeAccountNumber`| Generates a fake bank account number | `{{ fakeAccountNumber }}`             |
| `fakeURL`       | Generates a fake URL (optional protocol)| `{{ fakeURL "https" }}`               |
| `fakeDomain`    | Generates a fake domain name            | `{{ fakeDomain }}`                    |
| `fakeIPv4`      | Generates a fake IPv4 address           | `{{ fakeIPv4 }}`                      |
| `fakeIPv6`      | Generates a fake IPv6 address           | `{{ fakeIPv6 }}`                      |
| `fakeUserAgent` | Generates a fake user agent             | `{{ fakeUserAgent "chrome" }}`        |
| `fakeHTTPMethod`| Generates a fake HTTP method            | `{{ fakeHTTPMethod }}`                |
| `fakeMacAddress`| Generates a fake MAC address            | `{{ fakeMacAddress }}`                |
| `fakePort`      | Generates a fake port (optional range)  | `{{ fakePort 1024 65535 }}`           |
| `fakeAddress`   | Generates a full street address         | `{{ fakeAddress }}`                   |
| `fakeCity`      | Generates a random city name            | `{{ fakeCity }}`                      |
| `fakeCountry`   | Generates a random country name         | `{{ fakeCountry }}`                   |
| `fakeCountryCode`| Generates a random country code        | `{{ fakeCountryCode }}`               |
| `fakeZipCode`   | Generates a random zip/postal code      | `{{ fakeZipCode }}`                   |
| `fakeLatitude`  | Generates a random latitude             | `{{ fakeLatitude 10.0 20.0 }}`        |
| `fakeLongitude` | Generates a random longitude            | `{{ fakeLongitude 10.0 20.0 }}`       |
| `fakeStreet`    | Generates a random street address       | `{{ fakeStreet }}`                    |
| `fakeState`     | Generates a random state name           | `{{ fakeState }}`                     |
| `fakeUUID`      | Generates a UUID v4                     | `{{ fakeUUID }}`                      |
| `fakeULID`      | Generates a ULID                        | `{{ fakeULID }}`                      |
| `fakeWord`      | Generates a random word                 | `{{ fakeWord }}`                      |
| `fakeSentence`  | Generates a random sentence             | `{{ fakeSentence 10 }}`               |
| `fakeParagraph` | Generates a random paragraph            | `{{ fakeParagraph 3 }}`               |
| `fakeTitle`     | Generates a random job/content title    | `{{ fakeTitle }}`                     |
| `fakeText`      | Alias for `fakeParagraph`               | `{{ fakeText 3 }}`                    |
| `fakeEmoji`     | Generates a random emoji                | `{{ fakeEmoji }}`                     |
| `fakePassword`  | Generates a random password             | `{{ fakePassword 16 }}`               |
| `fakeLoremIpsum`| Generates random Lorem Ipsum text       | `{{ fakeLoremIpsum 20 }}`             |
| `fakePastDate`   | Generates a random date in the past     | `{{ fakePastDate 30 "2006-01-02" }}`  |
| `fakeFutureDate` | Generates a random date in the future   | `{{ fakeFutureDate 30 "2006-01-02" }}`|
| `fakeRecentDate` | Generates a random date from last 24h   | `{{ fakeRecentDate }}`                |
| `fakeTimestamp`  | Generates a random timestamp string      | `{{ fakeTimestamp }}`                 |
| `fakeUnixTime`   | Generates a random Unix timestamp (`int64`)| `{{ fakeUnixTime }}`                 |
| `fakeOrderID`    | Generates a realistic order ID          | `{{ fakeOrderID }}`                   |
| `fakeTransactionID`| Generates a realistic transaction ID   | `{{ fakeTransactionID }}`             |
| `fakeSessionID`  | Generates a unique session ID           | `{{ fakeSessionID }}`                 |
| `fakeHexColor`   | Generates a random hex color            | `{{ fakeHexColor }}`                  |
| `fakeImageURL`   | Generates a random placeholder image URL| `{{ fakeImageURL 800 600 }}`          |

#### Visual & Flair Functions

| Function      | Description                             | Example                               |
|---------------|-----------------------------------------|---------------------------------------|
| `randomEmoji` | Returns a random emoji from curated list| `{{ randomEmoji }}`                   |
| `randomColor` | Returns a random color name             | `{{ color randomColor "text" }}`      |
| `sessionAge`  | Returns duration since session started  | `{{ sessionAge \| shortTime }}`       |

#### Collection Functions

| Function   | Description                             | Example                               |
|------------|-----------------------------------------|---------------------------------------|
| `default`  | Returns default value if input is empty | `{{ .val \| default "fallback" }}`    |
| `ternary`  | Selects value based on boolean          | `{{ ternary .ok "yes" "no" }}`        |
| `dict`     | Creates a map from key-value pairs      | `{{ $d := dict "a" 1 "b" 2 }}`        |
| `list`     | Creates a list from arguments           | `{{ $l := list 1 2 3 }}`              |
| `keys`     | Returns sorted keys of a map            | `{{ keys .map \| join "," }}`         |
| `pick`     | Filters map by selected keys            | `{{ pick (list "a") .map }}`          |
| `chunk`    | Splits a list into chunks of size N     | `{{ chunk 2 .list }}`                 |
| `uniq`     | Returns unique items from a list        | `{{ .list \| uniq }}`                 |
| `first`    | Returns the first item of a list        | `{{ .list \| first }}`                |
| `last`     | Returns the last item of a list         | `{{ .list \| last }}`                 |
| `pluck`    | Extracts a field from a list of maps    | `{{ .users \| pluck "id" }}`          |



#### Template Context

The root context (`.`) available in templates provides access to connection, message, server, and session data. This context is automatically populated by xwebs.

| Field      | Description                                            | Example                     |
|------------|--------------------------------------------------------|-----------------------------|
| `.Conn`    | Connection metadata (URL, subprotocol, headers, etc.)  | `{{ .Conn.URL }}`           |
| `.Msg`     | Incoming/Outgoing message details (Data, Type, Length) | `{{ .Msg.Length }} bytes`   |
| `.Handler` | Execution results (Stdout, Stderr, ExitCode, Duration) | `{{ .Handler.Duration }}`   |
| `.Server`  | Global server metrics (ClientCount, Uptime)            | `{{ .Server.ClientCount }}` |
| `.RemoteAddr`| Remote client address (Server mode only)            | `{{ .RemoteAddr }}`         |
| `.Session` | Persistent key-value store for the current session     | `{{ index .Session "id" }}` |
| `.Env`     | Environment variables (if not sandboxed)               | `{{ .Env.PATH }}`           |

#### Context Management Functions

These functions allow you to interact with the session data dynamically within your templates.

| Function       | Description                                  | Example                             |
|----------------|----------------------------------------------|-------------------------------------|
| `sessionSet`   | Sets a value in the persistent session store | `{{ sessionSet "user" "admin" }}`   |
| `sessionGet`   | Retrieves a value from the session store     | `Welcome, {{ sessionGet "user" }}!` |
| `sessionClear` | Clears all data from the session store       | `{{ sessionClear }}`                |

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
      --no-shell-func      disable dangerous template functions (shell, env, fileRead, etc.)
      --color string      color output mode: auto, on, off (default "auto")
      --log-level string   logging level: debug, info, warn, error (default "info")
      --log-format string  log format: text, json (default "text")
      --log string         log all traffic to a JSONL file
      --record string      record session to a JSONL file with timing
  -h, --help               help for xwebs
```

## Configuration

### Config File Locations

xwebs searches for configuration files in the following order (first found wins):

1. Custom path specified via `-c, --config` flag
2. `.xwebs.yaml` in current working directory
3. `~/.xwebs.yaml` (user home directory)

### Environment Variable Mapping

| Flag              | Environment Variable  |
|-------------------|-----------------------|
| `--verbose`       | `XWEBS_VERBOSE`       |
| `--quiet`         | `XWEBS_QUIET`         |
| `--color`         | `XWEBS_COLOR`         |
| `--log-level`     | `XWEBS_LOG_LEVEL`     |
| `--log-format`    | `XWEBS_LOG_FORMAT`    |
| `--profile`       | `XWEBS_PROFILE`       |
| `--no-shell-func` | `XWEBS_NO_SHELL_FUNC` |

### REPL Configuration

The REPL supports persistent command history, reverse-i-search (Ctrl+R), and configurable limits.

Example configuration in `~/.xwebs.yaml`:

```yaml
repl:
  history-file: "~/.xwebs_history"
  history-limit: 1000
  # Custom prompt with colors and templates
  prompt: "{{green .Host}} {{bold \">\"}} "

profiles:
  debug:
    repl:
      prompt: "[{{magenta .ConnectionID}}] {{cyan .Host}} {{bold \">\"}} "
```

**History Features:**
- **Persistence**: Commands are saved across sessions to the configured `history-file`.
- **Search**: Use `Ctrl+R` in the REPL to search through previous commands.
- **Navigation**: Use Up/Down arrows to navigate command history.
- **Manual Inspection**: Use the `:history` command with powerful flags:

| Flag | Short | Description |
|------|-------|-------------|
| `--number <N>` | `-n` | Show last N commands (default: 20) |
| `--search <term>` | `-s` | Search history for commands containing the term (case-insensitive, highlighted) |
| `--filter <pattern>` | `-f` | Filter history using glob or `/regex/` pattern |
| `--unique` | | Show only unique commands (remove duplicates, keep last occurrence) |
| `--reverse` | `-r` | Display history in reverse chronological order |
| `--json` | | Output history in structured JSON format |
| `--export <file>` | `-e` | Export history to a file (`.jsonl` → JSONL, otherwise plain text) |
| `--clear` | `-c` | Clear the entire history after confirmation |

**History Examples:**
```text
# Show last 50 commands
> :history -n 50

# Search for commands containing "deploy" (highlighted)
> :history -s deploy

# Filter with a regex pattern
> :history -f /^:send/

# Filter with a glob pattern
> :history -f ":send*"

# Show only unique commands
> :history --unique

# Combine flags: search + unique + JSON output
> :history -s deploy --unique --json

# Export filtered history to a file
> :history -s deploy -e deploy_commands.txt

# Export as JSONL (one JSON object per line)
> :history -e session_history.jsonl

# Clear all history (with confirmation)
> :history -c
⚠  This will permanently clear 42 entries from ~/.xwebs_history.
   Are you sure? (y/N): y
✓ History cleared.
```

Flags can be combined logically. For example, `:history -s deploy --unique -r --json` will search for "deploy", deduplicate, reverse the order, and output as JSON.

#### History Editing (`:hedit`)

The `:hedit` command lets you recall a previous command, edit it in your `$EDITOR`, and then re-execute the modified version. This is especially useful for tweaking complex multi-line commands, heredocs, or long JSON payloads.

**Usage:**
```text
:hedit           # Edit the most recent non-:hedit command
:hedit -n <N>    # Edit history item number N (as shown by :history)
```

**How it works:**
1. The command loads the target history entry (including multiline blocks like heredocs and `\` continuations).
2. Opens the content in your `$EDITOR` (falls back to `$VISUAL`, then `vim`).
3. After saving and closing the editor, the edited content is loaded into the REPL prompt for review.
4. You can accept (press Enter), modify further, or cancel (Ctrl+C).

**Examples:**
```text
# Edit the most recent command
> :hedit

# Edit history item #42 (find the number with :history)
> :hedit -n 42

# Workflow: find a command, then edit it
> :history -s deploy
Command History (2 matching):
    15  :sendj {"action":"deploy","env":"staging"}
    28  :sendj {"action":"deploy","env":"production"}
> :hedit -n 15
# Editor opens with the command; modify and save to re-execute
```

> [!TIP]
> `:hedit` is multiline-aware — if the target command was a heredoc or `\`-continued block, the entire block is loaded into the editor, not just a single line.

### Tab Completion

The REPL features an intelligent, context-aware tab completion system:

- **Commands & Aliases**: Type `:` and press `Tab` to see all available commands.
- **Template Functions**: Suggestions for functions (e.g., `upper`, `toJSON`) appear when typing inside `{{ ... }}`.
- **Live JSON Keys**: As messages are received, the REPL learns the JSON structure and suggests keys for `:sendj` and `:sendt` commands.
- **Context-Aware Arguments**:
    - `:connect` suggests bookmarks and aliases from your configuration.
    - `:set`, `:get`, and `:vars` suggest session variables.
- **File Paths**: Suggestions for local files and directories where appropriate (e.g., in `:connect`).
- **Handler Names**: Suggestions for registered handlers (supported in EPIC 05).

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
xwebs connect --profile debug URL
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
