# Example WebSocket Handlers

This directory contains several examples of declarative message handlers for `xwebs`. You can use these files with the `--handlers` flag to automate tasks, log traffic, or build reactive clients.

## Handlers Overview

### 1. Standard Handlers (`handlers.yaml`)
Basic examples of text, regex, and json matching.

| Handler Name        | Match Condition                               | Actions                         | Example Server Message       |
|---------------------|-----------------------------------------------|---------------------------------|------------------------------|
| `logger`            | `type: "text"`, `pattern: "*"`                | `log`: "Received: {{.Data}}"    | *Any message*                |
| `json_path_matcher` | `json_path: "user.id"`, `equals: "dev-01"`    | `log`: "Hello developer!"       | `{"user": {"id": "dev-01"}}` |
| `auto_responder`    | `type: "json"`, `pattern: "$.type == 'ping'"` | `send`: `{"type": "pong", ...}` | `{"type": "ping"}`           |
| `system_info`       | `type: "text"`, `pattern: "status"`           | `shell`: `uptime`               | `status`                     |
| `system_monitor`    | `regex: "^status:(cpu\|mem\|disk)$"`          | `log`, `shell`: `top -l 1 ...`  | `status:cpu`                 |

### 2. JQ Advanced Matchers (`jq_handlers.yaml`)
Demonstrates complex logical matching using `gojq` queries.

| Handler Name   | JQ Query                                      | Actions                               | Example Server Message                     |
|----------------|-----------------------------------------------|---------------------------------------|--------------------------------------------|
| `jq_release`   | `.type == "release" and .env == "production"` | `log`: "Triggered release handler..." | `{"type": "release", "env": "production"}` |
| `jq_admin`     | `.user.role == "admin"`                       | `log`: "Admin action detected!"       | `{"user": {"role": "admin"}}`              |
| `jq_tag_match` | `.tags \| contains(["urgent"])`               | `log`: "Urgent tag found!"            | `{"tags": ["urgent", "normal"]}`           |

### 3. JSON Path Matchers (`json_path_handlers.yaml`)
Demonstrates focused field-based matching with `json_path` and `equals`.

| Handler Name         | JSONPath         | Equals Value | Example Server Message                  |
|----------------------|------------------|--------------|-----------------------------------------|
| `auth_event`         | `event.type`     | `"login"`    | `{"event": {"type": "login"}}`          |
| `system_alert`       | `$.status.level` | `5`          | `{"status": {"level": 5}}`              |
| `maintenance_mode`   | `maintenance`    | `true`       | `{"maintenance": true}`                 |
| `root_value_match`   | `$`              | `"PING"`     | `"PING"`                                |
| `nested_array_field` | `meta.tags[0]`   | `"urgent"`   | `{"meta": {"tags": ["urgent", "low"]}}` |

## Usage

To use any of these handler files, run `xwebs` with the `--handlers` flag:

```bash
# Example: Use the JSON Path examples
xwebs connect wss://echo.websocket.org --handlers examples/handlers/json_path_handlers.yaml
```
