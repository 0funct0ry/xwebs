---
title: echo
description: Echo the received message back to the sender, with optional delay and respond override.
---

# `echo` — Echo Message

**Mode:** `[*]` (client and server)

Echoes the received message back to the sender. The simplest possible handler — no shell, no config. When `respond:` is set, only the `respond:` template is sent; the verbatim message is suppressed. When `delay:` is set, the response is deferred by that duration without blocking other handlers.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `delay` | string | No | — | Defer the echo response by this duration (Go duration: `500ms`, `1s`). Supports template expressions. |

The `respond:` field works with `echo` to override the echoed content — the resulting message uses the full template context including `.Message`, `.ConnectionID`, `.Vars`, etc.

---

## Template Variables Injected

`echo` does not inject additional template variables. The standard message and connection context is available in `respond:`.

---

## Basic Examples

**Minimal echo server:**

```yaml
handlers:
  - name: echo-all
    match: "*"
    builtin: echo
```

**Echo with a custom response template:**

```yaml
handlers:
  - name: echo-with-metadata
    match: "*"
    builtin: echo
    respond: |
      {
        "echo": {{.Message | toJSON}},
        "connection_id": "{{.ConnectionID}}",
        "received_at": "{{now | formatTime "RFC3339"}}"
      }
```

**Delayed echo:**

```yaml
handlers:
  - name: delayed-echo
    match:
      jq: '.type == "delayed"'
    builtin: echo
    delay: "{{.Message | jq ".delay_ms"}}ms"
    respond: '{"delayed":true,"echo":{{.Message | toJSON}}}'
```

**Fixed delay echo:**

```yaml
handlers:
  - name: slow-echo
    match: "*"
    builtin: echo
    delay: "2s"
```

---

## REPL Example

Add an echo handler interactively from the server REPL:

```
xwebs> :handler add -m "*" --builtin echo
✓ added handler echo-1
```

With a custom response:

```
xwebs> :handler add -m ".type == \"ping\"" --builtin echo -R '{"type":"pong","ts":"{{now}}"}'
✓ added handler echo-2
```

---

## Edge Cases

- **`respond:` suppresses the verbatim echo.** If you set `respond:` on an echo handler and your template produces an empty string, no message is sent. This is intentional — it lets you conditionally suppress responses with `{{if ...}}`.
- **`delay:` is non-blocking.** Other handlers on the same message are not delayed. Only the echo response for this specific handler is deferred.
- **`delay:` with template expressions.** The delay is computed from the message content at dispatch time. If the template fails to evaluate (e.g., `.delay_ms` is missing), the delay defaults to 0.
- **Client mode.** In client mode, `echo` sends the message back to the server — not to a local listener. This is useful for relay testing.
