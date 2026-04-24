---
title: nats-publish
description: Publish a message to a NATS subject.
---

# `nats-publish` — NATS Publish

**Mode:** `[*]` (client and server)

Publishes a payload to a NATS subject. Enables bridging WebSocket traffic to NATS-based microservice architectures.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | No | `nats://localhost:4222` | NATS server URL. |
| `subject` | string | Yes | — | NATS subject. Template expression. |
| `message` | string | No | `.Message` | Payload. Template expression. |

---

## Example

```yaml
handlers:
  - name: ws-to-nats
    match:
      jq: '.type == "event"'
    builtin: nats-publish
    url: "nats://localhost:4222"
    subject: 'events.{{.Message | jq ".category"}}'
    message: '{{.Message | toJSON}}'
    respond: '{"published": true}'
```

---

## Edge Cases

- NATS is a fire-and-forget publish; there is no delivery confirmation.
- Subject wildcards are for subscriptions only; publish subjects must be fully qualified.
- Connection errors cause the handler to fail; NATS reconnect logic is handled by the NATS client library automatically.
- For NATS JetStream persistence, use a `run:` step with the `nats` CLI tool instead.
