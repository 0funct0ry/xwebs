---
title: "xwebs connect"
description: "Connect to a WebSocket server"
generated: "2026-04-24"
---

# xwebs connect

Connect to a remote WebSocket server in interactive (REPL) or non-interactive (pipeline) mode. In interactive mode a readline-based REPL is started with tab completion, persistent history, and syntax highlighting. In non-interactive mode (piped stdin or --once/--script flags) xwebs reads from stdin, sends messages, and writes responses to stdout.

---

## Synopsis

```bash
xwebs connect <url> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--header` | stringArray | — | HTTP header to include in the handshake (repeatable) |
| `--subprotocol` | stringArray | — | WebSocket subprotocol(s) to negotiate |
| `--cert` | string | — | TLS client certificate file (PEM) |
| `--key` | string | — | TLS client key file (PEM) |
| `--ca` | string | — | CA certificate file for server verification |
| `--insecure` | bool | false | Skip TLS certificate verification |
| `--compress` | bool | false | Enable permessage-deflate compression |
| `--max-message-size` | string | 16MB | Maximum incoming message size |
| `--ping-interval` | duration | 30s | Keepalive ping interval |
| `--reconnect` | bool | true | Automatically reconnect on disconnect |
| `--reconnect-max` | duration | 5m | Maximum reconnect backoff duration |
| `--interactive` | bool | auto | Force interactive REPL (even without TTY) |
| `--keymap` | string | emacs | REPL keymap: emacs or vi |
| `--history-size` | int | 10000 | Maximum REPL history entries |
| `--proxy` | string | — | Proxy URL (http://, https://, socks5://) |
| `--verbose` | bool | false | Verbose output |
| `--quiet` | bool | false | Suppress non-essential output |
| `--color` | string | auto | Color mode: auto, on, off |
| `--log-level` | string | info | Log level: debug, info, warn, error |
| `--log-format` | string | text | Log format: text or json |
| `--config` | string | ~/.xwebs.yaml | Config file path |
| `--profile` | string | — | Named profile from config file |
| `--handlers` | string | — | Handler config file (YAML) |
| `--on` | stringArray | — | Inline handler: '<match> :: <action>' |
| `--respond` | string | — | Default response template for --on handlers |
| `--no-shell-func` | bool | false | Disable shell/env/fileRead template functions |
| `--sandbox` | bool | false | Enable shell command allowlisting |
| `--allowlist` | strings | — | Allowed shell commands (requires --sandbox) |
| `--once` | bool | false | Exit after receiving the first response |
| `--input` | string | — | Send messages from file (one per line) |
| `--send` | string | — | Send this message on connect |
| `--expect` | int | — | Exit after receiving N responses |
| `--until` | string | — | Exit when this template evaluates truthy |
| `--output` | string | — | Write responses to file |
| `--jsonl` | bool | false | Output one JSON message per line |
| `--script` | string | — | Execute a .xwebs script file |
| `--watch` | string | — | Re-send file content on change |
| `--timeout` | duration | — | Exit after this duration |
| `--exit-on` | string | — | Template producing integer exit code |

## Examples

```bash
# Interactive REPL
xwebs connect wss://echo.websocket.org

# Non-interactive: send once and exit
echo '{"type":"ping"}' | xwebs connect wss://api.example.com --once

# With auth header (template expanded)
xwebs connect wss://api.example.com \
  --header "Authorization: Bearer {{env \"TOKEN\"}}"

# Pipe JSON responses to jq
xwebs connect wss://stream.example.com --jsonl | jq '.price'

# With inline handler
xwebs connect wss://api.example.com \
  --on '.type == "error" :: run:notify-send "Error" "{{.Message}}"'
```

