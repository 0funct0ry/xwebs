---
title: openai-chat
description: Maintain a per-connection chat history with any OpenAI-compatible API endpoint; reply in {{.OllamaReply}}.
---

# `openai-chat` — OpenAI-Compatible Chat

**Mode:** `[*]` (client and server)

Identical in behaviour to `ollama-chat` but targets any OpenAI-compatible `/v1/chat/completions` endpoint — including OpenAI, Azure OpenAI, Groq, Anthropic (via compatibility layer), and self-hosted models. Per-connection conversation history is maintained in memory.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | Yes | — | Model identifier (e.g. `gpt-4o`, `claude-3-5-sonnet`). |
| `url` | string | No | `https://api.openai.com/v1` | Base URL of the OpenAI-compatible API. |
| `api_key` | string | No | — | API key. Template expression; use `{{env "OPENAI_API_KEY"}}`. |
| `message` | string | No | `.Message` | User message. Template expression. |
| `system` | string | No | — | System prompt applied at conversation start. |
| `timeout` | string | No | `60s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.OllamaReply}}` | string | The model's reply text (same variable name as `ollama-chat` for interchangeability). |

---

## Example

```yaml
handlers:
  - name: gpt-chat
    match:
      jq: '.type == "chat"'
    builtin: openai-chat
    model: "gpt-4o"
    api_key: '{{env "OPENAI_API_KEY"}}'
    system: "You are a concise assistant. Reply in one sentence."
    message: '{{.Message | jq ".text"}}'
    respond: '{"reply": {{.OllamaReply | toJSON}}}'
```

---

## Edge Cases

- API costs apply; protect against abuse with `rate-limit` or `gate` builtins upstream.
- Conversation history is in-memory and lost on server restart; store history externally with `redis-set` for durability.
- `api_key:` is a template expression — never hardcode keys; use environment variables.
- To target Azure OpenAI, set `url:` to your Azure endpoint and include `api-version` in the URL or headers.
