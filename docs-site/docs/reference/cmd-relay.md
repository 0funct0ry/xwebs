---
title: "xwebs relay"
description: "Proxy/MITM WebSocket traffic between a client and upstream server"
generated: "2026-04-23"
---

# xwebs relay

Sit between a client and an upstream WebSocket server. Inspect, log, and optionally transform messages in both directions. Useful for debugging production protocols or recording sessions for replay.

---

## Synopsis

```bash
xwebs relay --listen <addr> --upstream <url> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--listen` | string | :9090 | Local address to listen on |
| `--upstream` | string | — | Upstream WebSocket URL (required) |
| `--client-to-server` | string | — | Transform template for client→server messages |
| `--server-to-client` | string | — | Transform template for server→client messages |
| `--log` | string | — | Log all traffic to JSONL file |

## Examples

```bash
xwebs relay --listen :9090 --upstream wss://api.example.com

xwebs relay --listen :9090 --upstream wss://api.example.com \
  --server-to-client '{{.Message | prettyJSON}}' \
  --log relay.jsonl
```

