---
title: Client REPL
description: All interactive REPL commands available in xwebs connect, including keyboard shortcuts and prompt customization.
---

# Client REPL

When `xwebs connect` is run on a TTY (or with `--interactive`), it drops into a readline-based REPL. The REPL gives you tab completion, persistent history, syntax highlighting, and dozens of built-in commands.

## Readline Capabilities

- Full readline editing with **emacs** (default) or **vi** keybindings — set via `--keymap emacs|vi`
- Persistent history stored in `~/.xwebs_history` (configurable size: `--history-size 10000`)
- Reverse history search with `Ctrl+R`
- Multi-line input: use `\` at the end of a line or `Shift+Enter`
- Syntax highlighting of JSON and template expressions as you type

## Tab Completion

The REPL offers context-aware completion for:

- **Builtin commands** — `:send`, `:subscribe`, `:close`, `:ping`, `:status`, etc.
- **Template functions** — `{{now`, `{{env`, `{{uuid`, etc.
- **JSON keys** — learned from received messages, offered as jq-style paths
- **File paths** — when using `:source` or `:load`
- **Connection URLs** — from history and bookmarks
- **Handler names** — from the loaded config

---

## Commands

### Connection Management

```
:connect <url>            Connect to a new WebSocket URL (switches connection)
:disconnect               Disconnect from the current server
:reconnect                Reconnect to the current URL
:status                   Show connection state, RTT, and message counts
:close [code] [reason]    Send close frame with optional code and reason
```

Example — reconnect with a custom close code:
```
xwebs> :close 4000 "session-expired"
```

### Messaging

```
:send <message>           Send a plain text message (bare input does this by default)
:sendj <json>             Validate JSON before sending
:sendb <hex|base64:data>  Send binary data — hex string or base64: prefix
:sendt <template>         Render a Go template then send the result
:ping [payload]           Send a ping frame with optional payload
:pong [payload]           Send a pong frame with optional payload
```

Binary send examples:
```
xwebs> :sendb 48656c6c6f
xwebs> :sendb base64:SGVsbG8=
xwebs> :ping hex:010203
```

Template send:
```
xwebs> :sendt {"id":"{{uuid}}","ts":"{{now | formatTime "RFC3339"}}","user":"{{env "USER"}}"}
```

### Session & Automation

```
:expect <pattern> [--timeout <dur>]   Wait for a message matching pattern (regex or jq)
:log <file> | off         Start/stop logging all traffic to JSONL
:record <file> | off      Start/stop recording session for replay
:replay <file>            Replay a previously recorded session
:mock <file> | off        Load/unload a mock scenario for automated responses
:bench <n> <message>      Benchmark sequential RTT for N messages
:flood <msg> [--rate <n>] Flood the server at a specified rate (msg/s)
:watch                    Enter real-time monitoring mode for connection stats
```

Record a session and replay it later:
```
xwebs> :record session.jsonl
xwebs> :send {"type":"ping"}
xwebs> :record off
xwebs> :replay session.jsonl
```

### Handler Management

```
:handlers                 List all active client-side message handlers
:handler add <flags>      Dynamically add a handler (e.g., -m "PING" -R "PONG")
:handler delete <id>      Remove a handler by name or ID
:handler edit [id]        Edit a handler or full config in $EDITOR
:handler save <file>      Save current handlers to a YAML config file
:load <file>              Load handler config from YAML/JSON file
```

### Variables & Environment

These are client-mode-only and scoped to your REPL session:

```
:set <key> <value>        Set a session variable (available as {{.Vars.<key>}})
:get <key>                Read a session variable
:vars                     List all session variables
:env                      List all process environment variables
```

After `:set env production`, templates can use `{{.Vars.env}}` or `{{.Session.env}}`.

### Output Formatting

```
:format json              Pretty-print JSON messages
:format raw               Show raw frames
:format hex               Hex dump binary frames
:format template <tmpl>   Custom output template per message
:filter <jq-expr>         Only display messages matching filter
:quiet                    Suppress non-message output
:verbose                  Show frame metadata (opcode, length, mask)
:timestamps on|off        Toggle timestamps
:color on|off|auto        Control ANSI color output
```

Filter to only see error messages:
```
xwebs> :filter .type == "error"
```

### Scripting & Utilities

```
:source <file>            Execute commands from a script file
:alias <name> <cmd>       Create a command alias
:wait <duration>          Pause execution (e.g., 1s, 500ms)
:sleep <duration>         Alias for :wait
:assert <expression>      Assert a condition; fails if empty, 0, or false
:write <file> [--flags]   Save content (last message, handlers, etc.) to a file
:history [-n <count>]     View and search command history
:hedit [-n <n>]           Edit and re-run a previous command in $EDITOR
```

### Shell & Filesystem

```
:! <command>              Execute a shell command from the REPL
:shell                    Switch to a full interactive shell session
:pwd / :cd / :ls          Standard filesystem navigation
```

### General

```
:help                     List all available commands
:clear                    Clear the terminal screen
:exit / :quit / Ctrl+D    Disconnect and exit
```

---

## Prompt Customization

The REPL prompt is a Go template. Set it with `:prompt set` and reset it with `:prompt reset`.

**Available variables:**

| Variable | Example value |
|----------|---------------|
| `.Host` | `echo.websocket.org` |
| `.URL` | `wss://echo.websocket.org/chat` |
| `.ConnectionID` | `ns-abc-123` |
| `.Path` | `/ws` |
| `.Scheme` | `wss` |
| `.Vars.<key>` / `.Session.<key>` | set via `:set` |
| `.Env.<KEY>` | any process env var |
| `.LastLatencyMs` | RTT of the last message |

**Color and style functions:**

| Function | Effect |
|----------|--------|
| `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `grey` | ANSI colors |
| `bold` | Bold/bright |
| `dim` / `faint` | Dim text |
| `italic` | Italic (terminal-dependent) |
| `underline` | Underline |
| `inverse` | Swap foreground/background (Powerline blocks) |
| `color "code" "text"` | Custom ANSI code |
| `reset` | Reset all styling |

**System functions available in prompts:**

`cwd`, `hostname`, `env`, `shell`, `pid`, `now`, `nowUnix`, `truncate`, `upper`, `lower`, `trim`, `contains`, `padLeft`, `padRight`

**Prompt examples:**

```bash
# Minimal colored chevron
:prompt set "{{green \">\"}} "

# Host with connection indicator
:prompt set "{{if .Host}}{{green \"●\"}}{{else}}{{red \"○\"}}{{end}} {{.Host}} > "

# Powerline-style block
:prompt set "{{inverse (blue (print \" \" .Host \" \"))}} {{cwd}} > "

# Production warning (red block if hostname contains "prod")
:prompt set "{{if contains \"prod\" .Host}}{{inverse (red (print \" \" .Host \" \"))}}{{else}}{{inverse (green (print \" \" .Host \" \"))}}{{end}} > "

# Latency-aware prompt
:prompt set "{{.Host}} [{{yellow (print .LastLatencyMs \"ms\")}}] > "

# Multi-line with timestamp
:prompt set "{{dim now}}\n{{bold .Host}} > "

# Session variable in prompt (after :set env staging)
:prompt set "[{{.Vars.env}}] {{.Host}} > "
```
