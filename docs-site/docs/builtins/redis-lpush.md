---
title: redis-lpush
description: Push a value onto the left end of a Redis list.
---

# `redis-lpush` — Redis List Push

**Mode:** `[*]` (client and server)

Prepends a value to a Redis list. Creates the list if it does not exist. Useful for work queues, event logs, and LIFO stacks.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis list key. Template expression. |
| `value` | string | No | `.Message` | Value to push. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Example

```yaml
handlers:
  - name: enqueue-job
    match:
      jq: '.type == "job"'
    builtin: redis-lpush
    key: "jobs:queue"
    value: '{{.Message | toJSON}}'
    respond: '{"enqueued": true}'
```

---

## Edge Cases

- `LPUSH` pushes to the head (left) of the list; use `redis-rpop` on the right to implement a FIFO queue.
- The list grows unboundedly unless trimmed externally with `LTRIM` or a separate `run:` step.
- Template errors in `key:` or `value:` abort the push.
- Redis connection errors cause the handler to fail.
