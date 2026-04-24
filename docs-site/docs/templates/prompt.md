---
title: Prompt Customization
description: Customize the xwebs client REPL prompt with Go templates, colors, and styles.
---

# Prompt Customization

The xwebs client REPL prompt is a Go template. Set it at runtime with `:prompt set` or configure it permanently in `~/.xwebs.yaml`.

```
:prompt set "<Go template>"
:prompt reset    # restore the default
```

The template uses the same engine as handlers, with a focused set of connection variables and additional color/style functions.

---

## Available Variables

| Variable | Type | Description | Example value |
|----------|------|-------------|---------------|
| `.Host` | string | Hostname of the connected server | `echo.websocket.org` |
| `.URL` | string | Full WebSocket URL | `wss://echo.websocket.org/chat` |
| `.ConnectionID` | string | Unique session identifier | `ns-abc-123` |
| `.Path` | string | URL path | `/ws` |
| `.Scheme` | string | Protocol scheme | `wss` |
| `.Vars.<key>` | any | Session variables (set via `:set`) | after `:set env prod` |
| `.Session.<key>` | any | Alias for `.Vars` | — |
| `.Env.<KEY>` | string | Process environment variables | `.Env.USER` |
| `.LastLatencyMs` | int64 | RTT of the last message in ms | `24` |

---

## Color Functions

Wrap text in ANSI color escape codes:

| Function | Color |
|----------|-------|
| `black` | Black |
| `red` | Red |
| `green` | Green |
| `yellow` | Yellow |
| `blue` | Blue |
| `magenta` | Magenta |
| `cyan` | Cyan |
| `white` | White |
| `grey` | Grey |
| `color "code" "text"` | Custom ANSI code |

---

## Style Functions

| Function | Effect |
|----------|--------|
| `bold` | Bold / bright text |
| `dim` / `faint` | Dim, subtle text |
| `italic` | Italic (terminal-dependent) |
| `underline` | Underline |
| `inverse` | Swap foreground and background — creates Powerline-style filled blocks |
| `reset` | Reset all styling |

Colors and styles can be composed: `{{bold (green "text")}}` renders bold green text.

---

## System Functions in Prompts

| Function | Returns |
|----------|---------|
| `cwd` | Current working directory |
| `hostname` | Machine hostname |
| `env "KEY"` | Environment variable value |
| `pid` | Process ID |
| `now` | Current time |
| `nowUnix` | Unix timestamp |
| `truncate n str` | String truncated to n chars with ellipsis |
| `upper`, `lower` | Case conversion |
| `trim` | Trim whitespace |
| `contains` | Substring test |
| `padLeft`, `padRight` | String padding |

---

## Prompt Examples

### 1. Minimal — just a colored chevron

```
:prompt set "{{green \">\"}} "
```

Renders as a green `>` followed by a space.

### 2. Host with connection indicator

```
:prompt set "{{if .Host}}{{green \"●\"}}{{else}}{{red \"○\"}}{{end}} {{.Host}} > "
```

Shows a green dot when connected, red when not, followed by the hostname.

### 3. Path-focused prompt

```
:prompt set "{{.Scheme}}://{{.Host}}{{.Path}} > "
```

Renders as `wss://echo.websocket.org/ws > ` — useful when working with multiple paths on the same server.

### 4. Powerline-style filled block

```
:prompt set "{{inverse (blue (print \" \" .Host \" \"))}} {{cwd}} > "
```

Renders the hostname in a blue filled block (inverted colors), followed by the current directory. The `inverse` function swaps foreground/background to simulate Powerline segment styling.

### 5. Production warning (environment-aware)

```
:prompt set "{{if contains \"prod\" .Host}}{{inverse (red (print \" \" .Host \" \"))}}{{else}}{{inverse (green (print \" \" .Host \" \"))}}{{end}} > "
```

Red block for production hosts, green for everything else. This makes it visually obvious when you're connected to production.

### 6. Latency-aware — color codes the RTT

```
:prompt set "{{.Host}} [{{if gt .LastLatencyMs 200}}{{red (print .LastLatencyMs \"ms\")}}{{else if gt .LastLatencyMs 100}}{{yellow (print .LastLatencyMs \"ms\")}}{{else}}{{green (print .LastLatencyMs \"ms\")}}{{end}}] > "
```

Green for fast (under 100ms), yellow for medium (under 200ms), red for slow (200ms or more).

### 7. Multi-line with timestamp

```
:prompt set "{{dim now}}\n{{bold .Host}} > "
```

First line shows the timestamp in dim text; second line shows the host in bold. Creates a two-line prompt.

### 8. Session variable in prompt

After `:set env staging`:

```
:prompt set "[{{.Vars.env}}] {{.Host}} > "
```

Renders as `[staging] api.example.com > `.

### 9. Connection ID truncated

```
:prompt set "{{truncate 12 .ConnectionID}} @ {{.Host}} > "
```

Shows only the first 12 characters of the connection ID.

### 10. Full Powerline-style

```
:prompt set "{{inverse (blue (print \" \" .Env.USER \" \"))}}{{inverse (cyan (print \" \" .Host \" \"))}}{{inverse (dim (print \" \" .Path \" \"))}} > "
```

Three consecutive filled blocks: username, hostname, path — each in a different color, creating a Powerline multi-segment prompt.

---

## Persistent Configuration

Set a default prompt in `~/.xwebs.yaml`:

```yaml
defaults:
  prompt: '{{green ">"}} '
```

Or per-bookmark:

```yaml
bookmarks:
  - name: Production API
    url: wss://prod-api.example.com/ws
    prompt: '{{inverse (red " PROD ")}} > '
```

The per-bookmark prompt overrides the default when you connect to that bookmark.
