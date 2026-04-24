---
title: throttle-broadcast
description: Broadcast to all clients while skipping any client that already received a message within a recent time window.
---

# `throttle-broadcast` — Throttled Broadcast

**Mode:** `[S]` (server only)

Broadcasts the current message to all connected clients, but silently skips any client that has already received a message from this handler within the last `window:` duration. This prevents high-frequency event sources from overwhelming slow or recently-served clients.

The per-client delivery timestamp is tracked in memory. Each time a message is delivered to a client, that client's timer resets.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `window` | string | Yes | — | Minimum duration between deliveries to any individual client. Go duration string (`500ms`, `1s`, `5s`). |
| `message` | string | No | `.Message` | Payload template. Defaults to the raw incoming message. |

---

## Template Variables Injected

None. Standard context is available in `message:`.

---

## Examples

**Throttle a sensor feed to at most one message per second per client:**

```yaml
handlers:
  - name: sensor-feed
    match:
      jq: '.type == "sensor_data"'
    builtin: throttle-broadcast
    window: "1s"
```

**Throttle with a custom payload:**

```yaml
handlers:
  - name: price-ticker
    match:
      jq: '.type == "tick"'
    builtin: throttle-broadcast
    window: "500ms"
    message: |
      {
        "type": "tick",
        "price": {{.Message | jq ".price"}},
        "ts": "{{now | formatTime "RFC3339"}}"
      }
```

**Slower throttle for dashboard updates:**

```yaml
handlers:
  - name: dashboard-update
    match:
      jq: '.type == "metrics"'
    builtin: throttle-broadcast
    window: "5s"
    respond: '{"throttled_broadcast": true}'
```

---

## Edge Cases

- A client that connects after the throttle window is always delivered the next message — the window starts from the first delivery, not from connection time.
- Messages skipped for a client are discarded; they are not queued for later delivery once the window expires.
- The delivery tracking state is per-handler instance, not global. Two `throttle-broadcast` handlers with the same `window:` maintain independent per-client clocks.
- Restarting the server resets all throttle state; clients will each receive the next message regardless of how recently they received one before the restart.
