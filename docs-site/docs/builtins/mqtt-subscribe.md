---
title: mqtt-subscribe
description: Subscribe to an MQTT broker topic and inject received messages into the handler pipeline.
---

# `mqtt-subscribe` — MQTT Subscribe

**Mode:** `[S]` (server only)

Opens a persistent MQTT subscription. Messages received from the broker topic are injected into the handler pipeline and can be forwarded to WebSocket clients via `respond:` or `broadcast`.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `broker` | string | Yes | — | MQTT broker URL. |
| `topic` | string | Yes | — | MQTT topic or wildcard (e.g. `sensors/#`, `device/+/status`). |
| `qos` | integer | No | `0` | QoS level for the subscription. |

---

## Example

```yaml
handlers:
  - name: mqtt-to-ws
    match: "__mqtt_subscribe__"
    builtin: mqtt-subscribe
    broker: "mqtt://localhost:1883"
    topic: "sensors/#"
    respond: '{"type": "sensor", "data": {{.Message | toJSON}}}'
```

---

## Edge Cases

- MQTT wildcards (`#`, `+`) are supported; `#` matches all subtopics recursively.
- The subscription is established when the handler loads, not on WebSocket connect.
- If the broker disconnects, xwebs will reconnect with exponential backoff and re-subscribe.
- Messages received before any WebSocket client connects are processed but `respond:` has no recipients; use `broadcast` to forward to all connected clients.
