---
title: sticky-broadcast
description: Broadcast a message to all subscribers of a topic and retain it so late-joining clients receive it immediately on subscription.
---

# `sticky-broadcast` — Sticky Broadcast

**Mode:** `[S]` (server only)

Delivers a message to all current subscribers of `topic:` and retains (caches) that message in memory. When a new client subscribes to the same topic, it immediately receives the retained message without waiting for the next publish event. Only the most recent message per topic is retained.

This is useful for "last known value" semantics — dashboards that must show the current state on load, presence indicators, or configuration pushed to clients.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `topic` | string | Yes | — | Topic name for broadcast and retention. Rendered as a Go template. |
| `message` | string | No | `.Message` | Payload to broadcast and retain. Rendered as a Go template. |

---

## Template Variables Injected

None. Standard context is available in `topic:` and `message:`.

---

## Examples

**Retain the latest server status for new subscribers:**

```yaml
handlers:
  - name: status-update
    match:
      jq: '.type == "status"'
    builtin: sticky-broadcast
    topic: "server-status"
    message: |
      {
        "type": "status",
        "state": "{{.Message | jq ".state"}}",
        "updated_at": "{{now | formatTime "RFC3339"}}"
      }
```

**Sticky presence — retain the last user count update:**

```yaml
handlers:
  - name: presence
    match:
      jq: '.type == "presence"'
    builtin: sticky-broadcast
    topic: "presence"
    message: '{"online": {{.ClientCount}}, "ts": "{{nowUnix}}"}'
    respond: '{"broadcast": true}'
```

**Per-room sticky last message:**

```yaml
handlers:
  - name: room-message
    match:
      jq: '.type == "room_msg"'
    builtin: sticky-broadcast
    topic: '{{.Message | jq ".room"}}'
    message: '{{.Message | toJSON}}'
```

---

## Edge Cases

- Only the single most recent message per topic is retained; previous retained values are overwritten on every publish.
- The retained message is stored in process memory and is lost on server restart. Use an external store (`redis-set`) for durable last-value semantics.
- A new subscriber receives the retained message synchronously at subscription time — before any subsequent publishes arrive.
- If `message:` renders to an empty string, the retained message is cleared (not retained) and existing subscribers receive nothing.
