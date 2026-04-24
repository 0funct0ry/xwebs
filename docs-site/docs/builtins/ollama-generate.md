---
title: ollama-generate
description: Send a prompt to a local Ollama model and receive the generated text as {{.OllamaReply}}.
---

# `ollama-generate` — Ollama Generate

**Mode:** `[*]` (client and server)

Sends a single `prompt:` to the specified Ollama model using the `/api/generate` endpoint and injects the response into `{{.OllamaReply}}`. No conversation history is maintained — use `ollama-chat` for multi-turn dialogue.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | Yes | — | Ollama model name (e.g. `llama3`, `mistral`). |
| `prompt` | string | Yes | — | Prompt text. Template expression. |
| `url` | string | No | `http://localhost:11434` | Ollama server base URL. |
| `timeout` | string | No | `60s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.OllamaReply}}` | string | Generated text from the model. |

---

## Example

```yaml
handlers:
  - name: ai-reply
    match:
      jq: '.type == "prompt"'
    builtin: ollama-generate
    model: "llama3"
    prompt: '{{.Message | jq ".text"}}'
    respond: '{"reply": {{.OllamaReply | toJSON}}}'
```

---

## Edge Cases

- Ollama must be running locally (or at `url:`) before the handler fires; connection failures cause the handler to error.
- Long generations may hit the `timeout:` limit; increase it for complex prompts.
- The full generated text is buffered before `{{.OllamaReply}}` is populated — streaming is not supported via this builtin.
- For per-connection chat history, use `ollama-chat` instead.
