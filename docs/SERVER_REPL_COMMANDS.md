# xwebs Server REPL Commands

This document provides a comprehensive list of commands available in the `xwebs serve` interactive REPL.

## Server Management Commands

| Command                      | Description                                                               | Example Usage                             |
|------------------------------|---------------------------------------------------------------------------|-------------------------------------------|
| `:status`                    | Show server status, uptime, and connected client count.                   | `:status`                                 |
| `:clients`                   | List all active client connections with IDs and remote addresses.         | `:clients`                                |
| `:client <id>`               | Show detailed information (uptime, message counts) for a specific client. | `:client clever_dog`                      |
| `:send [flags] <id> <msg>`   | Send a message to a specific client.                                      | `:send clever_dog {"type":"ping"}`        |
| `:broadcast [flags] <msg>`   | Broadcast a message to all connected clients.                             | `:broadcast "System update in 5 minutes"` |
| `:kick <id> [code] [reason]` | Disconnect a client with an optional close code and reason.               | `:kick clever_dog 1000 "Maintenance"`     |

### Flags for `:send` and `:broadcast`
- `-j, --json`: Validate and send as JSON.
- `-t, --template`: Render the message as a Go template before sending.
- `-b, --binary`: Send as binary (supports `base64:` prefix or hex strings).

---

## Handler Configuration

| Command                       | Description                                                | Example Usage                       |
|-------------------------------|------------------------------------------------------------|-------------------------------------|
| `:handlers`                   | List all loaded handlers and their performance statistics. | `:handlers`                         |
| `:handler add <flags>`        | Dynamically add a new message handler.                     | `:handler add -m "ping" -R "pong"`  |
| `:handler delete <name>`      | Remove a handler by name.                                  | `:handler delete my-handler`        |
| `:handler rename <old> <new>` | Rename an existing handler.                                | `:handler rename old-name new-name` |
| `:handler edit [name]`        | Edit a specific handler or full config in `$EDITOR`.       | `:handler edit my-handler`          |
| `:handler save [file]`        | Persist in-memory handlers to a YAML file.                 | `:handler save backup.yaml`         |
| `:handler <name>`             | Show detailed metrics and actions for a specific handler.  | `:handler my-handler`               |
| `:reload`                     | Hot-reload handler configuration from disk.                | `:reload`                           |
| `:enable <name>`              | Enable a previously disabled handler.                      | `:enable my-handler`                |
| `:disable <name>`             | Disable a handler at runtime.                              | `:disable my-handler`               |

---

## Pub/Sub & Topics

| Command                     | Description                                             | Example Usage                             |
|-----------------------------|---------------------------------------------------------|-------------------------------------------|
| `:topics`                   | List all active pub/sub topics and subscriber counts.   | `:topics`                                 |
| `:topic <name>`             | Inspect detailed subscriber state for a specific topic. | `:topic updates`                          |
| `:publish <topic> <msg>`    | Broadcast a message to a specific pub/sub topic.        | `:publish updates "New version released"` |
| `:subscribe <id> <topic>`   | Manually subscribe a connected client to a topic.       | `:subscribe clever_dog updates`           |
| `:unsubscribe <id> <topic>` | Remove a client from a specific topic.                  | `:unsubscribe clever_dog updates`         |
| `:unsubscribe <id> --all`   | Remove a client from every subscribed topic.            | `:unsubscribe clever_dog --all`           |

---

## General & Utility Commands

| Command                | Description                                          | Example Usage                       |
|------------------------|------------------------------------------------------|-------------------------------------|
| `:help`                | List all available commands.                         | `:help`                             |
| `:exit` / `:quit`      | Shut down the server and exit the REPL.              | `:exit`                             |
| `:clear`               | Clear the terminal screen.                           | `:clear`                            |
| `:! <command>`         | Execute a shell command from the REPL.               | `:! ls -la`                         |
| `:shell`               | Switch to a full interactive shell session.          | `:shell`                            |
| `:set <key> <val>`     | Set a session variable for templates.                | `:set env production`               |
| `:get <key>`           | Display the value of a session variable.             | `:get env`                          |
| `:vars`                | List all active session variables.                   | `:vars`                             |
| `:pwd` / `:cd` / `:ls` | Standard filesystem navigation commands.             | `:cd /path/to/project`              |
| `:history`             | View command history with searching and filtering.   | `:history -n 10`                    |
| `:hedit`               | Edit and re-run a previous command in `$EDITOR`.     | `:hedit -n 5`                       |
| `:format <mode>`       | Set display mode (`json`, `raw`, `hex`, `template`). | `:format json`                      |
| `:filter <expr>`       | Set a display filter (JQ or Regex).                  | `:filter .payload.status`           |
| `:source <file>`       | Execute commands from a script file.                 | `:source scripts/init.xwebs`        |
| `:wait <duration>`     | Pause execution (e.g., `1s`, `500ms`).               | `:wait 2s`                          |
| `:assert <expr>`       | Verify a condition (fails if empty, 0, or false).    | `:assert "{{eq .Last \"pong\"}}"`   |
| `:write <file>`        | Save content (last msg, handlers, etc.) to a file.   | `:write --last-message result.json` |
