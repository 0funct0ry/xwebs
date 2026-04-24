---
title: kv-set
description: Set a key-value pair in the server-scoped KV store, with optional TTL.
---

# `kv-set` — Set KV Entry

**Mode:** `[S]` (server only)

Sets a key-value pair in the server-scoped in-memory KV store. Values are stored as strings. All handlers and the server REPL share the same KV store. Read values in templates with `{{kv "key"}}`.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Key name. Supports template expressions. |
| `value` | string | Yes | — | Value to store. Supports template expressions. |
| `ttl` | string | No | — | Expiry duration (e.g., `60s`, `5m`). Key is deleted after TTL. |

---

## Template Variables Injected

`kv-set` does not inject additional template variables. Use `respond:` with `{{kv "key"}}` to confirm the stored value.

---

## Examples

**Store a session token:**

```yaml
handlers:
  - name: store-token
    match:
      jq: '.type == "auth"'
    builtin: kv-set
    key: 'session:{{.ConnectionID}}'
    value: '{{.Message | jq ".token"}}'
    ttl: "1h"
    respond: '{"stored":true}'
```

**Count messages with a template-derived key:**

```yaml
handlers:
  - name: message-counter
    match: "*"
    builtin: kv-set
    key: 'count:{{.ConnectionID}}'
    value: '{{add (kv (print "count:" .ConnectionID) | atoi) 1}}'
```

**Store JSON as a string:**

```yaml
handlers:
  - name: cache-result
    match:
      jq: '.type == "compute"'
    run: ./compute.sh
    builtin: kv-set
    key: 'result:{{.Message | jq ".id"}}'
    value: '{{.Stdout | trim}}'
    ttl: "10m"
    respond: '{"cached":true,"key":"result:{{.Message | jq ".id"}}"}'
```

---

## REPL Example

```
xwebs> :kv set maintenance_mode true
xwebs> :kv set -t last_restart "{{now | formatTime \"RFC3339\"}}"
xwebs> :kv get maintenance_mode
true
```

---

## Edge Cases

- Values are always stored as strings. Numbers must be converted back with `atoi` or `toFloat` when reading.
- `ttl` uses Go duration syntax. An expired key returns an empty string from `{{kv "key"}}`.
- Setting a key that already exists overwrites the value (and resets the TTL if provided).
- The KV store is not persisted to disk — it resets when the server restarts. For persistence, use `redis-set` or `sqlite`.
