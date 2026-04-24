---
title: http-mock-respond
description: Register a canned HTTP response at a given path to mock REST endpoints alongside the WebSocket server.
---

# `http-mock-respond` — HTTP Mock Endpoint

**Mode:** `[S]` (server only)

Registers a static or template-rendered HTTP response at a given `path:` on the xwebs HTTP server. HTTP clients that `GET` (or match the configured method) that path receive the canned response. This lets you co-locate a lightweight REST mock alongside your WebSocket server for integration testing.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `path` | string | Yes | — | HTTP path to register (e.g. `/api/status`). |
| `status` | integer | No | `200` | HTTP status code to return. |
| `body` | string | No | `""` | Response body. Template expression rendered once at registration time. |
| `content_type` | string | No | `application/json` | Content-Type header value. |
| `method` | string | No | `GET` | HTTP method to match. |

---

## Example

```yaml
handlers:
  - name: mock-health
    match:
      jq: '.type == "register_mock"'
    builtin: http-mock-respond
    path: "/api/health"
    status: 200
    body: '{"status": "ok", "registered_at": "{{now | formatTime "RFC3339"}}"}'
    content_type: "application/json"
    respond: '{"mock_registered": true}'
```

---

## Edge Cases

- The mock endpoint is registered dynamically when the handler fires — not at server startup. Send a trigger message to activate it.
- Registering the same path twice overwrites the previous mock response.
- Template expressions in `body:` are rendered once at registration time, not on each HTTP request. For dynamic responses, use a `run:` step instead.
- The registered path is scoped to the current server process and is cleared on restart.
