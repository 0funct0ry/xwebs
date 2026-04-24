---
title: once
description: Execute a handler exactly once per server lifetime, then automatically disable itself.
---

# `once` — Execute Once

**Mode:** `[S]` (server only)

Runs the handler body (shell command, `respond:`, or pipeline) the first time it matches a message, then permanently disables itself for the lifetime of the server process. Subsequent matching messages fall through to lower-priority handlers as if `once` were never registered.

Use `once` for one-shot initialisation tasks triggered by the first client message, single-use tokens, or welcome messages that should only be sent to the first connected client.

---

## Config Fields

`once` has no configuration fields of its own. It is applied as the builtin and the handler's normal `match:`, `respond:`, `run:`, and `pipeline:` fields take effect for the single allowed execution.

---

## Template Variables Injected

None. Standard context is available in all template fields during the single execution.

---

## Examples

**Send a welcome message to the first connected client only:**

```yaml
handlers:
  - name: first-client-welcome
    match:
      jq: '.type == "hello"'
    builtin: once
    respond: |
      {
        "type": "welcome",
        "message": "You are the first client!",
        "server_start": "{{now | formatTime "RFC3339"}}"
      }
```

**Run a one-time database migration on first message:**

```yaml
handlers:
  - name: one-time-migrate
    match:
      jq: '.type == "init"'
    builtin: once
    run: ./migrate.sh --run-pending
    respond: '{"migrated": true, "output": {{.Stdout | toJSON}}}'
```

**Issue a single-use token to the first authenticated client:**

```yaml
handlers:
  - name: bootstrap-token
    match:
      jq: '.type == "request_token"'
    builtin: once
    respond: '{"bootstrap_token": "{{env "BOOTSTRAP_SECRET"}}"}'
```

---

## Edge Cases

- The disable is in-memory only; restarting the server re-enables the handler for one more execution.
- If the handler execution fails (non-zero exit code, template error), it is still marked as executed and will not run again.
- `once` is compatible with `priority:` — higher-priority handlers still run before `once` on the same message. Only after `once` itself fires does it disable.
- Checking whether `once` has fired can be done indirectly via `:handler <name>` in the REPL — the match count will be 1 and the handler will show as disabled.
