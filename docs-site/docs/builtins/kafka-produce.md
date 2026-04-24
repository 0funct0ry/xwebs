---
title: kafka-produce
description: Produce a message to a Kafka topic with an optional partition key.
---

# `kafka-produce` — Kafka Produce

**Mode:** `[*]` (client and server)

Sends a message to a Kafka topic. An optional `key:` controls partition routing — messages with the same key are guaranteed to land in the same partition.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `brokers` | list of strings | Yes | — | Kafka broker addresses (e.g. `["localhost:9092"]`). |
| `topic` | string | Yes | — | Kafka topic name. Template expression. |
| `message` | string | No | `.Message` | Payload value. Template expression. |
| `key` | string | No | — | Partition key. Template expression. |

---

## Example

```yaml
handlers:
  - name: ws-to-kafka
    match:
      jq: '.type == "order"'
    builtin: kafka-produce
    brokers:
      - "kafka1:9092"
      - "kafka2:9092"
    topic: "orders"
    key: '{{.Message | jq ".order_id"}}'
    message: '{{.Message | toJSON}}'
    respond: '{"produced": true}'
```

---

## Edge Cases

- Messages are produced synchronously by default; the handler waits for broker acknowledgement before sending `respond:`.
- If all brokers are unreachable, the handler errors.
- `key:` is optional; without it, Kafka uses round-robin partition assignment.
- Message size limits depend on Kafka broker configuration (`message.max.bytes`); large payloads may be rejected.
