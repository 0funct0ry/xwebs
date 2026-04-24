---
title: broadcast-others
description: Send a message to all connected clients except the sender — classic chat-room fan-out.
---

# `broadcast-others` — Broadcast to Others

**Mode:** `[S]` (server only)

Sends a message to all connected clients **except the sender**. Classic pattern for chat rooms and collaborative editing — the sender sees their own message locally, so you avoid double-display.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `message` | string | No | `.Message` (raw incoming) | Template for the broadcast payload. |

---

## Examples

**Simple chat relay:**

```yaml
handlers:
  - name: chat-relay
    match:
      jq: '.type == "message"'
    builtin: broadcast-others
    message: |
      {
        "type": "message",
        "from": "{{.ConnectionID}}",
        "text": "{{.Message | jq ".text"}}",
        "ts": "{{now | formatTime "RFC3339"}}"
      }
    respond: '{"delivered":true}'
```

**Presence announcements:**

```yaml
on_connect:
  - builtin: broadcast-others
    message: '{"type":"join","user":"{{.ConnectionID}}"}'

on_disconnect:
  - builtin: broadcast-others
    message: '{"type":"leave","user":"{{.ConnectionID}}"}'
```

**Collaborative document editing:**

```yaml
handlers:
  - name: edit-propagate
    match:
      jq: '.type == "edit"'
    builtin: broadcast-others
    message: '{{.Message | setJSON "from" .ConnectionID | toJSON}}'
```

---

## Edge Cases

- If there is only one connected client (the sender), no message is delivered. No error is returned.
- The message template has access to the full handler context, including `.ConnectionID` (the sender's ID).
