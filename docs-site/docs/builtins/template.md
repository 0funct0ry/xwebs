---
title: template
description: Render a Go template file and send the result as the handler response.
---

# `template` — Render Template File

**Mode:** `[*]` (client and server)

Loads a Go template from a file on disk, renders it against the full handler context, and sends the result as the response. This lets you keep complex response payloads in version-controlled template files rather than embedding them inline in `handlers.yaml`.

The template file has access to all standard xwebs template variables and functions — `.Message`, `.ConnectionID`, `{{now}}`, `{{jq}}`, KV helpers, and everything else available in `respond:` strings.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `file` | string | Yes | — | Path to the Go template file (relative to the working directory or absolute). |

---

## Template Variables Injected

`template` does not inject additional variables. The complete handler execution context is available inside the template file.

| Variable | Available | Notes |
|----------|-----------|-------|
| `{{.Message}}` | Yes | Raw incoming message |
| `{{.ConnectionID}}` | Yes | Unique connection identifier |
| `{{.Stdout}}` | Yes | Shell command output, if a `run:` step preceded this in a pipeline |
| `{{.ForwardReply}}` | Yes | Upstream reply, if combined with `forward` in a pipeline |

---

## Examples

**Render a static template file for every ping:**

```yaml
handlers:
  - name: pong-template
    match:
      jq: '.type == "ping"'
    builtin: template
    file: templates/pong.tmpl
```

`templates/pong.tmpl`:
```
{"type":"pong","server":"{{hostname}}","ts":"{{now | formatTime "RFC3339"}}","conn":"{{.ConnectionID}}"}
```

**Use after a shell command to format its output:**

```yaml
handlers:
  - name: query-and-render
    match:
      jq: '.type == "query"'
    pipeline:
      - run: psql -U app -d mydb -t -A -c "SELECT * FROM events LIMIT 10;"
        as: db
    builtin: template
    file: templates/events.tmpl
```

**Dynamic template path (one template per message type):**

```yaml
handlers:
  - name: typed-render
    match:
      jq: '.type != null'
    builtin: template
    file: 'templates/{{.Message | jq ".type"}}.tmpl'
```

---

## Edge Cases

- The `file:` field itself is rendered as a template, allowing dynamic file paths. If the rendered path does not exist, the handler errors and no response is sent.
- Template files are re-read from disk on every handler invocation — no caching. Hot-reloading works automatically; use `--handler-timeout` to bound slow I/O.
- If the template fails to render (syntax error, missing variable), the handler logs the error and sends no response to the client.
- Relative paths in `file:` are resolved from the process working directory at startup, not from the location of `handlers.yaml`.
