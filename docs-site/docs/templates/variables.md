---
title: Template Variables
description: All template variables available in xwebs handler configs, CLI flags, and REPL commands.
---

# Template Variables

Go template expressions work in handler config fields (`run:`, `respond:`, `match.template:`, `topic:`, etc.), CLI flags (`--on`, `--respond`, `--header`), and REPL commands (`:sendt`, `:prompt set`).

All variables are accessed with `{{.VariableName}}` syntax.

---

## Connection Context

Available in all templates (client and server mode).

| Variable | Type | Description | Example value |
|----------|------|-------------|---------------|
| `{{.URL}}` | string | Full WebSocket URL | `wss://api.example.com/ws` |
| `{{.Host}}` | string | Remote hostname | `api.example.com` |
| `{{.Path}}` | string | URL path | `/ws` |
| `{{.Scheme}}` | string | Protocol scheme | `ws` or `wss` |
| `{{.RemoteAddr}}` | string | Remote IP:port (server mode) | `192.168.1.42:5891` |
| `{{.LocalAddr}}` | string | Local IP:port | `0.0.0.0:8080` |
| `{{.ConnectionID}}` | string | Unique connection identifier | `ns-abc-123` |
| `{{.Subprotocol}}` | string | Negotiated WebSocket subprotocol | `graphql-ws` |
| `{{.Headers}}` | map[string][]string | HTTP handshake headers | — |

```yaml
respond: |
  {
    "connected_to": "{{.Host}}",
    "path": "{{.Path}}",
    "connection_id": "{{.ConnectionID}}"
  }
```

---

## Message Context

Available whenever a message is being processed.

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Message}}` | string | Raw message text |
| `{{.MessageBytes}}` | []byte | Raw message bytes |
| `{{.MessageLen}}` | int | Message length in bytes |
| `{{.MessageType}}` | string | `text` or `binary` |
| `{{.MessageIndex}}` | int | Sequential message number (0-based, per connection) |
| `{{.Timestamp}}` | time.Time | Time the message was received |

```yaml
# Log every message with metadata
- name: logger
  match: "*"
  run: |
    echo '{"idx":{{.MessageIndex}},"len":{{.MessageLen}},"type":"{{.MessageType}}","msg":{{.Message | toJSON}}}' >> messages.jsonl
```

---

## Handler Execution Context

Available in `respond:` templates after a `run:` command or `pipeline:`.

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Stdout}}` | string | Shell command stdout (includes trailing newline) |
| `{{.Stderr}}` | string | Shell command stderr |
| `{{.ExitCode}}` | int | Shell command exit code (0 = success) |
| `{{.DurationMs}}` | int64 | Execution time in milliseconds |
| `{{.Steps.<name>.Stdout}}` | string | Pipeline step stdout (via `as: <name>`) |
| `{{.Steps.<name>.Stderr}}` | string | Pipeline step stderr |
| `{{.Steps.<name>.ExitCode}}` | int | Pipeline step exit code |

```yaml
- name: run-and-respond
  match: "*"
  run: hostname
  respond: '{"host":"{{.Stdout | trim}}","took_ms":{{.DurationMs}},"ok":{{eq .ExitCode 0}}}'
```

For pipeline handlers, access individual step outputs:

```yaml
respond: |
  {
    "step1": "{{.Steps.fetch.Stdout | trim}}",
    "step2": "{{.Steps.process.Stdout | trim}}"
  }
```

---

## Builtin-Specific Context

Certain builtins inject additional variables into the `respond:` template:

| Variable | Builtin | Description |
|----------|---------|-------------|
| `{{.KvValue}}` | `kv-get` | Value retrieved from the KV store |
| `{{.KvKeys}}` | `kv-list` | JSON array of all KV keys |
| `{{.ForwardReply}}` | `forward` | Response from the upstream WebSocket |
| `{{.HttpBody}}` | `http`, `http-get`, `webhook` | HTTP response body |
| `{{.HttpStatus}}` | `http`, `webhook` | HTTP response status code (int) |
| `{{.RetryAfter}}` | `rate-limit` | Seconds until the rate limit resets |
| `{{.RedisValue}}` | `redis-get`, `redis-rpop`, `redis-incr` | Redis operation result |
| `{{.Rows}}` | `sqlite`, `postgres`, `csv-parse` | Query result rows (slice of maps) |
| `{{.S3Body}}` | `s3-get` | S3 object content |
| `{{.OllamaReply}}` | `ollama-generate`, `ollama-chat` | Model response text |
| `{{.Label}}` | `ollama-classify` | Classification result label |
| `{{.ValidationErrors}}` | `schema-validate` | List of validation errors |
| `{{.GraphQLErrors}}` | `http-graphql` | GraphQL error array |
| `{{.Embedding}}` | `ollama-embed` | Embedding vector |

---

## Server Context

Available in server mode only.

| Variable | Type | Description |
|----------|------|-------------|
| `{{.ClientCount}}` | int | Number of currently connected clients |
| `{{.Clients}}` | []Client | Slice of all connected client objects |
| `{{.ServerUptime}}` | duration | Time since server started |

```yaml
- name: status
  match:
    jq: '.type == "status"'
  respond: |
    {
      "clients": {{.ClientCount}},
      "uptime": "{{.ServerUptime}}"
    }
```

---

## Session / Environment

| Variable | Type | Description | Mode |
|----------|------|-------------|------|
| `{{.Vars.<key>}}` | any | Session variables set via `:set` | Client only |
| `{{.Session.<key>}}` | any | Alias for `.Vars` | Client only |
| `{{.Env.<KEY>}}` | string | Process environment variables | Both |
| `{{.Config.<key>}}` | any | Values from the xwebs config file | Both |

```bash
# After :set env staging in the client REPL:
:sendt {"env":"{{.Vars.env}}","host":"{{.Host}}"}
```

```yaml
# Use an environment variable in a handler
run: curl -H "Authorization: Bearer {{.Env.API_TOKEN}}" https://api.example.com/data
```

---

## Key-Value Store

In server mode, read from the KV store in any template using the `kv` function:

```yaml
respond: '{"maintenance":{{kv "maintenance_mode"}}}'
```

The `kv` function is read-only in templates. To write, use the `kv-set` builtin.

---

## Prompt-Specific Variables

These additional variables are available only in `:prompt set` templates:

| Variable | Description |
|----------|-------------|
| `{{.LastLatencyMs}}` | Round-trip time of the last message in milliseconds |

```bash
:prompt set "{{.Host}} [{{if gt .LastLatencyMs 200}}{{red (print .LastLatencyMs "ms")}}{{else}}{{green (print .LastLatencyMs "ms")}}{{end}}] > "
```
