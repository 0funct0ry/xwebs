---
title: unsubscribe
description: Remove the current client from an internal pub/sub topic so it stops receiving published messages.
---

# `unsubscribe` — Unsubscribe from Topic

**Mode:** `[S]` (server only)

Removes the current client from one or all of its topic subscriptions. The `topic:` field is a template expression. When `all: true` is set, the client is removed from every topic it is subscribed to regardless of the `topic:` value.

This is the counterpart to the `subscribe` builtin and is used to implement graceful unsubscription flows driven by client messages.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `topic` | string | Yes (unless `all: true`) | — | Topic to remove the client from. Rendered as a Go template. |
| `all` | bool | No | `false` | When `true`, removes the client from every topic it is currently subscribed to. `topic:` is ignored. |

---

## Template Variables Injected

None. Standard connection context is available in `topic:`.

---

## Examples

**Unsubscribe from a specific channel named in the message:**

```yaml
handlers:
  - name: handle-unsubscribe
    match:
      jq: '.type == "unsubscribe"'
    builtin: unsubscribe
    topic: '{{.Message | jq ".channel"}}'
    respond: '{"unsubscribed": "{{.Message | jq ".channel"}}"}'
```

**Unsubscribe from all topics on logout:**

```yaml
handlers:
  - name: logout
    match:
      jq: '.type == "logout"'
    builtin: unsubscribe
    all: true
    respond: '{"logged_out": true}'
```

**Prefix-based unsubscription:**

```yaml
handlers:
  - name: prefix-unsub
    match: "unsub:*"
    builtin: unsubscribe
    topic: '{{.Message | trimPrefix "unsub:"}}'
    respond: 'ok:unsubscribed:{{.Message | trimPrefix "unsub:"}}'
```

---

## Edge Cases

- Unsubscribing from a topic the client is not subscribed to is a no-op — no error is raised.
- With `all: true`, the `topic:` field is evaluated but its result is ignored. To avoid confusion, omit `topic:` when using `all: true`.
- Unsubscription takes effect immediately; messages published in the same moment may or may not arrive depending on goroutine scheduling.
- Client disconnection automatically removes all subscriptions — explicit `unsubscribe` is only needed for mid-session topic changes.
