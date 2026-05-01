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
| `prompt` | string | No | `{{.Message}}` | User message to append to the conversation. |
| `system` | string | No | — | System prompt prepended to every conversation turn. |
| `max_history` | int | No | `0` (unlimited) | Limit the number of messages retained in history. |
| `ollama_url` | string | No | `http://localhost:11434` | Ollama server base URL. |
| `stream` | bool | No | `false` | Whether to stream the response. |
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
    match: "*"
    builtin: ollama-chat
    model: "llama3"
    system: "You are a helpful assistant."
    prompt: "{{.Message}}"
    max_history: 10
    respond: "AI: {{.OllamaReply}}"
```

---

## Edge Cases

- Conversation history is stored in process memory per connection ID; it is lost on server restart or client disconnect.
- `max_history` limits history size; oldest turns are dropped when the limit is exceeded.
- `system:` is prepended to the message history on every call.
- Use `ollama-generate` for stateless single-turn prompts.
