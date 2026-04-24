---
title: Match Strategies
description: All nine message matching strategies in xwebs with examples for each.
---

# Match Strategies

Every handler must declare a `match` condition. xwebs evaluates the match against each incoming message and dispatches to the handler if the condition is satisfied. Nine strategies are available, from simple string matching to composite logical expressions.

## Strategy Reference

| Strategy | Syntax | Use case |
|----------|--------|----------|
| Glob | `match: "pattern"` | Simple wildcard matching on raw text |
| Regex | `match.regex: "expr"` | PCRE regex on raw message |
| jq boolean | `match.jq: "expr"` | JSON field conditions |
| JSON Path + equals | `match.json_path` + `match.equals` | Extract and compare a single field |
| JSON Schema | `match.json_schema: "file.json"` | Full schema validation |
| Go template | `match.template: "expr"` | Arbitrary multi-condition logic |
| Binary | `match.binary: true/false` | Frame type (binary vs text) |
| Composite AND | `match.all: [...]` | All conditions must match |
| Composite OR | `match.any: [...]` | Any condition matches |

---

## Glob

Glob is the default and simplest strategy. When `match:` is a plain string, it's treated as a glob pattern matched against the raw message text.

| Wildcard | Matches |
|----------|---------|
| `*` | Any sequence of characters |
| `?` | Any single character |

```yaml
- name: catch-all
  match: "*"
  respond: '{"echo":{{.Message | toJSON}}}'

- name: ping-exact
  match: "ping"
  respond: "pong"

- name: cmd-prefix
  match: "cmd:*"
  run: ./dispatch.sh "{{.Message}}"

- name: error-detect
  match: "*error*"
  run: notify-slack.sh "{{.Message}}"
```

Glob is evaluated case-sensitively against the raw message bytes. For case-insensitive matching, use a template or regex strategy.

---

## Regex

PCRE-compatible regular expression matched against the raw message text.

```yaml
- name: json-ping
  match:
    regex: '"type"\s*:\s*"ping"'
  respond: '{"type":"pong"}'

- name: numeric-id
  match:
    regex: '"id"\s*:\s*\d+'
  run: process-by-id.sh
```

Regex is the right choice when you need character-level control (anchors, groups, quantifiers) that glob doesn't provide. For structured JSON, jq is usually cleaner.

---

## jq Boolean

A jq expression evaluated against the message. If the expression returns `true` (or a truthy value), the handler matches.

```yaml
- name: deploy-trigger
  match:
    jq: '.type == "deploy" and .env == "production"'
  run: ./deploy.sh

- name: critical-alerts
  match:
    jq: '.severity >= 3 and .acknowledged == false'
  run: page-oncall.sh

- name: nested-field
  match:
    jq: '.event.source == "github" and (.event.action | test("push|release"))'
  run: ci-trigger.sh

- name: error-messages
  match:
    jq: '.error != null'
  run: log-error.sh
```

jq is the recommended strategy for any JSON protocol. It's expressive, readable, and safe — it does not execute shell commands and handles malformed JSON gracefully (non-JSON messages simply don't match).

---

## JSON Path + Equals

Extract a field using a JSONPath expression and compare its value to a literal string.

```yaml
- name: auth-request
  match:
    json_path: "$.type"
    equals: "auth"
  run: verify-auth.sh

- name: v2-messages
  match:
    json_path: "$.api_version"
    equals: "2"
  respond: '{"api_version":2,"ok":true}'

- name: admin-action
  match:
    json_path: "$.user.role"
    equals: "admin"
  run: admin-handler.sh
```

Use JSON Path when you need to extract a deeply nested field without writing a full jq expression. The `equals` value is always a string comparison.

---

## JSON Schema

Validates the message against a [JSON Schema](https://json-schema.org/) file. The handler matches if the message is valid JSON and satisfies all schema constraints.

```yaml
- name: user-event
  match:
    json_schema: "schemas/user_event.json"
  run: process-user-event.sh

- name: order-create
  match:
    json_schema: "schemas/order.json"
  run: create-order.sh
```

Example schema file (`schemas/user_event.json`):

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["type", "user_id"],
  "properties": {
    "type": {"type": "string", "enum": ["login", "logout", "update"]},
    "user_id": {"type": "integer", "minimum": 1}
  }
}
```

JSON Schema is the right choice when you have formal API contracts and want to enforce them in the handler layer rather than in your shell script.

---

## Go Template (Truthy)

A Go template expression evaluated against the message context. The handler matches if the template produces a non-empty, non-zero, non-false output.

```yaml
- name: large-message
  match:
    template: '{{gt .MessageLen 1024}}'
  run: warn-large-message.sh

- name: json-with-error
  match:
    template: '{{and (isJSON .Message) (contains "error" .Message)}}'
  run: handle-json-error.sh

- name: even-messages
  match:
    template: '{{eq (mod .MessageIndex 2) 0}}'
  run: sample-even.sh
```

The template strategy is the most flexible — it has access to the full template context including `.MessageLen`, `.MessageIndex`, `.ConnectionID`, `.Vars`, and all template functions. Use it when no other strategy expresses your condition cleanly.

---

## Binary

Matches based on WebSocket frame type: `binary: true` matches only binary frames (opcode `0x2`); `binary: false` matches only text frames (opcode `0x1`).

```yaml
- name: handle-binary
  match:
    binary: true
  run: xxd    # hex dump the binary payload

- name: handle-text
  match:
    binary: false
  run: jq '.'

- name: protobuf-handler
  match:
    binary: true
  builtin: file-write
  path: /tmp/incoming.pb
```

This strategy is commonly combined with `any:` composite matchers to route binary and text to different processing paths.

---

## Composite AND (`all:`)

All conditions in the list must match. Conditions can be any combination of the above strategies.

```yaml
- name: critical-deploy
  match:
    all:
      - jq: '.type == "deploy"'
      - jq: '.env == "production"'
      - binary: false
  concurrent: false
  run: ./deploy.sh

- name: large-json
  match:
    all:
      - binary: false
      - template: '{{gt .MessageLen 512}}'
      - regex: '^\{'
  run: process-large-json.sh
```

---

## Composite OR (`any:`)

Any condition in the list matches. Useful for routing messages from multiple clients or protocols through a single handler.

```yaml
- name: admin-or-system
  match:
    any:
      - jq: '.role == "admin"'
      - jq: '.source == "system"'
      - glob: "*INTERNAL*"
  run: privileged-handler.sh

- name: ping-variants
  match:
    any:
      - glob: "ping"
      - jq: '.type == "ping"'
      - jq: '.type == "heartbeat"'
  respond: '{"type":"pong","ts":"{{now}}"}'
```

---

## Combining Strategies

`all:` and `any:` can be nested:

```yaml
- name: complex-match
  match:
    all:
      - binary: false
      - any:
          - jq: '.role == "admin"'
          - jq: '.priority >= 5'
  run: high-priority-handler.sh
```

---

## Performance Tips

- **Glob and regex** are evaluated on raw bytes — fastest for non-JSON messages.
- **jq** parses JSON only if the message is valid JSON — safe for mixed traffic.
- **JSON Schema** is the most expensive — cache the compiled schema (xwebs does this automatically).
- **Priority** controls evaluation order: set `priority: 100` on frequently-matching handlers to short-circuit the list early.
- **exclusive: true** stops evaluation after the first match, reducing work for high-throughput servers.
