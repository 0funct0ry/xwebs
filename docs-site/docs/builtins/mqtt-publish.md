---
title: mqtt-publish
description: Publish a message to an MQTT broker topic with configurable QoS and retain flag.
---

# `mqtt-publish` — MQTT Publish

**Mode:** `[*]` (client and server)

Publishes a payload to an MQTT broker. Useful for bridging WebSocket events to IoT devices or MQTT-based systems.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `broker` | string | Yes | — | MQTT broker URL (e.g. `mqtt://localhost:1883`, `mqtts://broker.example.com:8883`). |
| `topic` | string | Yes | — | MQTT topic. Template expression. |
| `message` | string | No | `.Message` | Payload to publish. Template expression. |
| `qos` | integer | No | `0` | QoS level: 0 (at most once), 1 (at least once), 2 (exactly once). |
| `retain` | bool | No | `false` | Whether the broker should retain the message for new subscribers. |

---

## Example

```yaml
handlers:
  - name: ws-to-mqtt
    match:
      jq: '.type == "sensor_update"'
    builtin: mqtt-publish
    broker: "mqtt://localhost:1883"
    topic: 'sensors/{{.Message | jq ".device_id"}}'
    message: '{{.Message | jq ".value" | toJSON}}'
    qos: 1
    respond: '{"published": true}'
```

---

## Edge Cases

- The MQTT connection is opened per-handler invocation by default; for high-frequency publishes, configure a persistent connection at the application level.
- QoS 2 has higher latency due to the four-part handshake; use QoS 0 or 1 for low-latency scenarios.
- `retain: true` causes the broker to deliver the message to new subscribers immediately on subscription.
- MQTT broker authentication (username/password, TLS client certs) is configured via the `broker` URL or additional fields.
