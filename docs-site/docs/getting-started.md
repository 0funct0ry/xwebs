---
title: Getting Started
description: Install xwebs, connect to a server, and write your first handler in five minutes.
---

# Getting Started

## Installation

### Go Install (Recommended)

```bash
go install github.com/0funct0ry/xwebs@latest
```

Requires Go 1.21+. The binary is placed in `$GOPATH/bin` (typically `~/go/bin`).

### Homebrew

```bash
brew install xwebs
```

### Binary Releases

Download a pre-built binary for your platform from the [GitHub releases page](https://github.com/0funct0ry/xwebs/releases). All releases include Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64).

```bash
# Quick install script (Linux/macOS)
curl -sSL https://raw.githubusercontent.com/0funct0ry/xwebs/main/install.sh | bash
```

### Docker

```bash
docker run --rm -it ghcr.io/0funct0ry/xwebs connect wss://echo.websocket.org
```

### From Source

```bash
git clone https://github.com/0funct0ry/xwebs
cd xwebs
make install
```

Verify the install:

```bash
xwebs version
```

---

## Connect to a Public Echo Server

The simplest possible session: connect to a public echo server and send a message.

```bash
xwebs connect wss://echo.websocket.org
```

You land in an interactive REPL. Type any text and press Enter to send it. The server echoes it back.

```
xwebs> hello world
← hello world
xwebs> {"type":"ping"}
← {"type":"ping"}
xwebs> :status
  URL:     wss://echo.websocket.org
  State:   connected
  RTT:     24ms
  Sent:    2 messages
  Received: 2 messages
xwebs> :exit
```

**Useful REPL commands for a first session:**

| Command | What it does |
|---------|-------------|
| `:status` | Show connection info and message counts |
| `:format json` | Pretty-print JSON responses |
| `:timestamps on` | Show receive timestamps |
| `:help` | List all commands |
| `:exit` | Disconnect and quit |

---

## Serve a Simple Ping/Pong Handler

Start a WebSocket server that responds to `ping` with `pong`:

```bash
xwebs serve --port 8080 --on 'ping :: respond:pong'
```

Test it with a second terminal:

```bash
xwebs connect ws://localhost:8080
# xwebs> ping
# ← pong
```

Or from the command line without a REPL:

```bash
echo "ping" | xwebs connect ws://localhost:8080 --once
# pong
```

---

## First Inline Handler with `--on`

The `--on` flag is xwebs's most powerful one-liner feature. Each `--on` value is a single string with segments separated by ` :: ` (space-colon-colon-space):

```
--on '<match> :: <action> [:: <action>]'
```

**Respond to a JSON message type:**

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong","ts":"{{now}}"}'
```

**Run a shell command and respond with its output:**

```bash
xwebs serve --port 8080 \
  --on '.type == "status" :: run:hostname :: respond:{"host":"{{.Stdout | trim}}"}'
```

**Multiple handlers with `--on` repeated:**

```bash
xwebs serve --port 8080 \
  --on 'ping :: respond:pong' \
  --on '.type == "echo" :: respond:{{.Message}}' \
  --on '* :: respond:{"error":"unknown_message"}'
```

The match expression is auto-detected: strings starting with `.` are jq expressions; `*` or `?` are glob wildcards; `^` or `(` are regex. Everything else is treated as a literal glob.

---

## Next Steps

- **[Client REPL](./client/repl.md)** — all REPL commands, keyboard shortcuts, prompt customization
- **[Handler YAML Schema](./handlers/yaml-schema.md)** — full config reference for complex handlers
- **[Builtins](./builtins/)** — echo, broadcast, kv-set, rate-limit, lua, and 70+ more
- **[Examples](./examples/)** — complete working examples you can run immediately
