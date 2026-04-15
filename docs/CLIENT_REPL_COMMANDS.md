# xwebs Client REPL Commands

This document provides a comprehensive list of commands available in the `xwebs connect` interactive REPL.

## Connection Management

| Command                  | Description                                               | Example Usage                     |
|--------------------------|-----------------------------------------------------------|-----------------------------------|
| `:connect <url>`         | Connect to a new WebSocket URL.                           | `:connect ws://localhost:8080/ws` |
| `:reconnect`             | Reconnect to the current URL.                             | `:reconnect`                      |
| `:disconnect`            | Disconnect from the current server.                       | `:disconnect`                     |
| `:close [code] [reason]` | Close the connection with an optional code and reason.    | `:close 1000 "Goodbye"`           |
| `:status`                | Show detailed connection status, RTT, and message counts. | `:status`                         |

---

## Messaging Commands

| Command                     | Description                              | Example Usage                         |
|-----------------------------|------------------------------------------|---------------------------------------|
| `:send <message>`           | Send a plain text message.               | `:send Hello server!`                 |
| `:sendj <json>`             | Send a validated JSON message.           | `:sendj {"type":"ping","id":123}`     |
| `:sendb <hex\|base64:data>` | Send binary data (hex or base64).        | `:sendb base64:SGVsbG8=`              |
| `:sendt <template>`         | Send a rendered Go template.             | `:sendt {"user":"{{.Session.user}}"}` |
| `:ping [payload]`           | Send a ping frame with optional payload. | `:ping "checking-alive"`              |
| `:pong [payload]`           | Send a pong frame with optional payload. | `:pong "checking-alive"`              |

---

## Session Tools & Automation

| Command                          | Description                                           | Example Usage                |
|----------------------------------|-------------------------------------------------------|------------------------------|
| `:expect <pattern> [--timeout] ` | Wait for a message matching a pattern (Regex or JQ).  | `:expect .type == "auth_ok"` |
| `:replay <file>`                 | Replay a previously recorded session.                 | `:replay session_log.json`   |
| `:record <file> [\| off]`        | Record current session messages to a file.            | `:record my_test_case.json`  |
| `:log <file> [\| off]`           | Log all traffic (including metadata) to a file.       | `:log traffic.log`           |
| `:mock <file> [\| off]`          | Load a mock scenario for automated responses.         | `:mock mocks/api.yaml`       |
| `:bench <n> <message>`           | Benchmark sequential RTT latency for N messages.      | `:bench 100 ping`            |
| `:flood <msg> [--rate <n>]`      | Flood the server with messages at a specific rate.    | `:flood "spam" --rate 50`    |
| `:watch`                         | Enter real-time monitoring mode for connection stats. | `:watch`                     |

---

## Handler Management
Used for client-side automation and reactive messaging.
 
| Command                       | Description                                                | Example Usage                       |
|-------------------------------|------------------------------------------------------------|-------------------------------------|
| `:handlers`                   | List all active client-side message handlers.              | `:handlers`                         |
| `:handler add <flags>`        | Add a handler to auto-respond to server messages.          | `:handler add -m "PING" -R "PONG"`  |
| `:handler delete <id>`        | Remove a handler by its name or ID.                        | `:handler delete auto-responder`    |
| `:handler edit [id]`          | Edit a handler or the full configuration in `$EDITOR`.     | `:handler edit auto-responder`      |
| `:handler save <file>`        | Save current handlers to a YAML config file.               | `:handler save my-handlers.yaml`    |
 
---

## General & Utility Commands

| Command                | Description                                          | Example Usage                       |
|------------------------|------------------------------------------------------|-------------------------------------|
| `:help`                | List all available commands.                         | `:help`                             |
| `:exit` / `:quit`      | Disconnect and exit the REPL.                        | `:exit`                             |
| `:clear`               | Clear the terminal screen.                           | `:clear`                            |
| `:! <command>`         | Execute a shell command from the REPL.               | `:! ls -la`                         |
| `:shell`               | Switch to a full interactive shell session.          | `:shell`                            |
| `:set <key> <val>`     | Set a session variable for templates.                | `:set user "alice"`                 |
| `:get <key>`           | Display the value of a session variable.             | `:get user`                         |
| `:vars`                | List all active session variables.                   | `:vars`                             |
| `:pwd` / `:cd` / `:ls` | Standard filesystem navigation commands.             | `:cd ~/logs`                        |
| `:history`             | View command history with searching and filtering.   | `:history -n 10`                    |
| `:hedit`               | Edit and re-run a previous command in `$EDITOR`.     | `:hedit -n 5`                       |
| `:format <mode>`       | Set display mode (`json`, `raw`, `hex`, `template`). | `:format json`                      |
| `:filter <expr>`       | Set a display filter (JQ or Regex).                  | `:filter .payload.id == 1`          |
| `:timestamps`          | Toggle ISO 8601 message timestamps.                  | `:timestamps on`                    |
| `:source <file>`       | Execute commands from a script file.                 | `:source scripts/auth.xwebs`        |
| `:wait <duration>`     | Pause execution (e.g., `1s`, `500ms`).               | `:wait 1.5s`                        |
| `:assert <expr>`       | Verify a condition (fails if empty, 0, or false).    | `:assert "{{eq .Last \"pong\"}}"`   |
| `:write <file>`        | Save content (last msg, handlers, etc.) to a file.   | `:write --last-message result.json` |
