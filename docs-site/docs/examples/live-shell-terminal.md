---
title: Live Shell Terminal
description: A browser-based terminal that executes shell commands on the xwebs server and streams results in real time.
---

# Live Shell Terminal

A single-file browser terminal backed by xwebs. Type any shell command in the browser — it runs on the server and stdout/stderr stream back immediately. No build step, no npm, no framework: just a 500-line HTML file with vanilla JS and Tailwind CDN.

**Source:** [`examples/apps/01-live-shell-terminal/`](https://github.com/0funct0ry/xwebs/tree/main/examples/apps/01-live-shell-terminal/)

---

## Features

- Execute any shell command from the browser
- Real-time stdout / stderr rendering with color coding
- Exit code display
- Command history (↑ / ↓ keys)
- `clear` and `help` built-in commands
- `Ctrl+L` to clear, `Ctrl+C` to signal the process
- Auto-reconnect on disconnect
- Dark theme, configurable server URL

---

## Launch

```bash
# 1. Start the xwebs server
xwebs serve --handlers examples/apps/01-live-shell-terminal/terminal.yaml

# 2. Open the frontend in your browser
open examples/apps/01-live-shell-terminal/index.html
```

The server listens on `ws://localhost:8080` by default. The HTML file connects to it automatically.

---

## `handlers.yaml`

```yaml
handlers:
  - name: run-command
    match: "*"
    concurrent: false
    timeout: 30s
    run: "{{.Message}}"
    respond: |
      {"stdout":"{{.Stdout | js}}","stderr":"{{.Stderr | js}}","exit":{{.ExitCode}},"cmd":"{{.Message | js}}"}
```

The handler is deliberately minimal: every message is treated as a shell command, executed via `sh -c`, and the result is sent back as JSON. `concurrent: false` ensures commands run one at a time per connection, preventing interleaved output. `timeout: 30s` kills runaway commands.

The `js` template function (alias for `shellEscape`) escapes the output for safe embedding in a JSON string.

---

## Protocol Table

| Direction | Message | Format |
|-----------|---------|--------|
| Browser → Server | Shell command text | Plain string, e.g. `ls -la` |
| Server → Browser | Command result | JSON: `{"stdout":"...","stderr":"...","exit":0,"cmd":"ls -la"}` |

---

## Frontend Notes

The entire frontend is a single `index.html` file — no build step required. Key implementation details:

- WebSocket connection is established on page load with automatic reconnect using exponential backoff.
- Each command is sent as a plain string. The server treats the entire message as a shell command.
- Responses are parsed as JSON. Stdout is rendered in white, stderr in red/orange, exit codes other than 0 in yellow.
- The UI is built with Tailwind CDN loaded from a `<script>` tag — no local dependencies.
- Command history is stored in `localStorage` and persists across page reloads.

---

## Security Warning

This example runs arbitrary shell commands from a browser. **Do not expose it on a public network.** For production use:

- Add authentication (JWT, session tokens, IP allowlist)
- Use `--sandbox` and `--allowlist` to restrict which commands can run
- Run xwebs as a non-root user
- Consider using `--allowed-origins` to restrict WebSocket connections

```bash
# Safer: restrict to specific commands only
xwebs serve --handlers terminal.yaml --sandbox --allowlist "ls,cat,echo,pwd"
```
