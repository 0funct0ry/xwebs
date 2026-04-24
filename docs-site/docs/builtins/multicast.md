---
title: multicast
description: Send a message to a specific named list of client IDs rather than all connected clients.
---

# `multicast` — Multicast to Client List

**Mode:** `[S]` (server only)

Delivers a message to a fixed or dynamically computed list of client connection IDs. Unlike `broadcast`, which reaches every connected client, `multicast` targets only the clients you name in `targets:`. Each entry in the list is rendered as a Go template, allowing IDs to be read from KV, derived from the message, or composed from template expressions.

Clients not in the list and clients that have disconnected are silently skipped.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `targets` | list of strings | Yes | — | List of client connection IDs to deliver to. Each item is rendered as a Go template. |
| `message` | string | No | `.Message` | Payload template. Defaults to the raw incoming message. |

---

## Template Variables Injected

None. Standard context is available in `targets` items and `message:`.

---

## Examples

**Send to two hardcoded client IDs (useful in tests):**

```yaml
handlers:
  - name: notify-pair
    match:
      jq: '.type == "pair_notify"'
    builtin: multicast
    targets:
      - "c-a1b2c3"
      - "c-d4e5f6"
    message: '{"type": "notification", "data": {{.Message | jq ".data" | toJSON}}}'
```

**Multicast to IDs stored in KV:**

```yaml
handlers:
  - name: team-notify
    match:
      jq: '.type == "team_message"'
    builtin: multicast
    targets:
      - '{{kv "team.member.1"}}'
      - '{{kv "team.member.2"}}'
      - '{{kv "team.member.3"}}'
    message: |
      {
        "type": "team_message",
        "text": "{{.Message | jq ".text"}}",
        "from": "{{.ConnectionID}}"
      }
```

**Single targeted message from message content:**

```yaml
handlers:
  - name: direct-message
    match:
      jq: '.type == "dm"'
    builtin: multicast
    targets:
      - '{{.Message | jq ".to"}}'
    message: |
      {
        "type": "dm",
        "from": "{{.ConnectionID}}",
        "text": "{{.Message | jq ".text"}}"
      }
    respond: '{"sent": true}'
```

---

## Edge Cases

- If a client ID in `targets` is not currently connected, it is silently skipped — no error is raised.
- Duplicate IDs in `targets` cause the message to be delivered multiple times to that client; deduplicate in the template if needed.
- Template errors in a single `targets` item cause that item to be skipped; other items in the list are still delivered.
- The sender itself is not excluded automatically; include or omit `{{.ConnectionID}}` from `targets` explicitly.
