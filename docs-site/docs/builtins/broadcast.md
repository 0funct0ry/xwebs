---
title: broadcast
description: Send a message to all connected clients.
---

# `broadcast` — Broadcast to All Clients

**Mode:** `[S]` (server only)

Broadcasts a message to every connected client, including the sender. Use `broadcast-others` to exclude the sender.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `message` | string | No | `.Message` (raw incoming) | Template to render as the broadcast payload. Defaults to forwarding the received message verbatim. |

---

## Examples

**Forward every incoming message to all clients (fan-out relay):**

```yaml
handlers:
  - name: relay
    match: "*"
    builtin: broadcast
```

**Broadcast a structured announcement:**

```yaml
handlers:
  - name: announce
    match:
      jq: '.type == "announce"'
    builtin: broadcast
    message: |
      {
        "type": "announcement",
        "text": "{{.Message | jq ".text"}}",
        "from": "{{.ConnectionID}}",
        "ts": "{{now | formatTime "RFC3339"}}"
      }
    respond: '{"broadcast":"sent","recipients":{{.ClientCount}}}'
```

**Broadcast with a respond: confirmation to the sender:**

```yaml
handlers:
  - name: chat
    match:
      jq: '.type == "chat"'
    builtin: broadcast
    respond: '{"delivered":true}'
```

---

## REPL Example

```
xwebs> :broadcast {"type":"maintenance","eta":"5m"}
→ broadcast to 3 clients
```

Or broadcast a Go template:

```
xwebs> :broadcast -t {"type":"ping","server_time":"{{now}}"}
```

---

## Edge Cases

- When there are no connected clients, `broadcast` sends nothing and does not error.
- `broadcast` includes the sender. Use `broadcast-others` for chat-room semantics where the sender shouldn't see their own message.
- The `message:` field is rendered as a Go template — all template functions are available.
- If the message template fails to render, no broadcast is sent and the error is logged.
