---
title: append-file
description: Append a message to a file with configurable separator and automatic size-based rotation.
---

# `append-file` — Append to File

**Mode:** `[*]` (client and server)

Appends content to a file. Supports a configurable record separator and automatic rotation when the file exceeds `max_size:`. Simpler than `file-write` for pure append workloads, and adds rotation support.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | — | File path to append to. Template expression. |
| `content` | string | No | `.Message` | Content to append. Template expression. |
| `separator` | string | No | `\n` | String appended after each write (e.g. `\n`, `---\n`). |
| `max_size` | string | No | — | Maximum file size before rotation (e.g. `100MB`). Rotated file is renamed with a timestamp suffix. |

---

## Example

```yaml
handlers:
  - name: event-log
    match: "*"
    builtin: append-file
    path: "logs/events.jsonl"
    content: '{{.Message}}'
    separator: "\n"
    max_size: "50MB"
```

---

## Edge Cases

- Rotation renames the current file to `<path>.<timestamp>` and starts a new file at the original path; external log shippers should handle renamed files.
- `separator:` is appended after each record, not between records — the last record in a file also ends with the separator.
- Concurrent appends from multiple goroutines are safe on most OS filesystems (appends are atomic for small writes), but very large content blocks may interleave.
- Parent directories must exist; `append-file` does not create them.
