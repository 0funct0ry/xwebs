# 01 · Live Shell Terminal

A browser-based terminal that executes shell commands on the xwebs server and renders stdout/stderr/exit results in real time.

## Start

```bash
xwebs serve --port 8080 --handlers terminal.yaml
```

Then open `index.html` in your browser.

## xwebs Features Demonstrated

- `run:` handler — executes arbitrary shell commands
- `.Stdout` / `.Stderr` / `.ExitCode` in `respond:` template
- `concurrent: false` — serialises commands (no parallel execution)
- `timeout: 30s` — kills runaway commands

## File Layout

```
01-live-shell-terminal/
├── index.html      ← single-file frontend (Vanilla JS + Tailwind CDN)
├── terminal.yaml   ← xwebs handler config
└── README.md
```

## UI Features

- Terminal-style dark theme with monospace font
- `stdout` rendered in white, `stderr` in red, non-zero exit highlighted in amber
- Command history (↑ / ↓ arrow keys, up to 200 entries)
- `clear` and `help` built-in client-side commands
- Ctrl+L to clear, Ctrl+C to cancel pending request
- Auto-reconnect on disconnect
- Configurable server URL at the top of the page
