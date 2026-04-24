---
title: json-merge
description: Deep-merge a JSON object into the incoming message and re-inject the merged result into the pipeline.
---

# `json-merge` — JSON Merge

**Mode:** `[*]` (client and server)

Performs a deep JSON merge of a `with:` template into the incoming JSON message. The merged result replaces `{{.Message}}` for subsequent pipeline steps and `respond:`. Keys in `with:` override keys in the original message; nested objects are merged recursively.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `with` | string | Yes | — | JSON object to merge into the incoming message. Template expression. |

---

## Example

**Inject server metadata into every outgoing response:**

```yaml
handlers:
  - name: enrich
    match:
      jq: '.type == "response"'
    builtin: json-merge
    with: |
      {
        "server": "{{hostname}}",
        "ts": "{{now | formatTime "RFC3339"}}",
        "version": "{{env "APP_VERSION"}}"
      }
    respond: '{{.Message}}'
```

**Override specific fields before forwarding:**

```yaml
handlers:
  - name: sanitize-forward
    match: "*"
    builtin: json-merge
    with: '{"source": "xwebs", "forwarded": true}'
    respond: '{{.Message}}'
```

---

## Edge Cases

- Both the incoming message and `with:` must be valid JSON objects. If either is not an object (e.g., a JSON array or plain string), the handler errors.
- Arrays within the merged object are replaced entirely, not appended.
- `with:` is rendered as a template before merging, so template errors abort the merge.
- After `json-merge`, `{{.Message}}` contains the merged JSON; the original message is no longer directly accessible.
