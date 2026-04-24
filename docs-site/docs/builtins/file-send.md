---
title: file-send
description: Read a local file and send its contents as a text or binary WebSocket frame.
---

# `file-send` — Send Local File

**Mode:** `[C]` (client only)

Reads a file from the local filesystem and sends its contents as a WebSocket frame. The `path:` field supports template expressions, so the file path can be derived from message content. Set `binary: true` to send the file as a binary frame (opcode `0x2`); the default sends a text frame.

This builtin is useful in client automation scripts where you need to upload files, send binary blobs, or replay stored payloads without spawning a shell.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | — | Filesystem path to the file to send. Rendered as a Go template. |
| `binary` | bool | No | `false` | When `true`, sends the file contents as a binary frame. When `false`, sends as a UTF-8 text frame. |

---

## Template Variables Injected

`file-send` does not inject additional template variables. The standard connection context is available in `path:`.

---

## Examples

**Send a static JSON file on every matching message:**

```yaml
handlers:
  - name: send-payload
    match:
      jq: '.type == "upload"'
    builtin: file-send
    path: payloads/upload.json
```

**Derive the file path from the incoming message:**

```yaml
handlers:
  - name: dynamic-file
    match:
      jq: '.type == "fetch"'
    builtin: file-send
    path: 'fixtures/{{.Message | jq ".name"}}.json'
```

**Send a binary file (e.g., image or protobuf):**

```yaml
handlers:
  - name: send-image
    match:
      jq: '.type == "get_image"'
    builtin: file-send
    path: 'assets/{{.Message | jq ".id"}}.png'
    binary: true
```

---

## Edge Cases

- If the file does not exist or cannot be read, the handler logs an error and sends no frame.
- `binary: false` sends raw file bytes as a text frame; if the file contains non-UTF-8 bytes, some WebSocket servers will reject it. Use `binary: true` for non-text content.
- The entire file is read into memory before sending. Avoid sending very large files (>64 MB) without testing the server's `max_message_size` setting.
- `file-send` is client-only; use `file-write` in server mode to persist file content received from clients.
