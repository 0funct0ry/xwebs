---
title: sequence
description: Cycle through an ordered list of responses, optionally per-client and with wraparound.
---

# `sequence` — Response Sequence

**Mode:** `[*]` (client and server)

Returns the next item from a predefined list of responses each time the handler matches. Useful for mocking multi-step protocols, simulating paginated data, or walking through a scripted conversation. The internal cursor advances by one on every match; when the end is reached, behaviour depends on the `loop:` flag.

In server mode, the cursor can be scoped per-connection (`per_client: true`) so that each client independently walks through the sequence.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `items` | list of strings | Yes | — | Ordered list of responses. Each item is rendered as a Go template before sending. |
| `loop` | bool | No | `false` | When `true`, wraps back to the first item after the last one. When `false`, the last item is repeated indefinitely after the sequence is exhausted. |
| `per_client` | bool | No | `false` | Maintain a separate cursor for each connected client. Only meaningful in server mode. |

---

## Template Variables Injected

`sequence` does not inject additional template variables. The full message and connection context is available in each `items` template.

---

## Examples

**Mock a three-step handshake:**

```yaml
handlers:
  - name: handshake-mock
    match:
      jq: '.type == "next"'
    builtin: sequence
    items:
      - '{"step": 1, "status": "started"}'
      - '{"step": 2, "status": "processing"}'
      - '{"step": 3, "status": "complete"}'
```

**Cycling countdown with loop, per-client tracking:**

```yaml
handlers:
  - name: countdown
    match: "*"
    builtin: sequence
    loop: true
    per_client: true
    items:
      - '{"count": 3}'
      - '{"count": 2}'
      - '{"count": 1}'
      - '{"count": 0, "done": true}'
```

**Dynamic items using template context:**

```yaml
handlers:
  - name: page-results
    match:
      jq: '.type == "fetch"'
    builtin: sequence
    loop: false
    items:
      - '{"page": 1, "requestedBy": "{{.ConnectionID}}"}'
      - '{"page": 2, "requestedBy": "{{.ConnectionID}}"}'
      - '{"page": 3, "done": true}'
```

---

## Edge Cases

- When `loop: false` and the sequence is exhausted, the last item is resent for every subsequent match. Use `drop` in a follow-up handler to stop responses entirely.
- With `per_client: true`, cursor state is held in memory and lost if the server restarts. Do not use for durable sequences.
- Each item in `items` is rendered as a full Go template at send time — template errors are logged and that item is skipped; the cursor still advances.
- In client mode, `per_client:` has no effect since there is only one connection.
