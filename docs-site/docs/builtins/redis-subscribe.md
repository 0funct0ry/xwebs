---
title: redis-subscribe
description: Subscribe to a Redis Pub/Sub channel and inject received messages into the handler pipeline.
---

# `redis-subscribe` — Redis Subscribe

**Mode:** `[S]` (server only)

Opens a persistent Redis Pub/Sub subscription. Messages received from the Redis channel are injected into the handler pipeline as if they were incoming WebSocket messages — they are processed by the same `respond:` or `run:` fields and can be forwarded to connected clients.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `channel` | string | Yes | — | Redis Pub/Sub channel to subscribe to. Template expression evaluated once at handler load time. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Example

```yaml
handlers:
  - name: redis-to-ws
    match: "__redis_subscribe__"
    builtin: redis-subscribe
    channel: "live-events"
    respond: '{"type": "event", "data": {{.Message | toJSON}}}'
```

---

## Edge Cases

- The subscription is established when the handler is loaded, not on WebSocket connect. Messages published before any WebSocket client connects are still received by the handler (but `respond:` will have no clients to send to).
- Channel name templates are evaluated once at startup; dynamic channel names require a handler reload.
- If Redis disconnects, xwebs will attempt to reconnect with exponential backoff.
- Combined with `broadcast`, `redis-subscribe` can forward Redis messages to all connected WebSocket clients.
