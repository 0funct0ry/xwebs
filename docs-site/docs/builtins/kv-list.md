---
title: kv-list
description: List all keys in the server KV store as a JSON array.
---

# `kv-list` — List KV Keys

**Mode:** `[S]` (server only)

Lists all keys in the server KV store and injects them as a JSON array into `{{.KvKeys}}`.

---

## Config Fields

No configuration fields. Uses `respond:` for the response template.

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.KvKeys}}` | string | JSON array of all current keys, e.g. `["key1","key2"]` |

---

## Examples

```yaml
handlers:
  - name: list-keys
    match:
      jq: '.type == "kv_list"'
    builtin: kv-list
    respond: '{"keys":{{.KvKeys}}}'
```

**Admin panel — show all stored state:**

```yaml
handlers:
  - name: admin-state
    match:
      jq: '.type == "admin" and .action == "list_state"'
    builtin: kv-list
    respond: '{"type":"state","keys":{{.KvKeys}},"count":{{.KvKeys | fromJSON | len}}}'
```

---

## REPL Example

```
xwebs> :kv list
  maintenance_mode  true
  session:ns-abc    token123
  counter           42
```
