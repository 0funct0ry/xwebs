---
title: delay
description: Defer the handler response by a configurable duration before passing through to the next action.
---

# `delay` — Deferred Response

**Mode:** `[*]` (client and server)

Pauses handler execution for a specified duration before the response or subsequent pipeline steps run. The `duration:` field supports Go duration strings (`500ms`, `2s`, `1m`) and template expressions, allowing the wait time to be derived from message content.

Unlike adding `sleep` to a shell command, `delay` is non-blocking at the goroutine level — other concurrent handlers continue to run normally while this one waits.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `duration` | string | Yes | — | How long to wait. Accepts Go duration strings (`500ms`, `1s`, `30s`) and template expressions. |

---

## Template Variables Injected

`delay` does not inject additional template variables. Standard context is available in `duration:` and in any `respond:` template that follows.

---

## Examples

**Fixed 1-second delay before replying:**

```yaml
handlers:
  - name: slow-pong
    match:
      jq: '.type == "ping"'
    builtin: delay
    duration: "1s"
    respond: '{"type": "pong", "delayed_ms": 1000}'
```

**Dynamic delay from message content:**

```yaml
handlers:
  - name: client-controlled-delay
    match:
      jq: '.type == "delayed_echo"'
    builtin: delay
    duration: '{{.Message | jq ".wait_ms"}}ms'
    respond: '{"type": "echoed", "after_ms": {{.Message | jq ".wait_ms"}}}'
```

**Delay in a pipeline before subsequent steps:**

```yaml
handlers:
  - name: rate-shape
    match:
      jq: '.type == "batch"'
    pipeline:
      - builtin: delay
        duration: "200ms"
      - run: process_batch.sh "{{.Message | shellEscape}}"
    respond: '{"processed": true, "stdout": {{.Stdout | toJSON}}}'
```

---

## Edge Cases

- If the template in `duration:` evaluates to an invalid duration string (e.g. `"abc"` or an empty string), the delay is treated as zero and execution continues immediately with a logged warning.
- `delay` participates in the handler-level `timeout:`. If the delay itself exceeds the timeout, the handler is cancelled.
- When used in server mode with many concurrent connections, large numbers of delayed handlers accumulate goroutines. Keep durations short or limit concurrency with `concurrent: false`.
- In client mode, `delay` pauses only that handler's response — outbound messages typed in the REPL are unaffected.
