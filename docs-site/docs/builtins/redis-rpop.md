---
title: redis-rpop
description: Pop a value from the right end of a Redis list; result available as {{.RedisValue}}.
---

# `redis-rpop` — Redis List Pop

**Mode:** `[*]` (client and server)

Atomically removes and returns the rightmost element from a Redis list. The result is available as `{{.RedisValue}}`. Returns an empty string if the list is empty.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis list key. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.RedisValue}}` | string | The popped value, or empty string if the list was empty. |

---

## Example

```yaml
handlers:
  - name: dequeue-job
    match:
      jq: '.type == "dequeue"'
    builtin: redis-rpop
    key: "jobs:queue"
    respond: '{"job": {{if .RedisValue}}{{.RedisValue}}{{else}}null{{end}}}'
```

---

## Edge Cases

- Returns empty string (not an error) when the list is empty; check `{{ne .RedisValue ""}}` in `respond:`.
- Combines with `redis-lpush` to form a FIFO queue (push left, pop right).
- The pop is atomic — safe for concurrent handlers dequeuing from the same list.
- Redis connection errors cause the handler to fail.
