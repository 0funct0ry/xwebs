---
title: ab-test
description: Route messages to one of two handlers based on a deterministic hash of a message field.
---

# `ab-test` — A/B Test Router

**Mode:** `[S]` (server only)

Splits incoming messages between two named handlers — `handler_a:` and `handler_b:` — using a deterministic, stable hash of a `field:` extracted from the message. The same field value always routes to the same bucket (A or B), making the split reproducible across restarts. The default split is 50/50 but can be adjusted with `ratio:`.

Use `ab-test` to gradually roll out new handler logic, run experiments, or implement canary deployments at the message-routing level.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `field` | string | Yes | — | jq expression to extract the routing key from the message (e.g. `".user_id"`). The hash is computed on the extracted string value. |
| `handler_a` | string | Yes | — | Name of an existing handler to invoke for the A bucket. |
| `handler_b` | string | Yes | — | Name of an existing handler to invoke for the B bucket. |
| `ratio` | float | No | `0.5` | Fraction of traffic routed to `handler_a`. Must be between 0.0 and 1.0. The remainder goes to `handler_b`. |

---

## Template Variables Injected

None. Routing is transparent — the selected handler receives the full original context.

---

## Examples

**50/50 split on user_id:**

```yaml
handlers:
  - name: ab-router
    match:
      jq: '.type == "action"'
    builtin: ab-test
    field: ".user_id"
    handler_a: action-v1
    handler_b: action-v2

  - name: action-v1
    match: "__never__"   # only called by ab-test
    respond: '{"version": 1, "result": "old-logic"}'

  - name: action-v2
    match: "__never__"
    respond: '{"version": 2, "result": "new-logic"}'
```

**10% canary rollout to new handler:**

```yaml
handlers:
  - name: canary-router
    match:
      jq: '.type == "compute"'
    builtin: ab-test
    field: ".session_id"
    handler_a: compute-canary
    handler_b: compute-stable
    ratio: 0.1
```

**A/B test on message content field:**

```yaml
handlers:
  - name: offer-ab
    match:
      jq: '.type == "homepage"'
    builtin: ab-test
    field: ".device_id"
    handler_a: offer-discount
    handler_b: offer-premium
    ratio: 0.5
```

---

## Edge Cases

- The hash function is stable across restarts — the same `field` value always maps to the same bucket. Changing `ratio:` will reroute some users.
- If the `field:` jq expression returns null or errors, the message defaults to `handler_b`.
- The referenced `handler_a` and `handler_b` handlers must exist in the same config. They are invoked directly and their own `match:` expressions are bypassed.
- `ab-test` does not record which bucket a client was assigned to; if you need persistence, store the assignment in KV using `kv-set`.
