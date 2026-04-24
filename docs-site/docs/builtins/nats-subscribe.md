---
title: nats-subscribe
description: Subscribe to a NATS subject and inject received messages into the handler pipeline.
---

# `nats-subscribe` — NATS Subscribe

**Mode:** `[S]` (server only)

Opens a persistent NATS subscription and injects received messages into the handler pipeline for forwarding to WebSocket clients.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | No | `nats://localhost:4222` | NATS server URL. |
| `subject` | string | Yes | — | NATS subject or wildcard (e.g. `events.>`, `device.*.status`). |
| `queue` | string | No | — | Optional NATS queue group name for load-balanced delivery. |

---

## Example

```yaml
handlers:
  - name: nats-to-ws
    match: "__nats_subscribe__"
    builtin: nats-subscribe
    url: "nats://localhost:4222"
    subject: "events.>"
    respond: '{"type": "nats_event", "data": {{.Message | toJSON}}}'
```

---

## Edge Cases

- Wildcard subjects: `*` matches a single token, `>` matches all remaining tokens.
- With `queue:` set, only one xwebs instance in the queue group receives each message — enabling horizontal scaling.
- The subscription is established at handler load time, not per WebSocket connection.
- NATS reconnects automatically; messages published during a reconnect window may be lost unless JetStream is used.
