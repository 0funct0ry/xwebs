---
title: jq-transform
description: Apply a jq expression to the incoming message and re-inject the result into the handler pipeline.
---

# `jq-transform` — JQ Transform

**Mode:** `[*]` (client and server)

Runs a jq `expression:` against the incoming message and replaces `{{.Message}}` with the result for subsequent pipeline steps and `respond:`. This lets you reshape, filter, or extract from JSON without spawning a shell process.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `expression` | string | Yes | — | jq expression applied to the incoming message (e.g. `".data"`, `"{id:.id, name:.name}"`). |

---

## Example

```yaml
handlers:
  - name: extract-data
    match:
      jq: '.type == "payload"'
    pipeline:
      - builtin: jq-transform
        expression: ".data"
      - run: process.sh
    respond: '{{.Stdout | toJSON}}'
```

**Strip sensitive fields before forwarding:**

```yaml
handlers:
  - name: sanitize
    match: "*"
    builtin: jq-transform
    expression: 'del(.password, .token, .secret)'
    respond: '{{.Message}}'
```

---

## Edge Cases

- If the jq expression errors (invalid JSON input or jq syntax error), the handler fails and no response is sent.
- The transformed result replaces `{{.Message}}` in all downstream steps; the original message is no longer accessible.
- jq expressions that produce non-string output (arrays, objects) are serialised to compact JSON before re-injection.
- For simple field extraction in `respond:`, the `jq` template function is usually sufficient and simpler than a full `jq-transform` step.
