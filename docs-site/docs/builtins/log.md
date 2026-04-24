---
title: log
description: Write a structured log entry to stdout, stderr, or a file.
---

# `log` — Structured Logging

**Mode:** `[*]` (client and server)

Writes a log entry without sending a WebSocket response. Useful for audit logging, debugging, and observability without requiring a shell command.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `message` | string | Yes | — | Log message template. Supports template expressions. |
| `level` | string | No | `"info"` | Log level: `debug`, `info`, `warn`, `error` |
| `target` | string | No | `"stdout"` | Output: `stdout`, `stderr`, or a file path |
| `format` | string | No | `"text"` | Log format: `text` or `json` |

---

## Examples

**Audit log to a file:**

```yaml
handlers:
  - name: audit-log
    match: "*"
    builtin: log
    message: '{{.ConnectionID}} received: {{.Message | compactJSON}}'
    target: /var/log/xwebs/audit.log
    format: json
```

**Log errors to stderr:**

```yaml
handlers:
  - name: error-logger
    match:
      jq: '.error != null'
    builtin: log
    message: 'Error from {{.ConnectionID}}: {{.Message | jq ".error"}}'
    level: error
    target: stderr
```

**Combined log and respond:**

```yaml
handlers:
  - name: log-and-reply
    match:
      jq: '.type == "deploy"'
    builtin: log
    message: 'Deploy requested by {{.ConnectionID}} for {{.Message | jq ".env"}}'
    level: info
    respond: '{"status":"queued"}'
```

---

## REPL Example

```
xwebs> :handler add -m "*" --builtin log --log-message "received: {{.Message}}"
```
