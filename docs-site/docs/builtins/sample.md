---
title: sample
description: Pass only one in every N messages, silently dropping the rest.
---

# `sample` — Message Sampling

**Mode:** `[*]` (client and server)

Lets exactly 1 out of every N messages through to the `respond:` template or subsequent pipeline steps. All other messages are silently discarded. The counter is global to the handler instance (not per-client) and resets on server restart.

Use `sample` to reduce noise from high-frequency sources, implement statistical sampling for monitoring, or throttle expensive downstream calls without a strict rate limit.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rate` | integer | Yes | — | Sampling denominator N. A value of `5` means 1-in-5 messages pass through. Must be ≥ 1. |

---

## Template Variables Injected

None. Standard context is available in `respond:` for messages that pass.

---

## Examples

**Sample 1 in 10 sensor messages for processing:**

```yaml
handlers:
  - name: sensor-sample
    match:
      jq: '.type == "sensor"'
    builtin: sample
    rate: 10
    respond: '{"sampled": true, "data": {{.Message | toJSON}}}'
```

**Log every 100th message for audit:**

```yaml
handlers:
  - name: audit-sample
    match: "*"
    builtin: sample
    rate: 100
    respond: |
      {
        "type": "audit",
        "message": {{.Message | toJSON}},
        "index": {{.MessageIndex}},
        "ts": "{{now | formatTime "RFC3339"}}"
      }
```

**Use in a pipeline to selectively forward to an expensive handler:**

```yaml
handlers:
  - name: selective-forward
    match:
      jq: '.type == "metrics"'
    pipeline:
      - builtin: sample
        rate: 5
      - run: send_to_analytics.sh "{{.Message | shellEscape}}"
    respond: '{"forwarded": true}'
```

---

## Edge Cases

- `rate: 1` passes every message (no sampling). `rate: 0` is invalid and treated as `rate: 1` with a warning.
- The counter is shared across all clients on a handler instance. In a high-concurrency server, the effective sample rate is approximate rather than exact.
- Dropped messages produce no response and no log entry by default. Add a `log` step before `sample` if you need to count total arrivals.
- The pass/drop decision is deterministic (counter-based), not random. The first message always passes.
