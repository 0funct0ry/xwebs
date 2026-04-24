---
title: close
description: Send a WebSocket close frame with a configurable status code and reason string.
---

# `close` — Send Close Frame

**Mode:** `[*]` (client and server)

Sends a WebSocket close frame (opcode `0x8`) to the peer with a numeric status code and an optional reason string. After the close frame is sent, the connection is terminated. The `reason:` field is rendered as a Go template so the message can include dynamic content.

Use `close` to implement clean protocol shutdowns, reject invalid sessions, or enforce server-side timeouts without leaving connections in a half-open state.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `code` | integer | No | `1000` | WebSocket close status code (RFC 6455). Common values: `1000` (normal), `1001` (going away), `1003` (unsupported data), `1008` (policy violation), `4000`–`4999` (application-defined). |
| `reason` | string | No | `""` | Human-readable close reason. Rendered as a Go template. Maximum 123 bytes per the WebSocket spec. |

---

## Template Variables Injected

None. Standard message and connection context is available in `reason:`.

---

## Examples

**Normal closure after processing a terminal message:**

```yaml
handlers:
  - name: handle-done
    match:
      jq: '.type == "done"'
    builtin: close
    code: 1000
    reason: "Session complete"
```

**Policy violation — close unauthenticated connections:**

```yaml
handlers:
  - name: reject-unauth
    match:
      jq: '.auth == null'
    builtin: close
    code: 1008
    reason: "Authentication required"
```

**Dynamic close reason from message content:**

```yaml
handlers:
  - name: server-shutdown
    match:
      jq: '.type == "shutdown"'
    builtin: close
    code: 1001
    reason: 'Server shutting down: {{.Message | jq ".reason"}}'
```

---

## Edge Cases

- After `close` sends the frame, no further handlers run and no `respond:` template is evaluated for that message.
- If the peer closes the connection simultaneously, a `close` builtin may silently succeed — the WebSocket handshake is already complete.
- Application-defined codes must be in the range `4000`–`4999`; codes in `1004`, `1005`, and `1006` are reserved and must not be used.
- The `reason` string is truncated to 123 bytes to comply with the WebSocket protocol; longer values are silently cut.
