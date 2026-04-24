---
title: redis-incr
description: Atomically increment a Redis counter; new value available as {{.RedisValue}}.
---

# `redis-incr` — Redis Increment

**Mode:** `[*]` (client and server)

Atomically increments a Redis integer key by 1. Creates the key at 0 and increments to 1 if it does not exist. The new value is available as `{{.RedisValue}}`.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis key to increment. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.RedisValue}}` | string | The new integer value after increment, as a string. |

---

## Example

```yaml
handlers:
  - name: count-events
    match:
      jq: '.type == "event"'
    builtin: redis-incr
    key: 'counters:{{.Message | jq ".category"}}'
    respond: '{"count": {{.RedisValue}}}'
```

---

## Edge Cases

- The increment is atomic — safe for concurrent handlers writing to the same key.
- `{{.RedisValue}}` is a string; use `{{.RedisValue | toInt}}` for arithmetic in templates.
- If the key holds a non-integer value, Redis returns an error and the handler fails.
- There is no `redis-decr` builtin; use a `run:` step with `redis-cli DECR` if needed.
