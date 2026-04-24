---
title: Examples
description: Complete working examples demonstrating xwebs features — from a live shell terminal to CI/CD webhook bridges.
---

# Examples

Each example is a self-contained, runnable application built with xwebs. Clone the repository and run any of them immediately.

## Quick Start

```bash
git clone https://github.com/0funct0ry/xwebs
cd xwebs
```

Then follow the launch instructions for each example below.

---

## Example Overview

| # | App | Features | Launch |
|---|-----|----------|--------|
| 01 | [Live Shell Terminal](./live-shell-terminal.md) | `run:`, `.Stdout`/`.Stderr`/`.ExitCode`, `concurrent: false` | `xwebs serve --handlers examples/apps/01-live-shell-terminal/terminal.yaml` |
| — | [Lua Examples](./lua.md) | `builtin: lua`, Lua API, pattern reference | Various |

---

## 01 — Live Shell Terminal

A browser-based terminal that executes shell commands on the xwebs server and streams output back in real time. Built with vanilla JS + Tailwind CDN — no build step required.

**What it demonstrates:**
- `run:` executing shell commands from WebSocket messages
- `.Stdout`, `.Stderr`, `.ExitCode` in response templates
- `concurrent: false` to serialize command execution
- `timeout:` to kill runaway commands
- Single-file HTML frontend with auto-reconnect

**Launch:**
```bash
# Start the server
xwebs serve --handlers examples/apps/01-live-shell-terminal/terminal.yaml

# Open the UI
open examples/apps/01-live-shell-terminal/index.html
```

Then type any shell command in the browser terminal.

---

## Lua Examples

Demonstrates embedded Lua scripting across multiple patterns: simple response logic, stateful per-connection tracking, JSON transformation, and KV store integration.

See [Lua Examples](./lua.md) for the full pattern reference.
