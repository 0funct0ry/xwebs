---
title: "xwebs diff"
description: "Compare responses from two WebSocket endpoints"
generated: "2026-04-23"
---

# xwebs diff

Send the same messages to two WebSocket servers and compare their responses. Useful for verifying API compatibility between versions.

---

## Synopsis

```bash
xwebs diff <url1> <url2> [flags]
```

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--input` | string | — | Messages file (one per line) |
| `--format` | string | text | Output format: text or json |

## Examples

```bash
xwebs diff wss://v1.api.example.com wss://v2.api.example.com --input messages.txt
```

