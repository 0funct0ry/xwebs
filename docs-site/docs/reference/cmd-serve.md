---
title: "xwebs serve"
description: "Start a WebSocket server"
generated: "2026-04-23"
---

# xwebs serve

Start a WebSocket server that responds to incoming messages using configurable handler pipelines. Handlers are defined in a YAML file (--handlers) or inline with --on flags. In interactive mode (--interactive), a server REPL is started for live administration.

---

## Synopsis

```bash
xwebs serve [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | 8080 | Port to listen on |
| `--host` | string | 0.0.0.0 | Host/interface to bind |
| `--path` | stringArray | /ws | WebSocket endpoint path(s) |
| `--tls` | bool | false | Enable TLS |
| `--cert` | string | — | TLS certificate file (PEM) |
| `--key` | string | — | TLS key file (PEM) |
| `--ui` | bool | false | Enable web UI at / |
| `--metrics` | bool | false | Expose Prometheus metrics at /api/metrics |
| `--interactive` | bool | auto | Force server REPL |
| `--handler-timeout` | duration | 30s | Default handler execution timeout |
| `--max-message-size` | string | 16MB | Maximum incoming message size |
| `--allowed-origins` | strings | — | CORS allowed origins |
| `--rate-limit` | string | — | Global rate limit (N/s or N/m) |
| `--allow-ip` | strings | — | IP allowlist (CIDR) |
| `--deny-ip` | strings | — | IP denylist (CIDR) |
| `--handlers` | string | — | Handler config file (YAML) |
| `--on` | stringArray | — | Inline handler: '' |
| `--respond` | string | — | Default response template for --on handlers |
| `--no-shell-func` | bool | false | Disable shell template functions |
| `--sandbox` | bool | false | Enable shell command allowlisting |
| `--allowlist` | strings | — | Allowed shell commands |
| `--verbose` | bool | false | Verbose output |
| `--quiet` | bool | false | Suppress non-essential output |
| `--log-level` | string | info | Log level |
| `--log-format` | string | text | Log format: text or json |
| `--config` | string | ~/.xwebs.yaml | Config file |
| `--profile` | string | — | Named profile |

## Examples

```bash
# Basic server
xwebs serve --port 8080

# With handler config
xwebs serve --port 8080 --handlers handlers.yaml

# Inline ping/pong
xwebs serve --port 8080 --on 'ping :: respond:pong'

# With TLS
xwebs serve --port 8443 --tls --cert server.crt --key server.key

# Interactive REPL for live administration
xwebs serve --port 8080 --handlers handlers.yaml --interactive
```

