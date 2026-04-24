---
title: Handler YAML Schema
description: Complete reference for handlers.yaml — every field, all three execution formats, and five working examples.
---

# Handler YAML Schema

Handler configuration lives in a YAML file passed via `--handlers handlers.yaml`. It defines how xwebs matches incoming WebSocket messages and what actions to take in response.

## Root Structure

```yaml
variables:       # Optional: global variables accessible as {{.Vars.<key>}}
  key: value

on_connect:      # Optional: actions run when a connection is established
  - run: echo "Connected"

on_disconnect:   # Optional: actions run when a connection closes
  - run: echo "Disconnected (code={{.CloseCode}})"

on_error:        # Optional: actions run on connection errors
  - run: echo "Error: {{.Error}}"

handlers:        # Required: list of message handlers
  - name: example
    match: "*"
    respond: '{"echo":{{.Message | toJSON}}}'
```

### `variables` Block

Global variables available in all templates as `{{.Vars.<key>}}`:

```yaml
variables:
  app_name: "myapp"
  log_dir: "/var/log/myapp"
  api_url: "https://api.internal.example.com"
```

### Lifecycle Hooks

`on_connect`, `on_disconnect`, and `on_error` accept a list of simple actions (not full pipelines):

```yaml
on_connect:
  - run: echo "Connected to {{.URL}} at {{now}}" >> {{.Vars.log_dir}}/connections.log
  - send: '{"auth":"{{env "WS_TOKEN"}}"}'

on_disconnect:
  - run: echo "Disconnected (code={{.CloseCode}})" >> {{.Vars.log_dir}}/connections.log

on_error:
  - run: 'echo "ERROR: {{.Error}}" | mail -s "xwebs alert" ops@example.com'
```

---

## Handler Fields

Each item in `handlers:` is a handler object.

### Core Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | — | Unique identifier used in logs, metrics, and REPL commands |
| `match` | Matcher | Yes | — | Condition to test against each incoming message |
| `priority` | int | No | `0` | Higher numbers are checked first; ties resolved by config order |
| `exclusive` | bool | No | `false` | Stop checking further handlers if this one matches |
| `concurrent` | bool | No | `true` | Set to `false` to serialize executions for this handler |
| `timeout` | string | No | `30s` | Per-handler execution timeout (Go duration: `10s`, `1m30s`) |
| `retry` | object | No | — | Retry config: `count`, `backoff` (`exponential`/`linear`), `initial` |
| `rate_limit` | string | No | — | Rate limit: `"10/s"`, `"100/m"`, `"1000/h"` |
| `debounce` | string | No | — | Debounce window: `"500ms"` |

---

## Three Execution Formats

Choose **one** execution format per handler.

### Format 1: Shorthands

The most common format — one command, one response.

| Field | Type | Description |
|-------|------|-------------|
| `run` | string | Shell command executed via `sh -c`; stdout available as `{{.Stdout}}` |
| `respond` | string | Template sent back to the client after `run` completes |
| `builtin` | string | Execute a [builtin](../builtins/) instead of a shell command |

```yaml
- name: healthcheck
  match:
    json_path: "$.type"
    equals: "ping"
  respond: |
    {
      "type": "pong",
      "server_time": "{{now | formatTime "RFC3339"}}",
      "hostname": "{{hostname}}"
    }
```

```yaml
- name: query-db
  match:
    jq: '.action == "query"'
  run: |
    psql -h localhost -U app -d mydb -t -A \
      -c "{{.Message | jq ".sql" | shellEscape}}"
  respond: '{"result":{{.Stdout | toJSON}},"duration_ms":{{.DurationMs}}}'
  timeout: 30s
```

### Format 2: Pipeline

Multi-step execution with named intermediate results.

| Field | Type | Description |
|-------|------|-------------|
| `pipeline` | list | Ordered list of steps |
| `respond` | string | Template sent after all steps complete |

**Step object fields:**

| Field | Type | Description |
|-------|------|-------------|
| `run` | string | Shell command for this step |
| `builtin` | string | Builtin to execute for this step |
| `as` | string | Store step output under `{{.Steps.<as>}}` |
| `timeout` | string | Per-step timeout |

```yaml
- name: health-aggregator
  match:
    jq: '.type == "health_check"'
  pipeline:
    - run: curl -s http://api:8080/health
      as: api
    - run: curl -s http://auth:8081/health
      as: auth
    - run: curl -s http://worker:8082/health
      as: worker
  respond: |
    {
      "type": "health_status",
      "services": {
        "api": {{.Steps.api.Stdout}},
        "auth": {{.Steps.auth.Stdout}},
        "worker": {{.Steps.worker.Stdout}}
      },
      "checked_at": "{{now | formatTime "RFC3339"}}"
    }
```

### Format 3: Verbose Actions

Full control — mix shell commands, sends, and log entries with per-step `env:` and `silent:`.

| Field | Type | Description |
|-------|------|-------------|
| `actions` | list | Ordered list of action objects |

**Action object fields:**

| Field | Type | Description |
|-------|------|-------------|
| `action` | string | Explicit type: `shell`, `send`, `log`, `builtin` (inferred if omitted) |
| `run` / `command` | string | Shell command or builtin to execute |
| `send` / `message` | string | Payload to send to the client |
| `log` / `message` | string | Text to print or log |
| `target` | string | Log target: `stdout`, `stderr`, or a file path |
| `timeout` | string | Per-action timeout |
| `env` | map | Environment variables injected into shell execution |
| `silent` | bool | Suppress implicit stdout/stderr printing |

```yaml
- name: upload-handler
  match: '*"action":"upload"*'
  actions:
    - log: "Upload started: size={{.MessageLen}}"
      target: stderr
    - run: ./process_and_upload.sh "{{.Message | base64Encode}}"
      env:
        S3_BUCKET: production-data
        AWS_REGION: us-west-2
      timeout: 60s
      silent: true
    - send: '{"status":"upload_complete","code":{{.ExitCode}}}'
```

---

## Match Strategies

### Glob

Wildcard matching against the raw message text:

```yaml
match: "*ping*"         # contains "ping"
match: "cmd:*"          # starts with "cmd:"
match: "*"              # matches everything
```

### Regex

PCRE-compatible regular expression:

```yaml
match:
  regex: '^\{"type":\s*"ping"'
```

### jq Boolean

A jq expression that returns `true` to match:

```yaml
match:
  jq: '.type == "deploy" and .env == "production"'
```

```yaml
match:
  jq: '.error != null'
```

### JSON Path + Equals

Extract a field value and compare it:

```yaml
match:
  json_path: "$.type"
  equals: "ping"
```

### JSON Schema

Validate the message against a JSON Schema file:

```yaml
match:
  json_schema: "schemas/user_event.json"
```

### Go Template (Truthy)

A template expression that evaluates to a truthy value:

```yaml
match:
  template: '{{gt .MessageLen 100}}'
```

```yaml
match:
  template: '{{and (isJSON .Message) (contains "error" .Message)}}'
```

### Binary Frame

Match only binary or only text frames:

```yaml
match:
  binary: true    # binary frames only
```

```yaml
match:
  binary: false   # text frames only
```

### Composite AND

All conditions must match:

```yaml
match:
  all:
    - regex: '^\{"type":'
    - jq: '.version >= 2'
    - binary: false
```

### Composite OR

Any condition matches:

```yaml
match:
  any:
    - glob: "*admin*"
    - glob: "*superuser*"
    - jq: '.role == "owner"'
```

---

## Complete Examples

### Example 1: Echo Server

```yaml
handlers:
  - name: echo
    match: "*"
    builtin: echo
```

### Example 2: Ping/Pong with Metadata

```yaml
handlers:
  - name: ping
    match:
      jq: '.type == "ping"'
    respond: |
      {
        "type": "pong",
        "server_time": "{{now | formatTime "RFC3339"}}",
        "uptime": "{{uptime}}",
        "client_id": "{{.ConnectionID}}"
      }
    exclusive: true
```

### Example 3: Deploy Trigger

```yaml
variables:
  app_name: "myapp"
  deploy_dir: "/opt/myapp"

handlers:
  - name: deploy
    match:
      jq: '.type == "deploy" and .env == "production"'
    run: |
      cd {{.Vars.deploy_dir}} && \
      git pull origin {{.Message | jq ".branch"}} && \
      make deploy ENV={{.Message | jq ".env"}}
    respond: |
      {
        "status": "deployed",
        "branch": "{{.Message | jq ".branch"}}",
        "commit": "{{shell "git rev-parse --short HEAD"}}"
      }
    timeout: 120s
    concurrent: false
```

### Example 4: Multi-Step Data Pipeline

```yaml
handlers:
  - name: transform-data
    match:
      jq: '.action == "transform"'
    pipeline:
      - run: echo '{{.Message | jq ".payload"}}' | tr '[:lower:]' '[:upper:]'
        as: uppercased
      - run: wc -c <<< "{{.Steps.uppercased.Stdout | trim}}"
        as: counted
    respond: |
      {
        "uppercased": "{{.Steps.uppercased.Stdout | trim}}",
        "byte_count": {{.Steps.counted.Stdout | trim}}
      }
```

### Example 5: Rate-Limited KV Store

```yaml
handlers:
  - name: kv-set
    match:
      jq: '.type == "set"'
    builtin: kv-set
    key: '{{.Message | jq ".key"}}'
    value: '{{.Message | jq ".value"}}'
    rate_limit: "100/m"
    respond: '{"stored":true,"key":"{{.Message | jq ".key"}}"}'

  - name: kv-get
    match:
      jq: '.type == "get"'
    builtin: kv-get
    key: '{{.Message | jq ".key"}}'
    respond: '{"key":"{{.Message | jq ".key"}}","value":"{{.KvValue}}"}'

  - name: catch-all
    match: "*"
    priority: -1
    respond: '{"error":"unknown_message"}'
```
