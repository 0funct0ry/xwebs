---
title: Introduction
description: What xwebs is, why it exists, and a 30-second quickstart.
slug: /
---

# xwebs — WebSocket Swiss Army Knife

Every WebSocket tool out there does one thing: connect and send messages. **xwebs** flips that model entirely. It treats WebSocket messages as **events that trigger shell pipelines**, with full Go template interpolation, an interactive REPL, a server mode, and an optional web UI.

Think of it as `curl` + `netcat` + `jq` + `bash` — but for WebSockets.

## Philosophy

The standard WebSocket workflow is manual: open a connection, type a message, read the response, repeat. xwebs automates every step of that loop. You bind patterns to shell commands so that incoming messages become inputs to your existing toolchain — no SDK required.

A few guiding principles:

- **Shell-first.** Any program you can run from a terminal becomes a WebSocket handler. `jq`, `psql`, `curl`, `ffmpeg`, `python` — wire them all up with a YAML file.
- **Templates everywhere.** Go template expressions work in handler configs, CLI flags, prompts, and REPL commands. Inject message content, environment variables, random IDs, timestamps, crypto hashes — anything.
- **Zero boilerplate for simple cases.** `xwebs connect wss://echo.websocket.org` gives you a working REPL in one command. `--on 'ping :: respond:pong'` adds a handler without any config file.
- **Scales to complex.** When you need pipelines, rate limiting, debouncing, pub/sub, KV storage, or embedded Lua — it's all there in `handlers.yaml`.

## 30-Second Quickstart

**Install:**

```bash
go install github.com/0funct0ry/xwebs@latest
```

**Connect to a public echo server:**

```bash
xwebs connect wss://echo.websocket.org
# xwebs> hello
# ← hello
```

**Serve a simple ping/pong server:**

```bash
xwebs serve --port 8080 --on 'ping :: respond:pong'
```

**Send one message and exit (great for CI/scripts):**

```bash
echo '{"action":"status"}' | xwebs connect wss://api.example.com --once
```

## Modes

| Mode | Command | Use case |
|------|---------|----------|
| Client | `xwebs connect` | Connect to a server, interactive or scripted |
| Server | `xwebs serve` | Run a WebSocket server with handler config |
| Relay | `xwebs relay` | MITM proxy for debugging and transformation |
| Broadcast | `xwebs broadcast` | Fan-out pub/sub server |
| Mock | `xwebs mock` | Scripted mock server for testing |
| Bench | `xwebs bench` | Load testing |
| Replay | `xwebs replay` | Replay recorded sessions |
| Diff | `xwebs diff` | Compare two server endpoints |

Each mode has its own dedicated page in this documentation. Start with [Getting Started](./getting-started.md) for a guided walkthrough.
