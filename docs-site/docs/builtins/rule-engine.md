---
title: rule-engine
description: Evaluate an ordered list of condition/response rules and send the first matching response.
---

# `rule-engine` — Rule Engine

**Mode:** `[*]` (client and server)

Evaluates a list of `rules:` in order. Each rule has a `condition:` (a jq boolean expression evaluated against the incoming message) and a `respond:` template. The first rule whose condition is truthy wins, and its response is sent. If no rule matches, the handler produces no response. This lets you express branching logic declaratively without multiple handlers.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rules` | list of rule objects | Yes | — | Ordered list of condition/response pairs. Evaluated top to bottom; first match wins. |

**Rule object fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `condition` | string | Yes | jq boolean expression. Evaluated against the raw message. A truthy result (non-false, non-null) triggers the rule. |
| `respond` | string | Yes | Go template rendered as the response when this rule matches. |

---

## Template Variables Injected

None. Standard context is available in every `respond:` template.

---

## Examples

**Route messages to different responses based on type:**

```yaml
handlers:
  - name: type-router
    match: "*"
    builtin: rule-engine
    rules:
      - condition: '.type == "ping"'
        respond: '{"type": "pong", "ts": "{{nowUnix}}"}'
      - condition: '.type == "echo"'
        respond: '{{.Message}}'
      - condition: '.type == "status"'
        respond: '{"clients": {{.ClientCount}}, "uptime": "{{uptime}}"}'
```

**Priority-based error classification:**

```yaml
handlers:
  - name: classify-error
    match:
      jq: '.level == "error"'
    builtin: rule-engine
    rules:
      - condition: '.code >= 500'
        respond: '{"severity": "critical", "alert": true}'
      - condition: '.code >= 400'
        respond: '{"severity": "warning", "alert": false}'
      - condition: 'true'
        respond: '{"severity": "info", "alert": false}'
```

**Business logic branching:**

```yaml
handlers:
  - name: order-handler
    match:
      jq: '.type == "order"'
    builtin: rule-engine
    rules:
      - condition: '.total > 1000'
        respond: '{"status": "requires_approval", "limit": 1000}'
      - condition: '.items | length == 0'
        respond: '{"status": "rejected", "reason": "empty_order"}'
      - condition: 'true'
        respond: '{"status": "accepted", "id": "{{uuid}}"}'
```

---

## Edge Cases

- Rules are evaluated in order; put more specific conditions before catch-all conditions like `true`.
- If a `condition:` jq expression errors (e.g., references a field that doesn't exist), that rule is treated as non-matching and evaluation continues to the next rule.
- A catch-all rule with `condition: 'true'` at the end guarantees a response even if all other conditions fail.
- The handler-level `match:` still applies — `rule-engine` only runs for messages that pass the outer match filter. Use `match: "*"` to apply the rule engine to all messages.
