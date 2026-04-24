---
title: webhook-hmac
description: POST a signed HTTP webhook with an HMAC-SHA256 signature header for receiver verification.
---

# `webhook-hmac` — Signed Webhook

**Mode:** `[*]` (client and server)

Identical to `webhook` but adds a `X-Hub-Signature-256` (or custom `signature_header:`) HMAC-SHA256 signature over the request body, computed using `secret:`. This allows the receiving endpoint to verify that the request originated from xwebs.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | Yes | — | Webhook endpoint URL. Template expression. |
| `secret` | string | Yes | — | HMAC secret key. Template expression (use `{{env "WEBHOOK_SECRET"}}`). |
| `body` | string | No | `.Message` | Request body. Template expression. |
| `signature_header` | string | No | `X-Hub-Signature-256` | Header name for the HMAC signature. |
| `headers` | map | No | — | Additional HTTP headers. |
| `timeout` | string | No | `30s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.HttpBody}}` | string | Response body. |
| `{{.HttpStatus}}` | int | HTTP response status code. |

---

## Example

```yaml
handlers:
  - name: signed-notify
    match:
      jq: '.type == "deploy"'
    builtin: webhook-hmac
    url: "https://ci.example.com/webhook"
    secret: '{{env "WEBHOOK_SECRET"}}'
    body: '{{.Message | toJSON}}'
    headers:
      Content-Type: "application/json"
    respond: '{"delivered": true, "status": {{.HttpStatus}}}'
```

---

## Edge Cases

- The signature format is `sha256=<hex>`, matching GitHub webhook conventions.
- The `secret:` field is rendered as a template; never hardcode secrets — use `{{env "..."}}`.
- Non-2xx responses do not cause handler errors; check `{{.HttpStatus}}`.
- Use `signature_header:` to match the header name expected by your receiver.
