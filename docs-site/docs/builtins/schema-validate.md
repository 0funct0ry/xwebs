---
title: schema-validate
description: Validate the incoming message against a JSON Schema; validation errors in {{.ValidationErrors}}.
---

# `schema-validate` — JSON Schema Validation

**Mode:** `[*]` (client and server)

Validates the incoming message against a JSON Schema file. If validation passes, the handler proceeds normally. If it fails, `{{.ValidationErrors}}` contains a JSON array of error objects and the handler can respond with an error message without invoking downstream steps.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `schema` | string | Yes | — | Path to a JSON Schema file (draft-07 or later). Template expression. |
| `on_invalid` | string | No | — | Template rendered and sent as the response when validation fails. If omitted, the handler errors silently. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.ValidationErrors}}` | string | JSON array of validation error objects. Empty string if validation passed. |

---

## Example

**Validate before processing and return structured errors:**

```yaml
handlers:
  - name: validated-action
    match:
      jq: '.type == "action"'
    builtin: schema-validate
    schema: schemas/action.json
    on_invalid: |
      {
        "error": "validation_failed",
        "details": {{.ValidationErrors}}
      }
    respond: '{"processed": true}'
```

**Validate in pipeline, abort on failure:**

```yaml
handlers:
  - name: safe-ingest
    match:
      jq: '.type == "ingest"'
    pipeline:
      - builtin: schema-validate
        schema: schemas/ingest.json
        on_invalid: '{"error":"invalid","details":{{.ValidationErrors}}}'
      - run: ingest.sh "{{.Message | shellEscape}}"
    respond: '{"ingested": true}'
```

---

## Edge Cases

- If the schema file does not exist or contains invalid JSON Schema, the handler errors.
- `on_invalid:` short-circuits the pipeline — subsequent steps do not run on validation failure.
- `{{.ValidationErrors}}` is only populated on failure; on success it is an empty string.
- JSON Schema drafts 4, 6, 7, and 2019-09 are supported; check your schema's `$schema` declaration for compatibility.
