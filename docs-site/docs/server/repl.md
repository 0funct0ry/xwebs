---
title: Server REPL
description: All commands available in the xwebs serve interactive REPL for live server administration.
---

# Server REPL

When `xwebs serve` runs on a TTY (or with `--interactive`), it drops into a **server-mode REPL** alongside the running server. The server continues accepting connections and processing messages in the background while you administer it from the foreground.

The server REPL shares the same readline infrastructure, history, tab completion, and output formatting as the [client REPL](../client/repl.md), but adds server-specific commands.

```bash
xwebs serve --port 8080 --handlers handlers.yaml --interactive
```

## Server Management

```
:status                           Show uptime, connected client count, message stats
:clients                          List all connected clients (ID, remote addr, connected time, msg counts)
:client <id>                      Inspect a single client in detail
:kick <id> [code] [reason]        Disconnect a client with optional close code and reason
:send [flags] <id> <message>      Send a message to a specific client
:broadcast [flags] <message>      Broadcast a message to all connected clients
```

**Flags for `:send` and `:broadcast`:**

- `-j, --json` — Validate and send as JSON
- `-t, --template` — Render message as a Go template before sending
- `-b, --binary` — Send as binary (supports `base64:` prefix or hex strings)

```
xwebs> :clients
  ID         REMOTE ADDR        CONNECTED     MSGS IN  MSGS OUT
  c-a1b2c3   192.168.1.42:5891  2m ago        47       23
  c-d4e5f6   10.0.0.5:38201     30s ago       3        1

xwebs> :send -t c-a1b2c3 {"type":"ping","server_time":"{{now | formatTime \"RFC3339\"}}"}
→ sent to c-a1b2c3 (47 bytes)

xwebs> :broadcast -j {"type":"maintenance","eta":"2m"}
→ broadcast to 2 clients (40 bytes each)

xwebs> :kick c-d4e5f6 1001 "going away"
✓ kicked c-d4e5f6
```

---

## Handler Management

```
:handlers                         List loaded handlers with match counts, avg latency, error rates
:handler <name>                   Show detailed metrics for a specific handler
:handler add <flags>              Dynamically add a new message handler
:handler delete <name>            Remove a handler by name
:handler rename <old> <new>       Rename an existing handler
:handler edit [name]              Edit a handler or full config in $EDITOR
:handler save [file]              Persist in-memory handlers to a YAML file
:reload                           Hot-reload handler config from disk
:enable <name>                    Enable a disabled handler
:disable <name>                   Temporarily disable a handler
```

Hot-reload is the most useful command during development — edit your `handlers.yaml` and reload without restarting the server:

```
xwebs> :reload
✓ Reloaded handlers.yaml (5 handlers, 1 new, 0 removed, 1 updated)
```

---

## Pub/Sub & Topics

```
:topics                           List active topics with subscriber counts
:topic <name>                     Show per-subscriber details
:publish <topic> <message>        Publish a raw message to all subscribers
:publish -t <topic> <template>    Publish a template-expanded message
:publish --allow-empty <topic> <message>  Publish even with zero subscribers
:subscribe <client-id> <topic>    Manually subscribe a client to a topic
:unsubscribe <client-id> <topic>  Remove a client from a topic
:unsubscribe <client-id> --all    Remove a client from all topics
```

```
xwebs> :topics
  TOPIC       SUBSCRIBERS  LAST ACTIVITY
  trades      4            2s ago
  quotes      2            100ms ago

xwebs> :publish -t trades {"price":"{{.Vars.last_price}}","ts":"{{now}}"}
→ published to 4 subscribers

xwebs> :subscribe c-a1b2c3 alerts
✓ c-a1b2c3 subscribed to alerts
```

---

## Key-Value Store

The KV store is server-scoped — shared across all connections and handlers. Read it in templates with `{{kv "key"}}`. Write and delete from handlers using `kv-set`/`kv-del` builtins.

```
:kv list                  List all keys
:kv get <key>             Get a value
:kv set <key> <value>     Set a key (stored as string)
:kv set -t <key> <value>  Set a key, expanding value as a Go template first
:kv set -j <key> <value>  Set a key, validating value as JSON first
:kv del <key>             Delete a key
```

```
xwebs> :kv set maintenance_mode true
xwebs> :kv get maintenance_mode
true
xwebs> :kv set -t last_updated "{{now | formatTime \"RFC3339\"}}"
xwebs> :kv list
  maintenance_mode  true
  last_updated      2024-01-15T10:30:00Z
```

KV differs from session `:set`/`:get` which are client-mode only. Use `:kv` for operator scratch state in server mode.

---

## Observability

```
:stats                    Show live counters: connections, messages in/out, handler executions, errors
:slow [n]                 Show the n slowest handler executions (default 10)
:uptime                   Server uptime and start time
```

```
xwebs> :stats
  Uptime:        14m32s
  Connections:   2 active, 7 total
  Messages:      142 received, 89 sent
  Handlers:      deploy-trigger (12 hits, avg 340ms), healthcheck (98 hits, avg 2ms)
  Errors:        1 (deploy-trigger: timeout)
```

---

## Server Admin

```
:drain                    Stop accepting new connections, wait for existing ones to close
:pause                    Pause message processing (buffer incoming messages)
:resume                   Resume processing (flush buffered messages)
:shutdown                 Graceful shutdown: drain + stop server
```

---

## Shared Commands

These are available in both client and server mode:

```
:source <file>            Execute commands from a script file
:wait <duration>          Pause (e.g., 1s, 500ms)
:assert <expr>            Verify a condition; fails if empty, 0, or false
:write <file>             Save content to a file
:history [-n <count>]     View command history
:hedit [-n <n>]           Edit and re-run a previous command in $EDITOR
:! <command>              Execute a shell command
:shell                    Switch to a full interactive shell session
:pwd / :cd / :ls          Filesystem navigation
:env                      List process environment variables
:format <mode>            Set display mode (json, raw, hex, template)
:filter <expr>            Set display filter (jq or regex)
:clear                    Clear screen
:help                     List all commands
:exit / :quit             Shut down the server and exit
```

**Note:** `:set`, `:get`, and `:vars` are **not** available in the server REPL. Use `:kv set`/`:kv get` for server-side scratch state.

---

## Example Session

```bash
$ xwebs serve --port 8080 --handlers handlers.yaml --interactive
xwebs server listening on :8080 (2 paths: /ws, /events)
4 handlers loaded from handlers.yaml

xwebs> :clients
  ID         REMOTE ADDR        CONNECTED     MSGS IN  MSGS OUT
  c-a1b2c3   192.168.1.42:5891  2m ago        47       23
  c-d4e5f6   10.0.0.5:38201     30s ago       3        1

xwebs> :handler disable catch-all
✓ disabled catch-all

xwebs> :reload
✓ Reloaded handlers.yaml (5 handlers, 1 new, 0 removed, 1 updated)

xwebs> :slow 5
  deploy-trigger    340ms avg  (12 calls, 1 timeout)
  query-db          180ms avg  (8 calls, 0 errors)

xwebs> :drain
⏳ Draining... waiting for 2 connections to close
✓ All connections closed.
```
