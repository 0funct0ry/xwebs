---
title: redis-del
description: Delete one or more keys from Redis.
---

# `redis-del` — Redis Delete

**Mode:** `[*]` (client and server)

Deletes the specified Redis key. Deleting a non-existent key is a no-op.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis key to delete. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Example

```yaml
handlers:
  - name: invalidate-cache
    match:
      jq: '.type == "invalidate"'
    builtin: redis-del
    key: 'cache:{{.Message | jq ".id"}}'
    respond: '{"invalidated": true}'
```

---

## Edge Cases

- Deleting a non-existent key is silent — no error is raised.
- `key:` is a single key; to delete multiple keys, chain multiple `redis-del` steps in a pipeline.
- Template errors in `key:` abort the deletion.
- Redis connection errors cause the handler to fail.
