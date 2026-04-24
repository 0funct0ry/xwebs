---
title: redis-get
description: Retrieve a value from Redis by key; result available as {{.RedisValue}}.
---

# `redis-get` — Redis Get

**Mode:** `[*]` (client and server)

Reads a Redis key and injects the value into `{{.RedisValue}}` for use in `respond:` or subsequent pipeline steps.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Redis key to read. Template expression. |
| `url` | string | No | `redis://localhost:6379` | Redis connection URL. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.RedisValue}}` | string | Value stored at the key, or empty string if the key does not exist. |

---

## Example

```yaml
handlers:
  - name: lookup-session
    match:
      jq: '.type == "get_session"'
    builtin: redis-get
    key: 'session:{{.Message | jq ".token"}}'
    respond: '{"session": {{.RedisValue | fromJSON | toJSON}}, "found": {{ne .RedisValue ""}}}'
```

---

## Edge Cases

- A missing key returns an empty string in `{{.RedisValue}}`; it does not error.
- Check `{{ne .RedisValue ""}}` in `respond:` to distinguish found vs. missing keys.
- Very large values may increase memory pressure on the xwebs process; prefer storing references rather than large blobs.
- Redis connection errors cause the handler to fail; `{{.RedisValue}}` is not populated.
