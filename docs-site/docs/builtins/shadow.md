---
title: shadow
description: Silently dispatch a message to a secondary handler while the primary handler responds normally.
---

# `shadow` — Shadow Dispatch

**Mode:** `[S]` (server only)

Sends the incoming message to a named `target:` handler in the background without affecting the primary response flow. The target handler runs concurrently and its output (if any) is discarded — the client only receives the response from the primary handler chain. Any errors from the shadow target are logged but not surfaced to the client.

Use `shadow` for dark launches, traffic mirroring, audit logging, analytics side-effects, or testing a new handler against live traffic without impacting clients.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `target` | string | Yes | — | Name of the handler to invoke silently. The target's `match:` expression is bypassed — it is called unconditionally. |

---

## Template Variables Injected

None. The shadow target receives the complete original handler context.

---

## Examples

**Mirror traffic to a new experimental handler without affecting clients:**

```yaml
handlers:
  - name: stable-handler
    match:
      jq: '.type == "query"'
    builtin: shadow
    target: experimental-query
    respond: '{"result": "stable"}'

  - name: experimental-query
    match: "__never__"   # only called via shadow
    run: ./new_query_engine.sh "{{.Message | shellEscape}}"
    # response is discarded; errors are logged
```

**Audit log side-channel:**

```yaml
handlers:
  - name: secure-action
    match:
      jq: '.type == "admin_action"'
    builtin: shadow
    target: audit-logger
    respond: '{"executed": true}'

  - name: audit-logger
    match: "__never__"
    builtin: file-write
    path: "/var/log/xwebs/audit.jsonl"
    mode: append
    content: '{"ts":"{{now | formatTime "RFC3339"}}","conn":"{{.ConnectionID}}","msg":{{.Message | toJSON}}}\n'
```

**Dark launch — call new microservice with live traffic:**

```yaml
handlers:
  - name: main-responder
    match:
      jq: '.type == "recommend"'
    builtin: shadow
    target: new-recommender
    run: ./old_recommender.sh "{{.Message | shellEscape}}"
    respond: '{"recommendations": {{.Stdout | toJSON}}}'

  - name: new-recommender
    match: "__never__"
    builtin: http
    method: POST
    url: "http://new-service.internal/recommend"
    body: '{{.Message}}'
```

---

## Edge Cases

- The shadow target runs in a separate goroutine. If it panics or times out, the error is logged and the primary response is unaffected.
- The shadow target does not send any response to the client — its `respond:` and `run:` stdout are discarded.
- `target:` must be a handler name defined in the same config file. Referencing an unknown handler name logs an error on startup.
- If the primary handler itself uses `drop`, the shadow dispatch still fires — shadowing happens before the drop takes effect.
