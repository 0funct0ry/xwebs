---
title: ollama-chat
description: Maintain a per-connection chat history with an Ollama model; latest reply in {{.OllamaReply}}.
---

# `ollama-chat` — Ollama Chat

**Mode:** `[*]` (client and server)

Sends a message to an Ollama model using the `/api/chat` endpoint while maintaining per-connection conversation history. Each connection has its own isolated message history so multi-turn conversations work naturally.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | Yes | — | Ollama model name. |
| `message` | string | No | `.Message` | User message to append to the conversation. Template expression. |
| `system` | string | No | — | System prompt set at the start of the conversation. |
| `url` | string | No | `http://localhost:11434` | Ollama server base URL. |
| `timeout` | string | No | `60s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.OllamaReply}}` | string | The model's reply to the latest message. |

---

## Example

```yaml
handlers:
  - name: chat-bot
    match:
      jq: '.type == "chat"'
    builtin: ollama-chat
    model: "llama3"
    system: "You are a helpful assistant."
    message: '{{.Message | jq ".text"}}'
    respond: '{"type": "reply", "text": {{.OllamaReply | toJSON}}}'
```

---

## Edge Cases

- Conversation history is stored in process memory per connection ID; it is lost on server restart or client disconnect.
- History grows unboundedly; for long sessions, the context window may be exceeded. Implement periodic resets via a separate `kv-del` handler.
- `system:` is only applied at the start of the conversation; changing it mid-session has no effect unless the history is cleared.
- Use `ollama-generate` for stateless single-turn prompts.
