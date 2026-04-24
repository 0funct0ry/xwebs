---
title: kv-get
description: Retrieve a value from the server KV store and inject it into the respond template.
---

# `kv-get` — Get KV Entry

**Mode:** `[S]` (server only)

Retrieves a value from the server KV store and makes it available as `{{.KvValue}}` in the `respond:` template.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | Key to retrieve. Supports template expressions. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.KvValue}}` | string | The stored value, or empty string if key doesn't exist |

---

## Examples

**Return a stored configuration value:**

```yaml
handlers:
  - name: get-config
    match:
      jq: '.type == "get_config"'
    builtin: kv-get
    key: '{{.Message | jq ".key"}}'
    respond: '{"key":"{{.Message | jq ".key"}}","value":"{{.KvValue}}"}'
```

**Check maintenance mode:**

```yaml
handlers:
  - name: check-maintenance
    match:
      jq: '.type == "status"'
    builtin: kv-get
    key: "maintenance_mode"
    respond: |
      {
        "maintenance": {{if eq .KvValue "true"}}true{{else}}false{{end}},
        "status": "{{if eq .KvValue "true"}}unavailable{{else}}ok{{end}}"
      }
```

**Retrieve a session token:**

```yaml
handlers:
  - name: verify-session
    match:
      jq: '.type == "verify"'
    builtin: kv-get
    key: 'session:{{.ConnectionID}}'
    respond: |
      {
        "valid": {{if .KvValue}}true{{else}}false{{end}},
        "session": "{{.KvValue}}"
      }
```

---

## Edge Cases

- If the key does not exist or has expired, `{{.KvValue}}` is an empty string. Check with `{{if .KvValue}}`.
- For reading KV values inside other templates without a full handler, use `{{kv "key"}}` directly.
