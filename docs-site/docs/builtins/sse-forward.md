---
title: sse-forward
description: Forward a WebSocket message as a Server-Sent Event to a named SSE stream.
---

# `sse-forward` — SSE Forward

**Mode:** `[S]` (server only)

Converts an incoming WebSocket message into a Server-Sent Event (SSE) and pushes it to a named SSE stream served by xwebs. HTTP clients subscribed to that stream receive the event in real time. This lets you bridge WebSocket publishers with SSE-consuming browser clients without an external message bus.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `stream` | string | Yes | — | Named SSE stream identifier. Template expression. |
| `event` | string | No | `message` | SSE event type name. Template expression. |
| `data` | string | No | `.Message` | SSE data payload. Template expression. |
| `id` | string | No | — | Optional SSE event ID for client-side reconnect tracking. Template expression. |

---

## Example

```yaml
handlers:
  - name: ws-to-sse
    match:
      jq: '.type == "live_update"'
    builtin: sse-forward
    stream: "live"
    event: '{{.Message | jq ".event_type"}}'
    data: '{{.Message | jq ".payload" | toJSON}}'
    respond: '{"forwarded_to_sse": true}'
```

---

## Edge Cases

- SSE streams are served at `/sse/<stream>` by default; the exact path depends on xwebs server configuration.
- If no HTTP clients are subscribed to the named stream, the event is discarded silently.
- SSE only supports text (UTF-8); binary message payloads should be base64-encoded in `data:`.
- SSE connections are long-lived HTTP responses; ensure your reverse proxy (nginx, Caddy) does not buffer them.
