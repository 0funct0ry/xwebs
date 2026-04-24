---
title: "xwebs replay"
description: "Replay a recorded xwebs session"
generated: "2026-04-23"
---

# xwebs replay

Replay a JSONL session file recorded with xwebs connect --record. Can replay against a live server (--target) or serve the recorded responses as a mock server (--serve).

---

## Synopsis

```bash
xwebs replay <file> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--target` | string | — | Target WebSocket URL to replay against |
| `--serve` | bool | false | Serve recorded responses as a mock server |
| `--port` | int | 8080 | Port for mock server mode |
| `--speed` | float | 1.0 | Playback speed multiplier |

## Examples

```bash
xwebs replay session.jsonl --target wss://staging.api.example.com

xwebs replay session.jsonl --serve --port 8080
```

