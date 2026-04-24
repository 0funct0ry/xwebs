---
title: webhook
description: POST an outgoing HTTP webhook to a URL; response available as {{.HttpBody}} and {{.HttpStatus}}.
---

# `webhook` — HTTP Webhook

**Mode:** `[*]` (client and server)

Sends an HTTP POST request to a configurable URL. The request body and headers are template expressions. The upstream response body and status code are injected into `{{.HttpBody}}` and `{{.HttpStatus}}` for use in `respond:`.

For signed webhooks, use `webhook-hmac` instead.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | Yes | — | Webhook endpoint URL. Template expression. |
| `body` | string | No | `.Message` | Request body. Template expression. |
| `headers` | map | No | — | HTTP headers to include. Values are template expressions. |
| `timeout` | string | No | `30s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.HttpBody}}` | string | Response body from the webhook endpoint. |
| `{{.HttpStatus}}` | int | HTTP response status code. |

---

## Example

```yaml
handlers:
  - name: notify-slack
    match:
      jq: '.type == "alert"'
    builtin: webhook
    url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    body: '{"text": "Alert: {{.Message | jq ".message"}}"}'
    headers:
      Content-Type: "application/json"
    respond: '{"notified": true, "status": {{.HttpStatus}}}'
```

---

## Edge Cases

- Non-2xx responses do not cause the handler to error by default; check `{{.HttpStatus}}` in `respond:` to handle failures.
- Connection timeouts and DNS failures do cause the handler to error.
- Response bodies larger than the configured max are truncated; the full body is available only in `run:` via `curl`.
- Use `webhook-hmac` to add an HMAC-SHA256 signature header for receiving-end verification.
