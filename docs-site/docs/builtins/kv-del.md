---
title: kv-del
description: Delete a key from the server KV store.
---

# `kv-del` — Delete KV Entry

**Mode:** `[S]` (server only)

Deletes a key from the server KV store. No error is returned if the key doesn't exist.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Key to delete. Supports template expressions. |

---

## Examples

```yaml
handlers:
  - name: logout
    match:
      jq: '.type == "logout"'
    builtin: kv-del
    key: 'session:{{.ConnectionID}}'
    respond: '{"logged_out":true}'
```

**Delete on disconnect (lifecycle hook):**

```yaml
on_disconnect:
  - builtin: kv-del
    key: 'session:{{.ConnectionID}}'
```

---

## REPL Example

```
xwebs> :kv del maintenance_mode
xwebs> :kv list
(empty)
```
