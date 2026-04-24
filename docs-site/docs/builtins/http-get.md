---
title: http-get
description: Perform an HTTP GET request and inject the response body as {{.HttpBody}}.
---

# `http-get` — HTTP GET

**Mode:** `[*]` (client and server)

Sends an HTTP GET request to `url:` and injects the response body into `{{.HttpBody}}`. Useful for fetching external data to enrich a WebSocket response without spawning a shell process.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | Yes | — | Target URL. Template expression. |
| `headers` | map | No | — | HTTP request headers. Values are template expressions. |
| `timeout` | string | No | `30s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.HttpBody}}` | string | HTTP response body. |
| `{{.HttpStatus}}` | int | HTTP response status code. |

---

## Example

```yaml
handlers:
  - name: fetch-price
    match:
      jq: '.type == "price_check"'
    builtin: http-get
    url: 'https://api.example.com/price/{{.Message | jq ".symbol"}}'
    headers:
      Authorization: "Bearer {{env "API_KEY"}}"
    respond: '{"price": {{.HttpBody | jq ".price"}}}'
```

---

## Edge Cases

- Non-2xx responses populate `{{.HttpBody}}` with the error body and `{{.HttpStatus}}` with the status code; no error is raised.
- Connection failures and timeouts cause the handler to error.
- The URL is template-rendered; user-supplied data in the URL should be URL-encoded with `{{.Value | urlEncode}}`.
- For POST/PUT requests, use `http` (the general HTTP builtin) instead.
