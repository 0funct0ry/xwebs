---
title: gate
description: Allow a message through only if a KV store key matches an expected value; otherwise respond with a closed-gate message.
---

# `gate` — KV-Gated Message Filter

**Mode:** `[S]` (server only)

Checks the server-scoped KV store: if the value at `key:` equals `expect:`, the message passes through to `respond:` or subsequent pipeline steps. If the value does not match (or the key does not exist), the gate is "closed" and the client receives `on_closed:` instead. No further handlers run when the gate is closed.

This is useful for feature flags, server-side circuit breakers, maintenance modes, or any scenario where message processing should be toggled without redeploying config.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `key` | string | Yes | — | KV store key to read. Rendered as a Go template. |
| `expect` | string | Yes | — | Expected value. The gate opens when the KV value equals this string exactly (case-sensitive). Rendered as a Go template. |
| `on_closed` | string | No | `""` | Template to render and send when the gate is closed (KV value does not match). If empty, no response is sent. |

---

## Template Variables Injected

None. Standard context is available in `key:`, `expect:`, and `on_closed:`.

---

## Examples

**Feature flag — only process messages when the feature is enabled:**

```yaml
handlers:
  - name: new-feature
    match:
      jq: '.type == "new_action"'
    builtin: gate
    key: "feature.new_action"
    expect: "enabled"
    on_closed: '{"error": "feature_disabled"}'
    respond: '{"processed": true}'
```

**Maintenance mode gate:**

```yaml
handlers:
  - name: maintenance-check
    match: "*"
    priority: 100
    builtin: gate
    key: "server.maintenance"
    expect: "false"
    on_closed: '{"error": "server_in_maintenance", "retry_after": "300s"}'
```

**Dynamic key based on client context:**

```yaml
handlers:
  - name: per-client-gate
    match:
      jq: '.type == "premium_action"'
    builtin: gate
    key: 'client.{{.ConnectionID}}.tier'
    expect: "premium"
    on_closed: '{"error": "upgrade_required"}'
    respond: '{"action": "executed"}'
```

---

## Edge Cases

- If the KV key does not exist, the gate is treated as closed regardless of `expect:`.
- Comparison is exact string equality; `"true"` does not equal `"True"` or `"1"`.
- Use `:kv set feature.new_action enabled` from the server REPL or a `kv-set` handler to open or close the gate at runtime without restarting.
- When the gate is closed, no further handlers run for that message — `gate` acts as an implicit `exclusive: true` on a closed match.
