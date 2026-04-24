---
title: Lua Examples
description: Embedded Lua scripting patterns in xwebs — from simple response logic to stateful per-connection tracking.
---

# Lua Examples

The `lua` builtin lets you write complex handler logic in Lua rather than shell commands. Lua scripts run inside the xwebs process, have access to the full message context and KV store, and are pooled for performance.

**Source:** [`examples/lua/`](https://github.com/0funct0ry/xwebs/tree/main/examples/lua/)

---

## When to Use Lua

Use Lua when:
- Your logic doesn't fit a single shell pipeline
- You need conditional branching with multiple possible responses
- You want stateful per-connection tracking without external storage
- You're doing complex JSON transformations that are awkward in jq
- You need a tight loop over data without spawning multiple shell processes

Use shell (`run:`) when:
- You need to call existing CLI tools (`psql`, `curl`, `ffmpeg`, etc.)
- The operation is single-step
- You want cross-platform compatibility via the system shell

---

## Pattern Reference

### Pattern 1: Simple Request/Response

The most basic pattern — parse JSON, branch on type, respond:

```lua
-- scripts/router.lua
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
  ws.respond(json.encode({error = "unknown_type", type = msg.type}))
end
```

```yaml
handlers:
  - name: lua-router
    match: "*"
    builtin: lua
    file: scripts/router.lua
```

### Pattern 2: JSON Transformation

Reshape an incoming message without a jq one-liner:

```lua
-- scripts/transform.lua
local msg = json.decode(ws.message)
if not msg or not msg.data then
  ws.respond('{"error":"missing_data"}')
  return
end

-- Normalize field names, add metadata
local out = {
  id = msg.data.ID or msg.data.id,
  name = (msg.data.name or ""):lower(),
  tags = msg.data.tags or {},
  processed_at = os.time(),
  source = ws.host
}

ws.respond(json.encode(out))
```

### Pattern 3: Stateful Per-Connection Counter

Track per-connection state using the KV store:

```lua
-- scripts/counter.lua
local key = "count:" .. ws.connection_id
local current = tonumber(ws.kv(key)) or 0
local next_count = current + 1

-- Store updated count (TTL managed by kv-set builtin, not here)
ws.respond(json.encode({
  connection = ws.connection_id,
  message_number = next_count,
  message = ws.message
}))
```

```yaml
handlers:
  - name: message-counter
    match: "*"
    builtin: lua
    file: scripts/counter.lua
```

### Pattern 4: Multi-Rule Engine

Evaluate multiple conditions and respond with the first match:

```lua
-- scripts/rules.lua
local msg = json.decode(ws.message)
if not msg then
  ws.respond('{"error":"parse_error"}')
  return
end

local rules = {
  {condition = function(m) return m.priority == "critical" end,
   response = {action = "page", team = "oncall"}},
  {condition = function(m) return m.severity and m.severity >= 3 end,
   response = {action = "alert", channel = "slack"}},
  {condition = function(m) return m.type == "info" end,
   response = {action = "log", level = "info"}},
  {condition = function() return true end,
   response = {action = "ignore"}},
}

for _, rule in ipairs(rules) do
  if rule.condition(msg) then
    ws.respond(json.encode(rule.response))
    return
  end
end
```

### Pattern 5: Inline Script (Short Logic)

For very short scripts, use `script:` instead of `file:`:

```yaml
handlers:
  - name: quick-transform
    match:
      jq: '.type == "transform"'
    builtin: lua
    script: |
      local m = json.decode(ws.message)
      ws.respond(json.encode({
        upper = (m.text or ""):upper(),
        len = #(m.text or ""),
        ts = os.time()
      }))
```

---

## Full Lua API Reference

| Global / Function | Description |
|-------------------|-------------|
| `ws.message` | Raw incoming message (string) |
| `ws.connection_id` | Unique connection identifier |
| `ws.host` | Connected server hostname |
| `ws.path` | WebSocket URL path |
| `ws.respond(msg)` | Send a response to the client |
| `ws.log(msg)` | Write to xwebs log |
| `ws.kv(key)` | Read from server KV store (returns string or "") |
| `json.decode(str)` | Parse JSON string → Lua table |
| `json.encode(val)` | Encode Lua table → JSON string |
| `os.time()` | Current Unix timestamp |
| `string.*` | Standard Lua string library |
| `table.*` | Standard Lua table library |
| `math.*` | Standard Lua math library |

**Restricted (sandboxed):** `os.execute()`, `io.open()`, `require()` — use `run:` shell commands for file I/O and subprocess execution.

---

## Configuration Reference

```yaml
handlers:
  - name: my-lua-handler
    match: "*"
    builtin: lua
    file: scripts/handler.lua   # path to .lua file (recommended)
    # OR:
    script: |                   # inline script (for short logic)
      ws.respond("hello")
    timeout: 10s                # script execution timeout (default: 30s)
```
