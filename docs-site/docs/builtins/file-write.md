---
title: file-write
description: Write the incoming message or a template expression to a file on disk.
---

# `file-write` — Write to File

**Mode:** `[*]` (client and server)

Writes data to a file. By default it writes the raw incoming message; the optional `content:` field accepts a template expression for writing computed or transformed content. The `mode:` field controls whether the file is overwritten or appended. The `path:` field is template-rendered so the destination can be derived from message content.

Use `file-write` to persist messages to disk, build audit logs, generate files on the fly, or act as a lightweight sink in a pipeline.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | — | Destination file path. Rendered as a Go template. |
| `mode` | string | No | `overwrite` | Write mode: `overwrite` (truncates the file first) or `append` (adds to the end). |
| `content` | string | No | `.Message` | Template expression for what to write. Defaults to the raw incoming message. |

---

## Template Variables Injected

`file-write` does not inject additional template variables. Standard connection and message context is available in `path:` and `content:`.

---

## Examples

**Persist every message to a log file (append mode):**

```yaml
handlers:
  - name: message-log
    match: "*"
    builtin: file-write
    path: logs/messages.jsonl
    mode: append
    content: "{{.Message}}\n"
```

**Write rendered content to a per-connection file:**

```yaml
handlers:
  - name: save-session
    match:
      jq: '.type == "save"'
    builtin: file-write
    path: 'sessions/{{.ConnectionID}}.json'
    mode: overwrite
    content: |
      {
        "connection_id": "{{.ConnectionID}}",
        "saved_at": "{{now | formatTime "RFC3339"}}",
        "data": {{.Message | jq ".payload" | toJSON}}
      }
    respond: '{"saved": true}'
```

**Overwrite a config file based on incoming message:**

```yaml
handlers:
  - name: update-config
    match:
      jq: '.type == "config_push"'
    builtin: file-write
    path: '/etc/myapp/config.json'
    mode: overwrite
    content: '{{.Message | jq ".config" | prettyJSON}}'
    respond: '{"updated": true}'
```

---

## Edge Cases

- Parent directories must exist; `file-write` does not create missing intermediate directories.
- In `append` mode, a trailing newline is not added automatically — include `\n` in `content:` if needed.
- If the template in `content:` fails to render, nothing is written to disk and the error is logged.
- In server mode with concurrent handlers, simultaneous writes to the same path are not serialised — use `concurrent: false` on the handler or route writes through a single handler to avoid interleaving.
