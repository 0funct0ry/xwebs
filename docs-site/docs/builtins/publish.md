---
title: publish
description: Publish a message to all subscribers of an internal topic.
---

# `publish` — Publish to Topic

**Mode:** `[S]` (server only)

Delivers a message to every client currently subscribed to the named topic. Both `topic:` and `message:` are Go template expressions, so the destination and payload can be computed from the incoming message. If the topic has no subscribers, the publish is a no-op unless `allow_empty: true` is explicitly set (the default behaviour is to silently succeed).

Combine `publish` with `subscribe` and `unsubscribe` to build a fully in-process pub/sub system without an external broker.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `topic` | string | Yes | — | Name of the topic to publish to. Rendered as a Go template. |
| `message` | string | No | `.Message` | Payload to deliver to subscribers. Rendered as a Go template. Defaults to the raw incoming message. |
| `allow_empty` | bool | No | `false` | When `false` (default), publishing to a topic with zero subscribers is silently ignored. When `true`, the publish still succeeds but a debug log entry is written. |

---

## Template Variables Injected

None. Standard connection context is available in `topic:` and `message:`.

---

## Examples

**Fan-out an incoming message to a channel named in the payload:**

```yaml
handlers:
  - name: route-publish
    match:
      jq: '.type == "publish"'
    builtin: publish
    topic: '{{.Message | jq ".channel"}}'
    message: '{{.Message | jq ".data" | toJSON}}'
    respond: '{"published": true}'
```

**Broadcast a server event to all admin subscribers:**

```yaml
handlers:
  - name: admin-notify
    match:
      jq: '.type == "admin_event"'
    builtin: publish
    topic: "admin-announcements"
    message: |
      {
        "type": "admin_event",
        "payload": {{.Message | jq ".payload" | toJSON}},
        "from": "{{.ConnectionID}}",
        "ts": "{{now | formatTime "RFC3339"}}"
      }
```

**Allow publish to empty topic (for debugging):**

```yaml
handlers:
  - name: debug-publish
    match:
      jq: '.type == "debug_broadcast"'
    builtin: publish
    topic: '{{.Message | jq ".topic"}}'
    allow_empty: true
    respond: '{"sent": true}'
```

---

## Edge Cases

- `publish` does not send a message to the publishing client itself — use `broadcast` if you want the sender to receive the message too.
- If the `message:` template fails to render, nothing is published and the error is logged; the `respond:` template is still evaluated.
- Topic matching is exact string equality — `publish` to `"news"` will not reach subscribers of `"news/sports"`.
- Publishing to a topic with no subscribers when `allow_empty: false` (the default) neither errors nor logs anything. Enable `allow_empty: true` to surface these events in the debug log.
