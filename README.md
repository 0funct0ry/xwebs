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

### On the Roadmap (Planned)
- **Server Mode** — WebSocket server with handler dispatch and administration REPL
- **Relay & Broadcast** — MITM proxy and pub/sub fan-out modes
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

The `connect` command performs a WebSocket handshake, injects custom headers, and negotiates subprotocols. Once connected, it enters a basic interactive mode where you can type messages to send and see incoming messages from the server.

```bash
# Direct URL
xwebs connect wss://echo.websocket.org
```

### Interactive REPL Mode

When running in a terminal (TTY), `xwebs connect` enters a rich interactive REPL mode. Unlike basic "line-at-a-time" interfaces, this REPL supports powerful built-in commands starting with a colon `:`.

**Common REPL Commands:**

| Command          | Description                                  |
|------------------|----------------------------------------------|
| `:help`          | List all available commands                  |
| `:status`        | Show detailed connection metadata            |
| `:send <text>`   | Send a text message (default for bare text)  |
| `:sendb <hex>`   | Send binary data (hex or `base64:`)          |
| `:sendj <json>`  | Send validated JSON message                  |
| `:sendt <tmpl>`  | Send rendered Go template                    |
| `:ping [p]`     | Send a ping frame (text or binary prefix)    |
| `:pong [p]`     | Send a pong frame (text or binary prefix)    |
| `:connect <url>` | Connect to a new URL in the same session     |
| `:reconnect`     | Force a reconnection to the current URL      |
| `:close [c][r]`  | Send a graceful close frame                  |
| `:disconnect`    | Disconnect from the current server           |
| `:set <k> <v>`   | Set a session variable for templates         |
| `:get <k>`       | Get the value of a session variable          |
| `:vars`          | List all active session variables            |
| :env           | List all environment variables               |
| :history [n]    | Display last N command history               |
| :bench <n> <m>  | Benchmark latency for N iterations           |
| :flood <msg>    | Stress test server with high-rate messages   |
| :watch          | Monitor connection statistics in real-time   |
| :exit, :quit      | Disconnect and quit the application (or `Ctrl+D`) |
| :clear           | Clear the terminal screen                    |
| `:format <type>` | Set display format: `json`, `raw`, `hex`, `template` |
| `:filter <expr>` | Set a display filter (`.jq`, `/regex/`, or `off`) |
| `:quiet`         | Toggle non-message output suppression        |
| `:verbose`       | Toggle frame-level metadata display          |
| `:timestamps`    | Toggle ISO 8601 message timestamps           |
| :color <mode>  | Set coloring mode: `on`, `off`, `auto`       |
| `:source <f>`   | Execute a `.xwebs` script file               |
| `:alias <n> <c>`| Create a command alias with positional args    |
| `:wait <dur>`   | Pause execution (e.g., `1s`, `500ms`)        |
| `:assert <ex>`  | Validate state with template expressions       |
| `:log <file>`   | Log traffic to JSONL (one object per line)   |
| `:record <f>`   | Start relative-time session recording        |
| `:replay <f>`   | Play back a session recording with timing    |
| `:mock <f>`     | Load YAML-based mock scenario                |

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

### Message Handlers (YAML)

`xwebs` allows you to define declarative message handlers in a YAML configuration file. This is useful for building reactive WebSocket clients and servers without writing custom Go code.

**Key Features:**
- **Priority-Based Execution**: Handlers can be assigned a `priority` (higher numbers execute first). Handlers with the same priority run in the order they appear in the file.
- **Match Conditions**: Match incoming messages by type (`text`, `json`, `regex`, `glob`) and pattern. The `glob` matcher converts `*` and `?` to logical regexes, intuitively supporting full and substring matching across newlines and slashes.
- **Actions**: Trigger actions like `shell` commands, `send` messages, `builtin` commands, or `log` to files.
- **Lifecycle Events**: Bind actions to `on_connect`, `on_disconnect`, and `on_error` events.
- **Template Support**: All message and command fields support full Go templates with access to `.Msg`, `.Conn`, `.Vars`, etc.
- **REPL Observability**: Use the `:handlers` command in the REPL to see the loaded handlers in their execution order.

**Usage:**
```bash
# Load handlers from a YAML file
xwebs connect wss://echo.websocket.org --handlers my_handlers.yaml
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
| `nowUnix`       | Returns current Unix timestamp          | `{{ nowUnix }}`                                   |
| `formatTime`    | Formats a time object or timestamp      | `{{ formatTime "2006-01-02" .t }}`                |
| `parseTime`     | Parses a time string                    | `{{ parseTime "2006-01-02" "2023-01-01" }}`       |
| `duration`      | Parses a duration string                | `{{ duration "1h30m" }}`                          |
| `since`         | Calculated duration since time          | `{{ since .start }}`                              |
| `uptime`        | Returns xwebs process uptime            | `{{ uptime }}`                                    |

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

| Function     | Description                     | Example                                   |
|--------------|---------------------------------|-------------------------------------------|
| `env`        | Returns an environment variable | `{{ env "HOME" }}`                        |
| `shell`      | Executes a shell command        | `{{ shell "ls -l" }}`                     |
| `hostname`   | Returns the system hostname     | `{{ hostname }}`                          |
| `pid`        | Returns the process ID          | `{{ pid }}`                               |
| `fileRead`   | Reads a file's content          | `{{ fileRead "config.json" }}`            |
| `fileExists` | Checks if a file exists         | `{{ if fileExists "a.txt" }}...{{ end }}` |

> [!NOTE]
> These functions can be disabled for security using the `--no-shell-func` global flag. See [Template Sandboxing](#template-sandboxing) for details.

#### Template Sandboxing

For security-sensitive environments or when running untrusted configurations, you can enable template sandboxing using the `--no-shell-func` flag (or `XWEBS_NO_SHELL_FUNC=true` environment variable). 

When enabled, the following functions are restricted and will return an error if called:
- `env`, `shell`, `fileRead`, `fileExists`, `glob`, `hostname`, `pid`, `cwd`, `tempFile`

Safe functions (string manipulation, JSON processing, math, time, encoding, cryptography, etc.) continue to work normally in sandbox mode.

#### ID Functions

| Function  | Description                             | Example                               |
|-----------|-----------------------------------------|---------------------------------------|
| `uuid`    | Generates a UUID v4                     | `{{ uuid }}`                          |
| `ulid`    | Generates a ULID                        | `{{ ulid }}`                          |
| `nanoid`  | Generates a NanoID                      | `{{ nanoid }}`                        |
| `shortid` | Generates a ShortID                     | `{{ shortid }}`                       |
| `counter` | Returns/increments a named counter      | `{{ counter "msg_id" }}`              |

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
| `.Server`  | Global server metrics (ClientCount, Uptime)            | `{{ .Server.Uptime }}`      |
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
```

**History Features:**
- **Persistence**: Commands are saved across sessions to the configured `history-file`.
- **Search**: Use `Ctrl+R` in the REPL to search through previous commands.
- **Navigation**: Use Up/Down arrows to navigate command history.
- **Manual Inspection**: Use the `:history [n]` command to view the last `n` entries.

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
