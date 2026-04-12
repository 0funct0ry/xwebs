# Inline Handlers Specification

Inline handlers let you define message→action bindings directly on the command line, without a YAML config file. They cover the most common handler patterns in a single `--on` flag; advanced features (pipelines, retry, rate limiting, composite matchers) require a config file.

This spec supersedes the earlier `--on '<match>' '<cmd>'` examples in SPECS.md §4, which are incompatible with cobra's `StringArray` flag model (cobra treats each space-separated quoted string as a separate value).

---

## Syntax

```
--on '<match> :: <action> [:: <action>] [:: <option>]'
```

A single quoted string containing **segments** separated by ` :: ` (space-colon-colon-space). The first segment is always the **match expression**; subsequent segments are **actions** or **options**, identified by prefix.

### Segments

| Position | Role | Required | Prefix |
|----------|------|----------|--------|
| 1st | Match expression | Yes | Auto-detected or explicit (`jq:`, `glob:`, `regex:`, `template:`) |
| 2nd+ | Action or option | At least one | `run:`, `respond:`, `timeout:`, or bare keyword (`exclusive`) |

### Why `::` ?

The separator must not collide with shell syntax, Go template delimiters, JSON, or jq expressions. `::` is unused in all of these, visually distinct, and easy to type. A single `:` would conflict with JSON keys, jq filters like `.[0:3]`, and Go template function calls.

---

## Match Expressions

The first segment is the match expression. xwebs auto-detects the match type from the expression's leading characters, or you can use an explicit prefix.

### Auto-Detection Rules

| First character(s) | Detected type | Rationale |
|---------------------|---------------|-----------|
| `.` | `jq` | jq expressions start with `.` |
| `*` or `?` | `glob` | Glob wildcards |
| `^` or `(` | `regex` | Regex anchors/groups |
| Anything else | `glob` | Safe default — literal strings are valid globs |

### Examples by Match Type

**Exact match** — a literal string is treated as a glob; it matches only that exact message:

```bash
# Matches only the exact string "ping"
--on 'ping :: respond:pong'

# Matches only this exact JSON string
--on '{"type":"ping"} :: respond:{"type":"pong"}'
```

**Glob** — use `*` and `?` wildcards:

```bash
# Any message containing "error" anywhere
--on '*error* :: run:notify-send "Error" "{{.Message}}"'

# Any message starting with "cmd:"
--on 'cmd:* :: run:handle-cmd.sh'

# Match everything
--on '* :: run:logger.sh'
```

**jq** — auto-detected when the expression starts with `.`:

```bash
# Match on a field value
--on '.type == "ping" :: respond:{"type":"pong"}'

# Match on nested field
--on '.event.level == "critical" :: run:alert.sh'

# Match on field existence
--on '.error != null :: run:handle-error.sh'
```

**Regex** — auto-detected when the expression starts with `^` or `(`:

```bash
# Match messages starting with a specific prefix
--on '^CMD: :: run:dispatch.sh'

# Match one of several message types
--on '^(ping|heartbeat)$ :: respond:{"type":"pong"}'
```

**Template** — requires explicit prefix; use for multi-condition logic:

```bash
--on 'template:{{eq (.Message | jq ".level") "critical"}} :: run:alert.sh'
```

### Explicit Prefixes

Override auto-detection with `<type>:<expression>`:

```bash
# Force jq even though expression doesn't start with "."
--on 'jq:true :: run:log-all.sh'

# Force glob on a string that starts with "^" literally
--on 'glob:^start* :: run:handle.sh'
```

The prefix is stripped before the expression is compiled. `jq:.type == "ping"` compiles as jq expression `.type == "ping"`.

### Match Shorthand

`*` alone matches every message (equivalent to `glob:*` or `jq:true`).

---

## Actions

Actions follow the match expression. Multiple actions execute in order.

### `run:<command>`

Execute a shell command via `sh -c`. The raw incoming message is piped to stdin. Template expressions in the command are expanded before execution.

```bash
--on '.type == "data" :: run:echo "{{.Message | jq ".payload"}}" >> data.jsonl'
```

The command's stdout is captured as `{{.Stdout}}`, stderr as `{{.Stderr}}`, and exit code as `{{.ExitCode}}` — all available in a subsequent `respond:` segment.

### `respond:<template>`

Send a response message. The template is expanded with the full handler context (`.Message`, `.Stdout`, `.Stderr`, `.ExitCode`, `.Vars`, `.ConnectionID`, etc.).

```bash
--on '.type == "ping" :: respond:{"type":"pong","ts":"{{now | formatTime "RFC3339"}}"}'
```

If both `run:` and `respond:` are present, `run:` executes first and its output is available in the `respond:` template.

### `timeout:<duration>`

Set a timeout for the `run:` command. Uses Go duration syntax (`5s`, `1m30s`, `500ms`).

```bash
--on '.type == "query" :: run:slow-query.sh :: respond:{"result":{{.Stdout | toJSON}}} :: timeout:10s'
```

If the command exceeds the timeout, it is killed via `context.WithTimeout`. The exit code will be non-zero and stderr may contain signal information.

### `exclusive`

Stop checking further handlers if this one matches. Equivalent to `exclusive: true` in YAML config.

```bash
--on '.type == "auth" :: run:verify-token.sh :: respond:{"auth":"ok"} :: exclusive'
```

---

## The `--respond` Flag

`--respond <template>` sets a **default response template for all inline handlers that do not specify their own response**. It is not a catch-all for unmatched messages — it only applies to `--on` handlers that match but have no `respond:` segment.

### Syntax

```bash
--respond '<template>'
```

### Behavior

1. When a message arrives, xwebs evaluates all `--on` handlers in order.
2. For each matching handler:
   - If the handler has its own `respond:` segment, that template is used for the response.
   - If the handler has **no `respond:` segment**, the `--respond` default template is used instead.
3. If **no handler matches**, no response is sent — `--respond` does not fire for unmatched messages.
4. When `--respond` fires, it has access to the same template context as any `respond:` segment, including `.Stdout`, `.Stderr`, and `.ExitCode` from the handler's `run:` command (if any).

### `--respond` With `run:`-Only Handlers

The primary use case: run a command and respond with a shared template that uses the command's output:

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: run:echo pong' \
  --on '.type == "status" :: run:get-status.sh' \
  --respond '{"output":"{{.Stdout | trim}}","ok":{{eq .ExitCode 0}}}'
```

Both handlers match their respective messages, run their commands, and respond using the same `--respond` template. Each response contains that handler's own `.Stdout` and `.ExitCode`.

### `--respond` With Mixed Handlers

Handlers with their own `respond:` segment are unaffected by `--respond`:

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong"}' \
  --on '.type == "query" :: run:query.sh' \
  --respond '{"result":{{.Stdout | toJSON}}}'
```

`ping` messages use their own `respond:` and ignore `--respond`. `query` messages run `query.sh` and respond using the `--respond` template. Messages that match neither handler get no response.

### `--respond` Without Any `--on`

When used without any `--on` flags, `--respond` has nothing to attach to and has no effect. Use `--on '* :: respond:<template>'` to reply to every message:

```bash
# Reply to every message
xwebs serve --port 8080 \
  --on '* :: respond:{"echo":{{.Message | toJSON}},"ts":"{{now}}"}'

# Or with a run: command for every message
xwebs serve --port 8080 \
  --on '* :: run:process.sh' \
  --respond '{"output":"{{.Stdout | trim}}"}'
```

### `--respond` Is Not a Fallback Handler

`--respond` does not add a new handler to the pipeline. It only fills in the missing `respond:` for handlers that don't have one. Messages that match no `--on` handler are silently dropped — `--respond` cannot change this.

---

## Execution Order

1. Message arrives.
2. Each `--on` handler is checked in command-line order.
3. For each matching handler:
   - `run:` executes (if present).
   - If the handler has a `respond:` segment, send that response.
   - Otherwise, if `--respond` is set, send the `--respond` template (with access to `.Stdout`/`.Stderr`/`.ExitCode` from step 3a).
4. If the matched handler has `exclusive`, stop checking further handlers.
5. If no handler matched, no response is sent.

Multiple `--on` handlers can match the same message (unless `exclusive` is used). Each matching handler runs independently.

---

## Client Mode vs Server Mode

Inline handlers work identically in both modes. The difference is which template context variables are available.

### Client Mode (`xwebs connect`)

Handlers trigger on messages received from the server.

```bash
xwebs connect wss://api.example.com \
  --on '.type == "error" :: run:notify-send "WS Error" "{{.Message | jq ".detail"}}"' \
  --on '.type == "data" :: run:echo {{.Message | jq ".payload"}} >> data.jsonl'
```

Available context: `.Message`, `.MessageLen`, `.MessageIndex`, `.ConnectionID`, `.Vars`, `.Stdout`, `.Stderr`, `.ExitCode`, `.DurationMs`, `.Timestamp`.

### Server Mode (`xwebs serve`)

Handlers trigger on messages received from connected clients.

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong","ts":"{{now}}"}' \
  --on '.type == "query" :: run:handle-query.sh :: timeout:10s' \
  --respond '{"result":{{.Stdout | toJSON}}}'
```

Here `ping` uses its own `respond:`. `query` runs a command and gets the `--respond` template as its response (with `.Stdout` from `handle-query.sh`).

Available context: everything in client mode plus `.RemoteAddr`, `.ClientCount`, `.ConnectionID` (per-client).

---

## Mapping to YAML Config

Every inline handler maps to a single YAML handler entry. The `::` segments map as follows:

| Inline segment | YAML field |
|----------------|------------|
| Match expression (1st segment) | `match:` with auto-detected sub-key (`jq:`, `glob:`, `regex:`, `template:`) |
| `run:<cmd>` | `run:` |
| `respond:<template>` | `respond:` |
| `timeout:<duration>` | `timeout:` |
| `exclusive` | `exclusive: true` |
| `--respond` flag | Default `respond:` injected into any handler that has no `respond:` segment |

### Example Equivalence

**Inline:**

```bash
xwebs serve --port 8080 \
  --on '.type == "ping" :: respond:{"type":"pong"} :: exclusive' \
  --on '.type == "data" :: run:process.sh :: respond:{"ok":true} :: timeout:5s' \
  --respond '{"error":"unknown"}'
```

**YAML equivalent:**

```yaml
handlers:
  - name: inline-1
    match:
      jq: '.type == "ping"'
    respond: '{"type":"pong"}'   # has its own respond, --respond is ignored
    exclusive: true

  - name: inline-2
    match:
      jq: '.type == "data"'
    run: process.sh
    respond: '{"error":"unknown"}'  # --respond injected as respond: (no run: output here)
    timeout: 5s
```

Note: `inline-1` has its own `respond:` so the `--respond` default does not apply to it. `inline-2` has no `respond:` so the `--respond` template is injected. Unmatched messages produce no response — there is no catch-all handler.

---

## What Requires a Config File

These YAML handler features have no inline equivalent:

| Feature | Why |
|---------|-----|
| `pipeline:` (multi-step with named steps) | Too complex for a single string |
| `retry:` (count, backoff, initial) | Multiple sub-fields |
| `rate_limit:` | Per-handler rate limiting |
| `debounce:` | Timing configuration |
| `priority:` (explicit numeric) | Inline handlers use command-line order |
| `concurrent: false` | Serialization control |
| Composite matchers (`and:`/`or:` with multiple conditions) | Nesting doesn't fit single-string syntax |
| `on_connect:`/`on_disconnect:`/`on_error:` lifecycle hooks | Not per-message handlers |
| `variables:` (global) | Use `--var key=value` flag instead |
| `name:` (handler naming) | Inline handlers are auto-named `inline-1`, `inline-2`, etc. |

For anything beyond match → run → respond, use `--config handlers.yaml`.

---

## Mixing `--on` and `--config`

Inline handlers and config file handlers can coexist. Inline handlers are appended after config file handlers in evaluation order:

```bash
xwebs serve --port 8080 \
  --config handlers.yaml \
  --on '.type == "debug" :: respond:{"debug":true}'
```

Config file handlers are checked first (in their defined order), then inline handlers (in command-line order). The `--respond` default fires last, after all handlers from both sources.

---

## Shell Quoting Guide

The entire `--on` value is a single shell string. Use single quotes for the outer wrapper and double quotes inside (for JSON, jq expressions):

```bash
# RECOMMENDED: Single quotes outside, double quotes inside
--on '.type == "ping" :: respond:{"type":"pong"}'
```

### JQ String Syntax Warning

JQ **requires** double quotes for string literals. Using single quotes inside a JQ expression (e.g., `.type == 'ping'`) will cause a `gojq` parsing error: `unexpected token "'"`.

If you must use double quotes for the outer shell wrapper, you must escape the inner JQ quotes:

```bash
# Alternative: Double quotes outside, escaped double quotes inside
--on ".type == \"ping\" :: respond:{\"type\":\"pong\"}"
```

### Complex Escaping

When the `run:` command itself needs single quotes, use `$'...'` (if supported by your shell) or nested escaping:

# Or split across lines with backslash continuation
xwebs serve --port 8080 \
  --on '.type == "query" :: run:psql -c "SELECT 1" :: respond:{"rows":{{.Stdout | toJSON}}}'

When templates contain double quotes inside JSON values, the shell's single-quote wrapper protects them. The Go template engine handles the inner double quotes.

---

## Quick Reference

```
--on '<match> :: run:<cmd> :: respond:<template> :: timeout:<dur> :: exclusive'
      ───┬───    ────┬────    ───────┬──────────    ──────┬──────    ────┬────
         │           │               │                    │              │
   jq/glob/regex   shell cmd     response msg      kill slow cmd   stop on match
   (auto-detect)   (optional)    (optional)         (optional)      (optional)

--respond '<template>'
           ─────┬─────
                │
     default response for --on handlers with no respond: segment
     (.Stdout/.Stderr/.ExitCode available from that handler's run:)
```
