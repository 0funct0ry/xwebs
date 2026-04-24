---
title: "xwebs mock"
description: "Start a scripted mock WebSocket server"
generated: "2026-04-24"
---

# xwebs mock

Start a mock server that responds to clients according to a scenario file. Scenarios define an ordered sequence of expect/respond steps, simulating a real server's behavior without running actual backend code.

---

## Synopsis

```bash
xwebs mock --port <port> --scenario <file> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | 8080 | Port to listen on |
| `--scenario` | string | — | Scenario YAML file (required) |

## Examples

```bash
xwebs mock --port 8080 --scenario test/auth-flow.yaml
```

