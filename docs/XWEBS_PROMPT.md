# xwebs Dynamic Prompting Guide

The `:prompt set` command allows you to customize your REPL environment using Go templates. This guide details the available variables, functions, and styling techniques to create beautiful, informative, and functional prompts.

## Variables Reference

The following variables are available in the template context:

| Variable | Description | Example |
| :--- | :--- | :--- |
| `.Host` | Hostname of the connected server | `echo.websocket.org` |
| `.URL` | Full WebSocket URL | `wss://echo.websocket.org/chat` |
| `.ConnectionID` | Unique ID of the current session | `ns-abc-123` |
| `.Path` | The path part of the URL | `/ws` |
| `.Scheme` | The protocol scheme | `wss` |
| `.Vars` | Global session variables set via `:set` | `{{.Vars.user}}` |
| `.Env` | Map of all environment variables | `{{.Env.PATH}}` |
| `.Session` | Alias for `.Vars` | `{{.Session.project}}` |

---

## Functions Reference

### 1. Colors & Styles
Enhance visibility with terminal colors and text styles. All functions wrap the input string with ANSI escape codes.

| Function | Type | Result |
| :--- | :--- | :--- |
| `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `grey` | Color | Changes text color |
| `bold` | Style | Makes text thick/bright |
| `dim`, `faint` | Style | Makes text subtle |
| `italic` | Style | Slanted text (terminal dependent) |
| `underline` | Style | Underlined text |
| `inverse` | Style | Swaps background and foreground |
| `color "code" "text"` | Custom | Uses a specific ANSI color code |
| `reset` | Utility | Manually resets all styling |

### 2. System Intelligence
Functions that pull live data from your operating system.

| Function | Description |
| :--- | :--- |
| `cwd` | Returns the absolute Current Working Directory. |
| `hostname`| Returns the system's hostname. |
| `env "KEY"` | Gets the value of an environment variable. |
| `shell "cmd"`| Runs a shell command and returns the trimmed output. |
| `pid` | Returns the Process ID of the xwebs instance. |

### 3. String Manipulation
Transform and clean up your prompt data.

| Function | Description | Example |
| :--- | :--- | :--- |
| `upper` / `lower`| Case conversion | `{{upper .Host}}` |
| `truncate <n>` | Shortens string with `...` | `{{truncate 10 cwd}}` |
| `trim` | Removes surrounding whitespace | `{{trim .Host}}` |
| `contains "p"` | Checks for presence | `{{if contains "prod" .Host}}...{{end}}` |
| `padLeft` / `padRight`| Add spacing | `{{padRight 10 "text"}}` |

### 4. Time
| Function | Description | Example |
| :--- | :--- | :--- |
| `now` | Current local time in standard format | `11:45:00` |
| `nowUnix` | Current Unix timestamp | `1712574000` |

---

## Design Principles

1.  **Use `inverse` for blocks**: To create a "Powerline" look, use `inverse` on a colored string.
2.  **Concatenate with expressions**: Standard template concatenation works by placing expressions side-by-side.
3.  **Hierarchy**: Put high-priority info (like the host or current path) in bold or distinct colors.

---

## 30 Prompt Examples

### Minimalist & Clean
1.  **Simple Arrow**: `:prompt set "> "`
2.  **Colored Arrow**: `:prompt set "{{green \">\"}} "`
3.  **Bold Host**: `:prompt set "{{bold .Host}} > "`
4.  **Grey Path**: `:prompt set "{{grey (cwd)}} $ "`
5.  **Dotted Separator**: `:prompt set "{{cyan .Host}} ··· "`

### Path & System Focused
6.  **Full Path Bold**: `:prompt set "{{bold (cwd)}} > "`
7.  **Truncated Path**: `:prompt set "{{truncate 20 (cwd)}} > "`
8.  **Hostname & User**: `:prompt set "{{env \"USER\"}}@{{hostname}} > "`
9.  **PID identification**: `:prompt set "[{{pid}}] {{cwd}} > "`
10. **OS Indicator**: `:prompt set "{{bold (shell \"uname\")}} {{cwd}} > "`

### Connection Aware
11. **URL Scheme Indicator**: `:prompt set "{{if eq .Scheme \"wss\"}}{{green \"✔\"}}{{else}}{{yellow \"!\"}}{{end}} {{.Host}} > "`
12. **Connection ID Tag**: `:prompt set "id:{{cyan .ConnectionID}} > "`
13. **Full URL (Detailed)**: `:prompt set "{{dim .URL}} \n> "`
14. **Host with Path**: `:prompt set "{{blue .Host}}{{if .Path}}{{grey .Path}}{{end}} > "`
15. **Status Dot**: `:prompt set "{{if .Host}}{{green \"●\"}}{{else}}{{red \"○\"}}{{end}} {{.Host}} > "`

### Stylized & "Powerline" Blocks
16. **Blue Inverse Host**: `:prompt set "{{inverse (blue (print \" \" .Host \" \"))}} {{cwd}} > "`
17. **Yellow Warning Host**: `:prompt set "{{if contains \"prod\" .Host}}{{inverse (red (print \" \" .Host \" \"))}}{{else}}{{inverse (green (print \" \" .Host \" \"))}}{{end}} > "`
18. **Multi-Block Path**: `:prompt set "{{inverse (magenta (print \" \" hostname \" \"))}}{{inverse (grey (print \" \" (cwd) \" \"))}} $ "`
19. **Cyan Segment**: `:prompt set "{{inverse (cyan (print \" xwebs \"))}} {{bold .Host}} > "`
20. **Dark Mode Subtle**: `:prompt set "{{inverse (grey (print \" \" .ConnectionID \" \"))}} "`

### Dynamic & Contextual
21. **User Defined Variable**: `:prompt set "[{{.Vars.env}}] {{.Host}} > "` (After running `:set env prod`)
22. **Environment Variable**: `:prompt set "{{env \"PROJECT_NAME\"}} > "`
24. **Multi-line Info**: `:prompt set "{{grey now}} \n{{bold .Host}} > "`
25. **Compact Mode**: `:prompt set "{{if gt (len .Host) 15}}{{truncate 15 .Host}}{{else}}{{.Host}}{{end}} > "`

### Advanced & Creative
26. **Time Stamped**: `:prompt set "[{{dim now}}] {{green \">\"}} "`
27. **Random Emoji (Static)**: `:prompt set "⚡ {{bold .Host}} > "`
28. **Git Status Lite**: `:prompt set "{{cwd}} {{if shell \"git status --porcelain 2>/dev/null\"}}{{red \"*\"}}{{else}}{{green \"-\"}}{{end}} > "`
29. **Latency Aware**: `:prompt set "{{.Host}} [{{yellow (print .LastLatencyMs \"ms\")}} ] > "`
30. **Rainbow Start**: `:prompt set "{{red \"x\"}}{{yellow \"w\"}}{{green \"e\"}}{{cyan \"b\"}}{{blue \"s\"}} > "`


