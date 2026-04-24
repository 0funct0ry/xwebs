---
title: sqlite
description: Execute parameterised SQL against an embedded SQLite database; result rows in {{.Rows}}.
---

# `sqlite` — SQLite Query

**Mode:** `[*]` (client and server)

Executes a SQL statement against an embedded SQLite database file. Query results are injected as `{{.Rows}}` — a JSON array of row objects. Supports both read (SELECT) and write (INSERT, UPDATE, DELETE) operations.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `db` | string | Yes | — | Path to the SQLite database file. Created if it does not exist. |
| `query` | string | Yes | — | SQL statement. Template expression. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Rows}}` | string | JSON array of row objects from a SELECT. Empty array for non-SELECT statements. |

---

## Example

```yaml
handlers:
  - name: sqlite-query
    match:
      jq: '.type == "lookup"'
    builtin: sqlite
    db: "data/app.db"
    query: 'SELECT * FROM users WHERE id = "{{.Message | jq ".user_id" | shellEscape}}"'
    respond: '{"users": {{.Rows}}}'
```

---

## Edge Cases

- SQLite allows only one writer at a time; concurrent write handlers may contend. Use `concurrent: false` on write handlers to serialise.
- Template expressions in `query:` can introduce SQL injection if user-supplied data is interpolated without escaping. Prefer parameterised approaches via `run:` with the `sqlite3` CLI and proper quoting.
- SELECT results with no rows return an empty JSON array `[]`.
- For production workloads with concurrent writes, consider `postgres` instead.
