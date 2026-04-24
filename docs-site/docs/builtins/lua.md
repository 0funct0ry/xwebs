---
title: lua
description: Run embedded Lua scripts from a handler for complex logic that doesn't fit shell commands or templates.
---

# `lua` — Embedded Lua Scripting

**Mode:** `[*]` (client and server)

Executes an embedded Lua script within the handler. Lua scripts have access to the message, connection context, and template functions. The script's return value (or anything written to `ws.respond`) is sent as the response.

xwebs maintains a Lua VM pool to amortize startup cost — scripts run in isolated states with no shared global state between calls.

---

## Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `script` | string | No* | — | Inline Lua script (short scripts). |
| `file` | string | No* | — | Path to a `.lua` file (recommended for long scripts). |

*One of `script` or `file` is required.

---

## Lua API

Inside the script, xwebs exposes:

| Global | Type | Description |
|--------|------|-------------|
| `ws.message` | string | Raw incoming message |
| `ws.connection_id` | string | Unique connection identifier |
| `ws.host` | string | Connected host |
| `ws.path` | string | URL path |
| `ws.vars` | table | Session variables (client mode) |
| `ws.kv(key)` | function | Read from KV store (server mode) |
| `ws.respond(msg)` | function | Send response to client |
| `ws.log(msg)` | function | Log a message |
| `json.decode(str)` | function | Parse JSON string to Lua table |
| `json.encode(val)` | function | Encode Lua table to JSON string |

---

## Examples

**Inline script — complex routing logic:**

```yaml
handlers:
  - name: lua-router
    match: "*"
    builtin: lua
    script: |
      local msg = json.decode(ws.message)
      if not msg then
        ws.respond('{"error":"invalid_json"}')
        return
      end
      
      if msg.type == "ping" then
        ws.respond('{"type":"pong","ts":' .. os.time() .. '}')
      elseif msg.type == "echo" then
        ws.respond(ws.message)
      else
        ws.respond('{"error":"unknown_type","received":"' .. (msg.type or "nil") .. '"}')
      end
```

**File-based script:**

```yaml
handlers:
  - name: complex-handler
    match:
      jq: '.type == "complex"'
    builtin: lua
    file: scripts/complex_handler.lua
    respond: '{"result":"{{.LuaResult}}"}'
```

**Lua script with KV store access (server mode):**

```yaml
handlers:
  - name: kv-lua
    match:
      jq: '.type == "get_kv"'
    builtin: lua
    script: |
      local msg = json.decode(ws.message)
      local value = ws.kv(msg.key)
      ws.respond(json.encode({
        key = msg.key,
        value = value,
        found = value ~= ""
      }))
```

---

## Full Script Example (`scripts/rate_counter.lua`)

```lua
-- Count messages per connection and rate-limit at 100/min
local msg = json.decode(ws.message)
if not msg then return end

local key = "counter:" .. ws.connection_id
local count = tonumber(ws.kv(key)) or 0

if count >= 100 then
  ws.respond(json.encode({
    error = "rate_limited",
    count = count,
    limit = 100
  }))
  return
end

-- Increment counter (TTL handled outside in kv-set)
ws.respond(json.encode({
  ok = true,
  count = count + 1,
  message = msg
}))
```

---

## Edge Cases

- Scripts have a default 30-second execution timeout. Override with `timeout:` on the handler.
- Lua `print()` output goes to the xwebs log (not to the WebSocket response). Use `ws.respond()` to send responses.
- Scripts share no global state between calls. Each execution starts with a clean Lua state (from the pool).
- The `os.execute()` and `io.open()` functions are sandboxed — file I/O and shell execution are restricted. Use the `shell` builtin or `run:` for shell access.
- `file:` scripts are read once at startup and cached. Use `:reload` to pick up changes during development.
