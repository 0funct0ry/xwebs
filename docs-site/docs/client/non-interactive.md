---
title: Non-Interactive Mode
description: Using xwebs connect in pipelines, scripts, and CI/CD automation.
---

# Non-Interactive Mode

When stdin is not a TTY, xwebs automatically enters non-interactive mode. You can also force it explicitly with `--no-interactive`. This mode is designed for pipelines, scripts, and CI/CD workflows.

## Flags

| Flag | Description |
|------|-------------|
| `--once` | Exit after receiving the first response |
| `--input <file>` | Send messages from a file (one per line) |
| `--send <message>` | Send a message on connect |
| `--expect <n>` | Exit after receiving N responses |
| `--until <template>` | Exit when the template evaluates truthy |
| `--output <file>` | Write responses to a file |
| `--jsonl` | Print one JSON response per line (for piping to jq) |
| `--script <file>` | Execute a .xwebs script file |
| `--watch <file>` | Re-send the file content whenever it changes |
| `--timeout <dur>` | Exit after this duration |
| `--exit-on <template>` | Exit with a code based on template output |

---

## Patterns and Examples

### Send and receive once

The most common pattern for scripting: send one message, wait for one response, exit.

```bash
echo '{"action":"status"}' | xwebs connect wss://api.example.com --once
```

Or with `--send`:

```bash
xwebs connect wss://api.example.com \
  --send '{"type":"ping"}' \
  --once
```

### Pipe JSON responses to jq

`--jsonl` ensures one message per line, making the output safe to pipe to `jq`:

```bash
xwebs connect wss://stream.example.com --jsonl | jq '.price'
```

Filter and aggregate:

```bash
xwebs connect wss://events.example.com --jsonl \
  | jq 'select(.type == "trade") | .amount' \
  | awk '{sum += $1} END {print sum}'
```

### Send multiple messages from a file

```bash
# messages.txt — one message per line
echo '{"type":"subscribe","channel":"trades"}' > messages.txt
echo '{"type":"subscribe","channel":"quotes"}' >> messages.txt

xwebs connect wss://api.example.com --input messages.txt --expect 2
```

### Wait until a condition is met

```bash
xwebs connect wss://job-api.example.com \
  --send '{"job_id":"abc123","subscribe":true}' \
  --until '{{eq (.Message | jq ".status") "complete"}}' \
  --output result.json
```

When the `--until` template evaluates to a truthy value, xwebs writes the triggering message to `--output` and exits.

### Timeout

```bash
xwebs connect wss://api.example.com \
  --send '{"query":"data"}' \
  --timeout 5s \
  --once
```

If no response arrives within 5 seconds, xwebs exits with a non-zero exit code.

### Exit code based on response content

Use `--exit-on` with a template that produces an integer exit code:

```bash
xwebs connect wss://api.example.com \
  --send '{"health":"check"}' \
  --once \
  --exit-on '{{if eq (.Message | jq ".status") "ok"}}0{{else}}1{{end}}'
```

This makes xwebs usable as a health-check in shell scripts:

```bash
if xwebs connect wss://api.example.com --send '{"health":"check"}' --once \
     --exit-on '{{if eq (.Message | jq ".status") "ok"}}0{{else}}1{{end}}'; then
  echo "Service is healthy"
fi
```

### Execute a script file

A `.xwebs` script file contains REPL commands, one per line:

```bash
# test.xwebs
:send {"type":"subscribe","channel":"trades"}
:wait 500ms
:assert {{eq (.Last | jq ".type") "subscribed"}} "Expected subscription confirmation"
:send {"type":"ping"}
:wait 100ms
:assert {{lt .LastLatencyMs 100}} "Latency should be under 100ms"
:exit
```

Run it:

```bash
xwebs connect wss://api.example.com --script test.xwebs
```

### Watch a file and re-send on change

Useful during development — change `payload.json` and xwebs automatically resends it:

```bash
xwebs connect wss://api.example.com --watch payload.json
```

---

## CI/CD Examples

### GitHub Actions health check

```yaml
- name: WebSocket health check
  run: |
    echo '{"health":"check"}' \
      | xwebs connect ${{ secrets.WS_URL }} --once --timeout 10s \
      | jq -e '.status == "ok"'
```

### Contract testing with assertions

```bash
#!/bin/bash
# ws-contract-test.sh

set -e

xwebs connect wss://api.example.com --script - <<'EOF'
:send {"type":"auth","token":"test-token"}
:expect .type == "auth_ok" --timeout 5s
:send {"type":"subscribe","channel":"prices"}
:expect .type == "subscribed" --timeout 5s
:send {"type":"ping"}
:expect .type == "pong" --timeout 2s
:assert {{lt .LastLatencyMs 500}} "Ping latency must be under 500ms"
:exit
EOF

echo "Contract test passed"
```

### Compare staging vs production

```bash
xwebs diff wss://staging.api.example.com wss://prod.api.example.com \
  --input test-messages.txt \
  --format json
```
