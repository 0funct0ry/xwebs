---
title: kafka-consume
description: Consume messages from a Kafka topic and inject them into the handler pipeline.
---

# `kafka-consume` — Kafka Consumer

**Mode:** `[S]` (server only)

Runs a persistent Kafka consumer. Each consumed record is injected into the handler pipeline, enabling Kafka-to-WebSocket bridging.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `brokers` | list of strings | Yes | — | Kafka broker addresses. |
| `topic` | string | Yes | — | Kafka topic to consume from. |
| `group` | string | Yes | — | Consumer group ID. Used for offset tracking and load balancing. |
| `offset` | string | No | `latest` | Start offset: `latest` (skip historical) or `earliest` (replay from beginning). |

---

## Example

```yaml
handlers:
  - name: kafka-to-ws
    match: "__kafka_consume__"
    builtin: kafka-consume
    brokers:
      - "kafka1:9092"
    topic: "live-events"
    group: "xwebs-bridge"
    offset: "latest"
    respond: '{"type": "kafka_event", "data": {{.Message | toJSON}}}'
```

---

## Edge Cases

- Consumer group offsets are committed automatically; restarting xwebs resumes from the last committed offset.
- Using `offset: earliest` on an existing consumer group has no effect if offsets are already committed; reset with `kafka-consumer-groups --reset-offsets` externally.
- Consumed messages are processed by `respond:` but no response is sent to a specific client — combine with `broadcast` to fan out to all WebSocket clients.
- Kafka broker connection errors are logged and retried with exponential backoff.
