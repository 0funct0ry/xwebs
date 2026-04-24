---
title: metric
description: Increment a named Prometheus counter with optional labels from a handler.
---

# `metric` — Prometheus Counter

**Mode:** `[*]` (client and server)

Increments a named Prometheus counter metric. Metrics are exposed at `/api/metrics` when the server is started with `--metrics`. Use this builtin to track business-level events without writing shell commands.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | — | Counter metric name. Should follow Prometheus naming conventions (`snake_case`). |
| `labels` | map | No | — | Label key-value pairs. Values support template expressions. |
| `value` | float | No | `1.0` | Amount to increment by. |

---

## Examples

**Count messages by type:**

```yaml
handlers:
  - name: count-pings
    match:
      jq: '.type == "ping"'
    builtin: metric
    name: "xwebs_pings_total"
    respond: '{"type":"pong"}'
```

**Track deploys with labels:**

```yaml
handlers:
  - name: count-deploys
    match:
      jq: '.type == "deploy"'
    builtin: metric
    name: "xwebs_deploys_total"
    labels:
      env: '{{.Message | jq ".env"}}'
      branch: '{{.Message | jq ".branch"}}'
    run: ./deploy.sh
    respond: '{"status":"deploying"}'
```

**Track errors:**

```yaml
handlers:
  - name: count-errors
    match:
      jq: '.error != null'
    builtin: metric
    name: "xwebs_client_errors_total"
    labels:
      error_type: '{{.Message | jq ".error.code"}}'
```

---

## Prometheus Output

With `xwebs serve --port 8080 --metrics`, metrics are available at `http://localhost:8080/api/metrics`:

```
# HELP xwebs_deploys_total
# TYPE xwebs_deploys_total counter
xwebs_deploys_total{branch="main",env="production"} 12
xwebs_deploys_total{branch="feat/x",env="staging"} 3
```

---

## Built-in Metrics

xwebs also exposes these built-in metrics automatically:

| Metric | Description |
|--------|-------------|
| `xwebs_connections_total` | Total connections accepted |
| `xwebs_messages_sent_total` | Messages sent |
| `xwebs_messages_received_total` | Messages received |
| `xwebs_handler_executions_total{handler}` | Handler executions by name |
| `xwebs_handler_duration_seconds{handler}` | Handler duration histogram |
| `xwebs_handler_errors_total{handler}` | Handler errors by name |
