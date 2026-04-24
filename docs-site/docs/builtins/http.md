---
title: http
description: Make an HTTP request from a handler and inject the response body and status into templates.
---

# `http` — HTTP Request

**Mode:** `[*]` (client and server)

Makes an HTTP request from within a handler. The response body and status code are available in `respond:` as `{{.HttpBody}}` and `{{.HttpStatus}}`.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `method` | string | No | `"GET"` | HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `url` | string | Yes | — | Request URL. Supports template expressions. |
| `headers` | map | No | — | HTTP headers to include. Values support template expressions. |
| `body` | string | No | — | Request body. Supports template expressions. |
| `timeout` | string | No | `"30s"` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.HttpBody}}` | string | Response body |
| `{{.HttpStatus}}` | int | HTTP status code |

---

## Examples

**GET request:**

```yaml
handlers:
  - name: fetch-weather
    match:
      jq: '.type == "weather"'
    builtin: http
    method: GET
    url: 'https://api.weather.example.com/current?city={{.Message | jq ".city" | urlEncode}}'
    respond: |
      {
        "type": "weather_response",
        "status": {{.HttpStatus}},
        "data": {{.HttpBody}}
      }
```

**POST with JSON body:**

```yaml
handlers:
  - name: create-record
    match:
      jq: '.type == "create"'
    builtin: http
    method: POST
    url: 'https://api.example.com/records'
    headers:
      Authorization: 'Bearer {{env "API_TOKEN"}}'
      Content-Type: 'application/json'
    body: '{{.Message | jq ".data" | toJSON}}'
    respond: |
      {
        "created": {{if eq .HttpStatus 201}}true{{else}}false{{end}},
        "response": {{.HttpBody}}
      }
```

**Webhook forward:**

```yaml
handlers:
  - name: forward-to-webhook
    match: "*"
    builtin: http
    method: POST
    url: 'https://hooks.example.com/ws-events'
    headers:
      Content-Type: 'application/json'
      X-Source: 'xwebs'
    body: '{{.Message}}'
    respond: '{"forwarded":true,"webhook_status":{{.HttpStatus}}}'
```

**Chain HTTP calls in a pipeline:**

```yaml
handlers:
  - name: auth-and-fetch
    match:
      jq: '.type == "fetch"'
    pipeline:
      - builtin: http
        method: POST
        url: 'https://auth.example.com/token'
        body: '{"client_id":"{{env "CLIENT_ID"}}","secret":"{{env "CLIENT_SECRET"}}"}'
        as: auth
      - builtin: http
        method: GET
        url: 'https://api.example.com/data'
        headers:
          Authorization: 'Bearer {{.Steps.auth.HttpBody | jq ".token"}}'
        as: data
    respond: '{{.Steps.data.HttpBody}}'
```

---

## Edge Cases

- Network errors (connection refused, timeout) set `{{.HttpStatus}}` to `0` and `{{.HttpBody}}` to the error message.
- Non-2xx responses do not cause handler errors — check `{{.HttpStatus}}` in your `respond:` template.
- For simple GET requests, consider `http-get` which has a more concise config.
- For POST with HMAC signing, use `webhook-hmac`.
