---
title: forward
description: Proxy an incoming message to another WebSocket server and capture the upstream reply.
---

# `forward` — Forward to Another WebSocket

**Mode:** `[S]` (server only)

Proxies the incoming message to an upstream WebSocket server at `target:` and waits for the first reply. The upstream response is injected into `{{.ForwardReply}}` so you can use it in a `respond:` template. The connection to the upstream server is opened on demand and closed after each exchange unless `persistent:` is enabled.

This builtin is useful for fan-out gateways, protocol bridges, and transparent inspection layers where the server must delegate to another WebSocket endpoint.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target` | string | Yes | — | Upstream WebSocket URL (template expression allowed, e.g. `wss://backend.example.com/ws`). |
| `timeout` | string | No | `30s` | Maximum time to wait for the upstream reply. |
| `persistent` | bool | No | `false` | Keep the upstream connection open across messages instead of reconnecting each time. |
| `headers` | map | No | — | Additional HTTP headers to send during the upstream handshake (values are template-rendered). |
| `message` | string | No | `.Message` | Template for the payload forwarded upstream. Defaults to the raw incoming message. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.ForwardReply}}` | string | The first text or binary frame received from the upstream server. Empty string if the upstream returned nothing before the timeout. |

---

## Examples

**Transparent proxy — forward and relay the reply verbatim:**

```yaml
handlers:
  - name: upstream-proxy
    match: "*"
    builtin: forward
    target: wss://backend.internal/ws
    respond: "{{.ForwardReply}}"
```

**Transform before forwarding, wrap the reply:**

```yaml
handlers:
  - name: bridge
    match:
      jq: '.type == "query"'
    builtin: forward
    target: "wss://data-service.example.com/ws"
    message: '{"q": {{.Message | jq ".payload" | toJSON}}}'
    respond: |
      {
        "type": "result",
        "data": {{.ForwardReply}},
        "forwarded_at": "{{now | formatTime "RFC3339"}}"
      }
```

**Route to different upstreams based on message content:**

```yaml
handlers:
  - name: dynamic-forward
    match:
      jq: '.service != null'
    builtin: forward
    target: 'wss://{{.Message | jq ".service"}}.internal/ws'
    timeout: 10s
    respond: "{{.ForwardReply}}"
```

---

## Edge Cases

- If the upstream server closes the connection or does not reply within `timeout:`, `{{.ForwardReply}}` is an empty string and no error is surfaced to the client unless `respond:` explicitly checks it.
- With `persistent: true`, a single upstream connection is shared for all messages on a handler; if the upstream disconnects, the next message will trigger a reconnect.
- Binary frames received from the upstream are decoded to a UTF-8 string in `{{.ForwardReply}}`; if the bytes are not valid UTF-8, they are base64-encoded.
- `target:` is rendered as a template — any template error will cause the handler to fail and log an error without sending a client response.
