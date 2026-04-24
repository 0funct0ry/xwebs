---
title: ollama-embed
description: Generate an embedding vector for the input text using an Ollama model; result in {{.Embedding}}.
---

# `ollama-embed` — Ollama Embeddings

**Mode:** `[*]` (client and server)

Calls the Ollama `/api/embeddings` endpoint to generate a vector embedding for the provided text. The resulting float array is injected as `{{.Embedding}}` (JSON array of numbers).

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | Yes | — | Embedding model name (e.g. `nomic-embed-text`). |
| `input` | string | No | `.Message` | Text to embed. Template expression. |
| `url` | string | No | `http://localhost:11434` | Ollama server base URL. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Embedding}}` | string | JSON array of float values representing the embedding vector. |

---

## Example

```yaml
handlers:
  - name: embed-text
    match:
      jq: '.type == "embed"'
    builtin: ollama-embed
    model: "nomic-embed-text"
    input: '{{.Message | jq ".text"}}'
    respond: '{"embedding": {{.Embedding}}}'
```

---

## Edge Cases

- `{{.Embedding}}` is a JSON array string; pipe it to `fromJSON` for further template manipulation.
- Different models produce different vector dimensions; ensure downstream consumers expect the correct size.
- Embedding generation is CPU/GPU-intensive; set `timeout:` appropriately for large inputs.
- Ollama must have the specified model pulled locally before the handler is invoked.
