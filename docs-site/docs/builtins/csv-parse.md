---
title: csv-parse
description: Parse a CSV message into a row array; rows available as {{.Rows}}.
---

# `csv-parse` — CSV Parse

**Mode:** `[*]` (client and server)

Parses the incoming message as CSV and injects the result as `{{.Rows}}` — a JSON array of row arrays (or objects when `header: true`). Useful for clients that send tabular data as CSV frames.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `header` | bool | No | `false` | When `true`, the first row is treated as a header and each subsequent row is a JSON object keyed by header names. |
| `delimiter` | string | No | `,` | Field delimiter character (e.g. `","`, `"\t"` for TSV). |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Rows}}` | string | JSON array of rows. With `header: false`, each row is an array of strings. With `header: true`, each row is an object. |

---

## Example

**Parse a TSV payload and store rows:**

```yaml
handlers:
  - name: tsv-ingest
    match:
      jq: '.type == "tsv_upload"'
    builtin: csv-parse
    delimiter: "\t"
    header: true
    respond: '{"rows": {{.Rows}}, "count": {{.Rows | fromJSON | length}}}'
```

**Parse a raw CSV frame:**

```yaml
handlers:
  - name: csv-handler
    match:
      regex: "^[a-zA-Z0-9].*,.*"
    builtin: csv-parse
    header: false
    respond: '{"parsed": {{.Rows}}}'
```

---

## Edge Cases

- Malformed CSV (unmatched quotes, inconsistent column counts) may cause a parse error or silently produce incomplete rows depending on the parser's strictness.
- With `header: true`, rows that have fewer columns than the header produce objects with missing keys; extra columns are ignored.
- `{{.Rows}}` is a JSON string; pipe through `fromJSON` for further template manipulation (e.g. `{{.Rows | fromJSON | length}}`).
- Very large CSV documents are fully buffered in memory before processing; avoid sending files larger than available RAM.
