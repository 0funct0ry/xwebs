---
title: debounce
description: Suppress repeated messages within a time window and pass only the final one after the window expires.
---

# `debounce` — Debounce Messages

**Mode:** `[*]` (client and server)

Holds incoming messages and waits. If no new message arrives within `window:`, the most recent message is passed through to `respond:` or subsequent pipeline steps. If another message arrives before the window expires, the timer resets and the previous message is discarded.

This mirrors the classic UI debounce pattern — useful for suppressing bursts of noisy events (e.g. keystroke streams, sensor bursts, rapid state changes) and reacting only to the settled value.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `window` | string | Yes | — | Quiet period to wait after the last message before passing it through. Go duration string (`200ms`, `1s`). |

---

## Template Variables Injected

None. The final message content is available as `{{.Message}}` in the `respond:` template when the window expires.

---

## Examples

**Debounce a search input stream — only process after 300 ms of silence:**

```yaml
handlers:
  - name: search-debounce
    match:
      jq: '.type == "search"'
    builtin: debounce
    window: "300ms"
    respond: '{"type": "results", "query": "{{.Message | jq ".q"}}"}'
```

**Suppress rapid config updates, apply only the last one:**

```yaml
handlers:
  - name: config-debounce
    match:
      jq: '.type == "config_update"'
    builtin: debounce
    window: "2s"
    run: apply_config.sh "{{.Message | shellEscape}}"
    respond: '{"applied": true}'
```

**Debounce per-connection in server mode:**

```yaml
handlers:
  - name: typing-indicator
    match:
      jq: '.type == "typing"'
    builtin: debounce
    window: "500ms"
    respond: '{"type": "stopped_typing", "user": "{{.ConnectionID}}"}'
```

---

## Edge Cases

- In server mode, the debounce window is per-handler instance and shared across all clients by default. Rapid messages from different clients share the same timer — for independent per-client debouncing, set `concurrent: false` and rely on handler isolation, or use per-client KV timestamps.
- While messages are held in the window, the client receives no response. Plan timeouts accordingly.
- If the server restarts mid-window, the held message is lost.
- Debounce interacts with handler-level `timeout:`. If `timeout:` is shorter than `window:`, the handler will be cancelled before the debounce fires. Set `timeout:` larger than `window:`.
