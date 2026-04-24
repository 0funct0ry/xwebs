---
title: drop
description: Silently discard the message with no response and no further handler processing.
---

# `drop` — Drop Message

**Mode:** `[*]` (client and server)

Consumes the message and does nothing. No response is sent to the client or server, and no subsequent handlers in the chain are evaluated. Think of it as `/dev/null` for WebSocket messages.

`drop` is the recommended way to explicitly suppress a message rather than relying on unmatched handlers to stay silent. It is also useful as a catch-all at the end of a handler list to suppress noisy messages without logging them.

---

## Config Fields

`drop` has no configuration fields.

---

## Template Variables Injected

None.

---

## Examples

**Silently discard heartbeat messages to reduce noise:**

```yaml
handlers:
  - name: ignore-heartbeats
    match:
      jq: '.type == "heartbeat"'
    builtin: drop
```

**Drop binary frames, respond to text only:**

```yaml
handlers:
  - name: reject-binary
    match:
      binary: true
    builtin: drop

  - name: handle-text
    match: "*"
    builtin: echo
```

**Rate-limit filter — drop overflow messages:**

```yaml
handlers:
  - name: rate-guard
    match: "*"
    builtin: rate-limit
    rate: "5/s"
    on_exceeded: drop        # drop instead of responding with an error
```

---

## Edge Cases

- `drop` halts the handler chain — any handlers with lower priority that would otherwise match are not evaluated.
- No close frame, no error, and no log entry is produced by default. To log drops, place a `log` builtin in a pipeline step before `drop`.
- In client mode, dropping a message means nothing is forwarded to the remote server. In server mode, it means nothing is sent back to the connecting client.
- Combining `drop` with `exclusive: true` on the handler is redundant — `drop` already prevents further handler evaluation.
