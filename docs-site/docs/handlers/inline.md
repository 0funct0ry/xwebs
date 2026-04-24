---
title: Inline Handlers
description: Define WebSocket message handlers directly on the command line with --on, without a YAML config file.
---

# Inline Handlers

Inline handlers let you define message→action bindings directly on the command line using `--on`, without a YAML config file. They cover the most common handler patterns in a single flag; advanced features like pipelines, retry, and rate limiting require [handlers.yaml](./yaml-schema.md).

## Syntax

```
--on '<match> :: <action> [:: <action>] [:: <option>]'
```

A single quoted string containing **segments** separated by ` :: ` (space-colon-colon-space). The first segment is the **match expression**; subsequent segments are **actions** or **options** identified by prefix.

| Position | Role | Required | Prefix |
|----------|------|----------|--------|
| 1st | Match expression | Yes | Auto-detected or explicit |
| 2nd+ | Action or option | At least one | `run:`, `respond:`, `timeout:`, or `exclusive` |

The `::` separator was chosen because it does not appear in shell syntax, Go template delimiters, JSON, or jq expressions.

---

## Match Auto-Detection

The match type is inferred from the expression's leading characters:

| First character(s) | Detected type | Rationale |
|---------------------|---------------|-----------|
| `.` | `jq` | jq expressions start with `.` |
| `*` or `?` | `glob` | Glob wildcards |
| `^` or `(` | `regex` | Regex anchors/groups |
| Anything else | `glob` | Literal strings are valid globs |

Override with explicit prefixes: `jq:`, `glob:`, `regex:`, `template:`.

### Examples by Match Type

**Exact match** — a literal string only matches that exact message:

```bash
--on 'ping :: respond:pong'
--on '{"type":"ping"} :: respond:{"type":"pong"}'
```

**Glob** — `*` and `?` wildcards:

```bash
--on '*error* :: run:notify-send "Error" "{{.Message}}"'
--on 'cmd:* :: run:handle-cmd.sh'
--on '* :: run:logger.sh'
```

**jq** — auto-detected on leading `.`:

```bash
--on '.type == "ping" :: respond:{"type":"pong"}'
--on '.event.level == "critical" :: run:alert.sh'
--on '.error != null :: run:handle-error.sh'
```

**Regex** — auto-detected on `^` or `(`:

```bash
--on '^CMD: :: run:dispatch.sh'
--on '^(ping|heartbeat)$ :: respond:{"type":"pong"}'
```

**Template** — requires explicit prefix:

```bash
--on 'template:{{eq (.Message | jq ".level") "critical"}} :: run:alert.sh'
```

---

## Actions

### `run:<command>`

Execute a shell command via `sh -c`. The incoming message is piped to stdin. Template expressions are expanded before execution.

```bash
--on '.type == "data" :: run:echo "{{.Message | jq ".payload"}}" >> data.jsonl'
```

`{{.Stdout}}`, `{{.Stderr}}`, and `{{.ExitCode}}` are available in a subsequent `respond:`.

### `respond:<template>`

Send a response message. The template is expanded with full handler context.

```bash
--on '.type == "ping" :: respond:{"type":"pong","ts":"{{now | formatTime "RFC3339"}}"}'
```

When both `run:` and `respond:` are present, `run:` executes first and its output is available in `respond:`.

### `timeout:<duration>`

Kill the `run:` command if it exceeds this duration. Uses Go duration syntax: `5s`, `1m30s`, `500ms`.

```bash
--on '.type == "query" :: run:slow-query.sh :: respond:{"result":{{.Stdout | toJSON}}} :: timeout:10s'
```

### `exclusive`

Stop checking further handlers if this one matches:

```bash
--on '.type == "auth" :: run:verify-token.sh :: respond:{"auth":"ok"} :: exclusive'
```

---

## The `--respond` Flag

`--respond <template>` sets a **default response template** for inline handlers that match but have no `respond:` segment.

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: run:echo pong' \
  --on '.type == "status" :: run:get-status.sh' \
  --respond '{"output":"{{.Stdout | trim}}","ok":{{eq .ExitCode 0}}}'
```

Both handlers use `--respond` as their response. Each gets its own `.Stdout` and `.ExitCode`.

**Key behaviors:**

1. `--respond` applies only to handlers **that matched** but have no `respond:` segment.
2. It does **not** fire for unmatched messages.
3. Handlers with their own `respond:` ignore `--respond`.
4. `--respond` alone (with no `--on`) has no effect.

```bash
# ping uses its own respond:, query uses --respond
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong"}' \
  --on '.type == "query" :: run:query.sh' \
  --respond '{"result":{{.Stdout | toJSON}}}'
```

---

## Execution Order

1. Each `--on` handler is evaluated in command-line order.
2. For each matching handler: `run:` executes (if present), then the response is sent.
3. If a handler has `exclusive`, stop checking further handlers.
4. If no handler matched, no response is sent.

Multiple `--on` handlers can match the same message unless `exclusive` is used.

---

## Client vs Server Mode

Inline handlers work identically in both modes. The difference is context variables.

**Client mode** (`xwebs connect`) — handlers trigger on messages from the server:

```bash
xwebs connect wss://api.example.com \
  --on '.type == "error" :: run:notify-send "WS Error" "{{.Message | jq ".detail"}}"' \
  --on '.type == "data" :: run:echo {{.Message | jq ".payload"}} >> data.jsonl'
```

**Server mode** (`xwebs serve`) — handlers trigger on messages from connected clients:

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong","ts":"{{now}}"}' \
  --on '.type == "query" :: run:handle-query.sh :: timeout:10s' \
  --respond '{"result":{{.Stdout | toJSON}}}'
```

Server mode adds `.RemoteAddr`, `.ClientCount`, and per-client `.ConnectionID`.

---

## Mapping to YAML Config

Every inline handler maps directly to a YAML handler:

| Inline segment | YAML field |
|----------------|------------|
| Match expression (1st) | `match:` (auto-detected type) |
| `run:<cmd>` | `run:` |
| `respond:<template>` | `respond:` |
| `timeout:<dur>` | `timeout:` |
| `exclusive` | `exclusive: true` |
| `--respond` flag | Default `respond:` for handlers without one |

**Equivalence example:**

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong"} :: exclusive' \
  --on '.type == "data" :: run:process.sh :: timeout:5s' \
  --respond '{"error":"unknown"}'
```

Is equivalent to:

```yaml
handlers:
  - name: inline-1
    match:
      jq: '.type == "ping"'
    respond: '{"type":"pong"}'
    exclusive: true

  - name: inline-2
    match:
      jq: '.type == "data"'
    run: process.sh
    respond: '{"error":"unknown"}'
    timeout: 5s
```

---

## What Requires a Config File

These features have no inline equivalent:

| Feature | Why |
|---------|-----|
| `pipeline:` | Too complex for a single string |
| `retry:` | Multiple sub-fields |
| `rate_limit:` | Per-handler rate limiting |
| `debounce:` | Timing configuration |
| `concurrent: false` | Serialization control |
| Composite matchers (`all:`/`any:`) | Nesting doesn't fit single-string syntax |
| `on_connect:` / `on_disconnect:` | Lifecycle hooks |
| `variables:` | Use `--var key=value` flag instead |

---

## Shell Quoting Guide

Use single quotes for the outer shell wrapper and double quotes inside (for JSON and jq):

```bash
# Recommended
--on '.type == "ping" :: respond:{"type":"pong"}'
```

**JQ string syntax warning:** jq requires double quotes for string literals. `.type == 'ping'` causes a `gojq` parse error: `unexpected token "'"`. If you must use double quotes for the outer wrapper, escape the inner ones:

```bash
# Alternative: double quotes outside, escaped inside
--on ".type == \"ping\" :: respond:{\"type\":\"pong\"}"
```

---

## Quick Reference

```
--on '<match> :: run:<cmd> :: respond:<tmpl> :: timeout:<dur> :: exclusive'
      ───┬───    ────┬────    ───────┬──────    ──────┬──────    ────┬────
     jq/glob/     shell cmd    response msg     kill slow cmd   stop here
     regex                    (optional)        (optional)
     (auto-detect)

--respond '<template>'
           default response for --on handlers with no respond: segment
```
