# Mock Scenario Examples

Example YAML files for the xwebs `:mock` command and `xwebs mock --scenario` mode. Each file simulates a different server behavior pattern.

## How `:mock` Works

The mock watches **incoming messages from the server**. When an incoming message matches an `expect` step, the mock sends the `respond` message back to the server. Steps with `after` inject server-push-style messages on a timer.

This means **the server must actually send something** for `expect` steps to trigger. The simplest setup is an **echo server** — you send a message, the server echoes it back, the mock matches the echo and responds.

```
Client                    Echo Server              Mock (local)
  │                           │                        │
  │── send {"type":"ping"} ──▶│                        │
  │                           │                        │
  │◀── echo {"type":"ping"} ──│                        │
  │                           │    match .type=="ping"  │
  │                           │◀── respond {"type":"pong"} ──│
  │◀── {"type":"pong"} ──────────────────────────────────────│
```

## Quick Start

**Option A — REPL mock with a local echo server:**

```bash
# Start a local echo server with websocat
websocat -t ws-l:127.0.0.1:8080 mirror: &

# Connect and load a mock
xwebs connect ws://localhost:8080
:mock examples/mock/04-subscribe-events.yaml
:send {"type":"subscribe","channel":"trades"}
# The echo server echoes your message back → mock matches it → sends subscription confirmation
# Then timed server-push events arrive at 1s, 2s, 3s
```

**Option B — REPL mock against a public echo server:**

```bash
xwebs connect wss://echo.websocket.org
:mock examples/mock/01-echo.yaml
:send {"msg":"hello"}
# echo.websocket.org echoes it back → mock matches → sends wrapped response
```

**Option C — Standalone mock server (no echo server needed):**

```bash
# xwebs itself IS the server — no external process required
xwebs mock --port 8080 --scenario examples/mock/02-auth-flow.yaml

# In another terminal:
xwebs connect ws://localhost:8080
:send {"type":"auth","token":"secret-123"}
```

> **Note:** In options A and B, you'll see **both** the echo server's reply and the mock's response — that's expected. The echo is the trigger; the mock response is the simulated behavior. If the double message is distracting, use Option C instead.

## Setting Up a Local Echo Server

These mock examples rely on an echo server to bounce messages back and trigger `expect` matches. Here's how to set one up with `websocat`:

```bash
# Install (macOS)
brew install websocat

# Install (Linux)
cargo install websocat
# or download a binary from https://github.com/vi/websocat/releases

# Start an echo server (mirrors all messages back to the sender)
websocat -t ws-l:127.0.0.1:8080 mirror:
```

Then connect with `xwebs connect ws://localhost:8080`.

**Important:** `websocat -s 8080` is NOT an echo server — it reads from stdin and sends to the client. For mock testing you need `mirror:` so that your sent messages come back and trigger the mock's `expect` steps.

## Examples

| # | File | Pattern | What It Demonstrates |
|---|------|---------|----------------------|
| 01 | `01-echo.yaml` | Wildcard match | Catch-all handler that wraps any message in a response envelope |
| 02 | `02-auth-flow.yaml` | Conditional branching | Two scenarios: reject invalid tokens, accept valid ones |
| 03 | `03-ping-pong.yaml` | Single request/response | Application-level ping with dynamic server metadata in the response |
| 04 | `04-subscribe-events.yaml` | Request + server push | Subscription confirmation followed by timed server-initiated events |
| 05 | `05-crud-resource.yaml` | Ordered multi-step | Full create/read/update/delete lifecycle consumed in sequence |
| 06 | `06-error-codes.yaml` | Multiple independent scenarios | Each scenario matches a different error type (403, 404, 429, 500) |
| 07 | `07-chat-room.yaml` | Mixed expect + push | Room join, member list, server-pushed user join, message ack, and reply |
| 08 | `08-slow-pipeline.yaml` | Long-running with progress | Job submission → 4 progress updates → completion. Tests timeout handling |
| 09 | `09-graphql-ws.yaml` | Protocol simulation | graphql-ws handshake (connection_init/ack), subscription, 3 data events, complete |
| 10 | `10-health-check.yaml` | State transitions | Health check that starts ok, degrades, then recovers. Good for CI assertion scripts |

## Scenario File Format

```yaml
scenarios:
  - name: scenario-name      # Human-readable label
    steps:
      # Wait for a matching INCOMING message, then respond
      - expect:
            jq: '.type == "ping"'       # jq, regex, match (glob), or json_path
        respond: '{"type":"pong"}'       # Go template — has access to .Message, {{now}}, {{uuid}}, etc.
        delay: 100ms                     # Optional delay before sending response

      # Send a message on a timer (no expect — server-initiated push)
      - after: 2s
        send: '{"type":"event","data":"pushed"}'
```

**Key rules:**

- **Trigger model:** `expect` steps match against messages received FROM the server (or echo). They do not match outgoing messages you type.
- `expect` steps are consumed in order. Each waits for the next matching incoming message.
- `after` steps fire on a timer relative to the previous step completing.
- `respond` values are Go templates with access to `.Message` (the matched incoming message) and all builtin template functions (`{{uuid}}`, `{{now}}`, `{{.Message | jq ".field"}}`, etc.).
- Once all steps in a scenario are consumed, the scenario is exhausted. Unmatched messages pass through.
- Multiple scenarios in one file are evaluated independently (a message can match the first unconsumed `expect` across any scenario).

## Which Setup to Use

| Setup | How `expect` triggers | Best for |
|-------|----------------------|----------|
| `websocat mirror:` | Your message echoes back → mock matches the echo | Quick REPL testing, exploring scenarios interactively |
| `wss://echo.websocket.org` | Same as above (public echo server) | No local setup needed, demos |
| `xwebs mock --scenario` | xwebs IS the server — incoming client messages trigger `expect` directly | CI/CD, integration tests, clean mock-only testing |
| Real server | Server's actual replies trigger `expect` | Augmenting a real server with extra mock behaviors |

## Using Mocks in Test Scripts

Combine `:mock` with `:assert` (Story 04.12) for automated testing:

```bash
# test-health.xwebs
:mock examples/mock/10-health-check.yaml
:send {"type":"health"}
:wait 500ms
:assert {{eq (.Last | jq ".status") "ok"}} "First health check should be ok"
:send {"type":"health"}
:wait 4s
:assert {{eq (.Last | jq ".status") "degraded"}} "Second check should show degradation"
:send {"type":"health"}
:wait 500ms
:assert {{eq (.Last | jq ".status") "ok"}} "Third check should recover"
```

```bash
# Requires an echo server (so :send triggers the mock via echo)
websocat -t ws-l:127.0.0.1:8080 mirror: &
xwebs connect ws://localhost:8080 --script test-health.xwebs
```
