---
title: rate-limit
description: Rate limit incoming messages per client or globally, with configurable window and retry-after response.
---

# `rate-limit` — Rate Limiting

**Mode:** `[*]` (client and server)

Rate limits message processing using a token bucket algorithm. When a client exceeds the configured rate, the message is dropped and `respond:` is sent with `{{.RetryAfter}}` available — the number of seconds until the next token is available.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rate` | string | Yes | — | Rate limit in `N/unit` format: `"10/s"`, `"100/m"`, `"1000/h"` |
| `scope` | string | No | `"client"` | `"client"` — per-connection limit; `"global"` — shared across all connections |
| `respond` | string | No | — | Template sent when the rate limit is exceeded |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.RetryAfter}}` | int | Seconds until the next allowed message |

---

## Examples

**Per-client rate limit:**

```yaml
handlers:
  - name: api-rate-limit
    match: "*"
    builtin: rate-limit
    rate: "10/s"
    respond: '{"error":"rate_limited","retry_after":{{.RetryAfter}}}'
```

**Global rate limit across all clients:**

```yaml
handlers:
  - name: global-throttle
    match: "*"
    builtin: rate-limit
    rate: "1000/m"
    scope: "global"
    respond: '{"error":"server_busy","retry_after_seconds":{{.RetryAfter}}}'
```

**Rate limit a specific action:**

```yaml
handlers:
  - name: deploy-rate-limit
    match:
      jq: '.type == "deploy"'
    builtin: rate-limit
    rate: "2/m"
    respond: '{"error":"too_many_deploys","wait_seconds":{{.RetryAfter}}}'
    exclusive: true

  - name: deploy-handler
    match:
      jq: '.type == "deploy"'
    run: ./deploy.sh
    priority: -1
```

**Rate limit with inline handler:**

```bash
xwebs serve --port 8080 \
  --on '.type == "query" :: run:query.sh :: timeout:5s' \
  --on '* :: respond:{"error":"rate_limited"}' \
```

---

## REPL Example

```
xwebs> :handler add -m "*" --builtin rate-limit --rate "5/s" -R '{"error":"rate_limited","retry_after":{{.RetryAfter}}}'
✓ added handler rate-limit-1
```

---

## Edge Cases

- **Token bucket algorithm:** the bucket is replenished continuously, not in discrete intervals. A rate of `"10/s"` means one token every 100ms, not 10 tokens per second as a burst.
- **Scope `"client"`:** each connection has its own independent bucket. Disconnecting and reconnecting resets the bucket.
- **Scope `"global"`:** all connections share one bucket. Useful for protecting external resources (databases, APIs) from aggregate overload.
- **No response:** if `respond:` is not set on the `rate-limit` handler, the message is silently dropped with no feedback to the client. Always set a `respond:` for user-facing APIs.
- **Priority:** place the `rate-limit` handler at a higher priority than the actual processing handler so rate-limited messages never reach the processor.
