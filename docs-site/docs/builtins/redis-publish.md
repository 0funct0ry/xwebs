---
title: redis-publish
description: Publish a message to a Redis Pub/Sub channel.
---

# `redis-publish` — Redis Publish

**Mode:** `[*]` (client and server)

Publishes a payload to a Redis Pub/Sub channel. All Redis clients subscribed to that channel receive the message.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `channel` | string | Yes | — | Redis Pub/Sub channel name. Template expression. |
| `message` | string | No | `.Message` | Payload to publish. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Example

```yaml
handlers:
  - name: fan-out-redis
    match:
      jq: '.type == "broadcast"'
    builtin: redis-publish
    channel: 'events:{{.Message | jq ".room"}}'
    message: '{{.Message | jq ".data" | toJSON}}'
    respond: '{"published": true}'
```

---

## Edge Cases

- Publishing to a channel with no subscribers is a no-op in Redis — the message is discarded.
- This builtin publishes to Redis Pub/Sub, not to xwebs internal topics. Use `publish` for in-process topic delivery.
- Redis connection errors cause the handler to fail without sending `respond:`.
- Large messages (>64 MB) may be rejected by Redis depending on server configuration.
