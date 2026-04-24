---
title: "xwebs broadcast"
description: "Fan-out pub/sub WebSocket server"
generated: "2026-04-23"
---

# xwebs broadcast

Start a simple broadcast server where every message received from any client is forwarded to all connected clients. With --topics enabled, clients can subscribe to named channels.

---

## Synopsis

```bash
xwebs broadcast [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | 8080 | Port to listen on |
| `--topics` | bool | false | Enable topic-based subscription |

## Examples

```bash
xwebs broadcast --port 8080

xwebs broadcast --port 8080 --topics
```

