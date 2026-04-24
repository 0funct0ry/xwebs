---
title: round-robin
description: Distribute incoming messages across a pool of client IDs in rotation.
---

# `round-robin` — Round-Robin Distribution

**Mode:** `[S]` (server only)

Sends each matched message to the next client in a fixed `pool:` list, rotating through the list in order. After the last client in the pool receives a message, the next message goes to the first client again. Clients that have disconnected are skipped and the rotation continues.

Use this for work-queue distribution, load balancing across a set of worker clients, or partitioned message delivery.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `pool` | list of strings | Yes | — | Ordered list of client connection IDs. Each item is rendered as a Go template. |
| `message` | string | No | `.Message` | Payload sent to the selected client. Rendered as a Go template. |

---

## Template Variables Injected

None. Standard context is available in `pool` items and `message:`.

---

## Examples

**Distribute work across three worker clients:**

```yaml
handlers:
  - name: work-distributor
    match:
      jq: '.type == "job"'
    builtin: round-robin
    pool:
      - "worker-1"
      - "worker-2"
      - "worker-3"
    message: |
      {
        "type": "job",
        "payload": {{.Message | jq ".payload" | toJSON}},
        "assigned_at": "{{now | formatTime "RFC3339"}}"
      }
    respond: '{"dispatched": true}'
```

**Round-robin with dynamic pool from KV:**

```yaml
handlers:
  - name: dynamic-pool
    match:
      jq: '.type == "task"'
    builtin: round-robin
    pool:
      - '{{kv "workers.0"}}'
      - '{{kv "workers.1"}}'
      - '{{kv "workers.2"}}'
    message: '{{.Message}}'
```

**Two-client ping-pong relay:**

```yaml
handlers:
  - name: relay-pair
    match:
      jq: '.type == "relay"'
    builtin: round-robin
    pool:
      - "c-alpha"
      - "c-beta"
```

---

## Edge Cases

- If all clients in the pool are disconnected, the message is silently dropped without error.
- Disconnected clients are skipped in the rotation; the counter still advances, so the next online client receives the message.
- Pool entries are rendered as templates at each message dispatch, allowing dynamic IDs; template errors for an entry cause that slot to be skipped.
- The rotation counter is per-handler instance and resets on server restart.
