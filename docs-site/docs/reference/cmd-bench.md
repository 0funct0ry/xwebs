---
title: "xwebs bench"
description: "Load test a WebSocket endpoint"
generated: "2026-04-24"
---

# xwebs bench

Open N concurrent connections and send messages at a configured rate for a specified duration. Reports latency percentiles, throughput, and error rates.

---

## Synopsis

```bash
xwebs bench <url> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--connections` | int | 10 | Number of concurrent connections |
| `--rate` | int | 1 | Messages per second per connection |
| `--duration` | duration | 10s | Test duration |
| `--message` | string | — | Message template (evaluated per send) |
| `--output` | string | — | Write results to file |

## Examples

```bash
xwebs bench wss://api.example.com \
  --connections 100 \
  --rate 10 \
  --duration 60s \
  --message '{"type":"ping","id":"{{counter "req"}}"}'
```

