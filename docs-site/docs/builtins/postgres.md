---
title: postgres
description: Execute SQL against a PostgreSQL database; result rows in {{.Rows}}.
---

# `postgres` — PostgreSQL Query

**Mode:** `[*]` (client and server)

Executes a SQL statement against a PostgreSQL database. Results are injected as `{{.Rows}}` — a JSON array of row objects.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `dsn` | string | Yes | — | PostgreSQL connection string (e.g. `postgres://user:pass@localhost/dbname`). Template expression; use `{{env "DATABASE_URL"}}`. |
| `query` | string | Yes | — | SQL statement. Template expression. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Rows}}` | string | JSON array of row objects. Empty array for non-SELECT statements. |

---

## Example

```yaml
handlers:
  - name: pg-query
    match:
      jq: '.type == "search"'
    builtin: postgres
    dsn: '{{env "DATABASE_URL"}}'
    query: 'SELECT id, name FROM products WHERE name ILIKE "%{{.Message | jq ".q"}}%"'
    respond: '{"results": {{.Rows}}}'
```

---

## Edge Cases

- User-supplied data interpolated directly into `query:` is a SQL injection risk; use the `run:` builtin with `psql` and parameterised queries for untrusted input.
- Connection pooling is per-handler; high concurrency may exhaust PostgreSQL's connection limit — consider `PgBouncer` in front.
- Non-SELECT statements (INSERT, UPDATE, DELETE) return an empty `{{.Rows}}`; check exit codes via a `run:` step if you need affected row counts.
- DSN credentials should always come from environment variables, never hardcoded in config files.
