---
title: redis-set
description: Set a key in Redis with an optional TTL.
---

# `redis-set` — Redis Set

**Mode:** `[*]` (client and server)

Writes a key-value pair to Redis. The `key:` and `value:` fields are Go template expressions. An optional `ttl:` sets the key expiry.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis key. Template expression. |
| `value` | string | Yes | — | Value to store. Template expression. |
| `ttl` | string | No | — | Key expiry as a Go duration (`60s`, `5m`). |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Example

```yaml
handlers:
  - name: cache-result
    match:
      jq: '.type == "cache"'
    builtin: redis-set
    key: 'cache:{{.Message | jq ".id"}}'
    value: '{{.Message | jq ".data" | toJSON}}'
    ttl: "5m"
    respond: '{"cached": true}'
```

---

## Edge Cases

- If Redis is unreachable, the handler errors and `respond:` is not sent.
- `ttl:` values shorter than 1 second are rounded to 1 second by the Redis server.
- Template errors in `key:` or `value:` abort the write; no entry is created.
- Keys are always stored as Redis strings; for complex types, serialize with `toJSON`.
