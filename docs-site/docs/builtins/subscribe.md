---
title: subscribe
description: Subscribe the current client to an internal pub/sub topic so it receives future published messages.
---

# `subscribe` — Subscribe to Topic

**Mode:** `[S]` (server only)

Registers the current client connection as a subscriber for a named topic. Future messages published to that topic via the `publish` builtin or the `:publish` REPL command will be delivered to all subscribers. The `topic:` field is a template expression, allowing clients to self-select their topics via message content.

This builtin enables server-side fan-out patterns without any external broker — topics are maintained in memory by the xwebs server process.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `topic` | string | Yes | — | Topic name to subscribe the client to. Rendered as a Go template. |
| `respond` | string | No | — | Optional confirmation message sent back to the subscribing client after subscription is recorded. |

---

## Template Variables Injected

None. Standard connection context (`{{.ConnectionID}}`, `{{.Message}}`, etc.) is available in `topic:`.

---

## Examples

**Subscribe based on a JSON field:**

```yaml
handlers:
  - name: handle-subscribe
    match:
      jq: '.type == "subscribe"'
    builtin: subscribe
    topic: '{{.Message | jq ".channel"}}'
    respond: '{"subscribed": "{{.Message | jq ".channel"}}"}'
```

**Subscribe with prefix-stripping:**

```yaml
handlers:
  - name: prefix-sub
    match: "sub:*"
    builtin: subscribe
    topic: '{{.Message | trimPrefix "sub:"}}'
    respond: 'ok:{{.Message | trimPrefix "sub:"}}'
```

**Subscribe to a static admin broadcast channel:**

```yaml
handlers:
  - name: admin-feed
    match:
      jq: '.role == "admin"'
    builtin: subscribe
    topic: "admin-announcements"
    respond: '{"subscribed_to": "admin-announcements"}'
```

---

## Edge Cases

- A client can subscribe to multiple topics by matching multiple `subscribe` handlers or by sending multiple subscription messages.
- Subscribing to a topic the client is already subscribed to is idempotent — no duplicate deliveries occur.
- Topic names are case-sensitive plain strings; no wildcards or pattern matching is applied at subscription time.
- When a client disconnects, all its topic subscriptions are automatically removed.
