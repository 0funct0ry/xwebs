---
title: ollama-classify
description: Classify an incoming message against a set of labels using an Ollama model; top label in {{.Label}}.
---

# `ollama-classify` — Ollama Classification

**Mode:** `[*]` (client and server)

Sends the message to an Ollama model with a classification prompt built from `labels:`. The model returns the most likely label, which is injected as `{{.Label}}`. Useful for intent detection, sentiment analysis, and content routing.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model` | string | Yes | — | Ollama model name. |
| `labels` | list of strings | Yes | — | Candidate labels for classification. |
| `input` | string | No | `.Message` | Text to classify. Template expression. |
| `url` | string | No | `http://localhost:11434` | Ollama server base URL. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.Label}}` | string | The label the model selected as the best match. |

---

## Example

```yaml
handlers:
  - name: intent-router
    match:
      jq: '.type == "message"'
    builtin: ollama-classify
    model: "llama3"
    labels: ["greeting", "question", "complaint", "purchase"]
    input: '{{.Message | jq ".text"}}'
    respond: '{"intent": "{{.Label}}"}'
```

---

## Edge Cases

- The model may return a label not in `labels:` if it ignores the constraint; validate `{{.Label}}` against the list before routing.
- Classification accuracy depends heavily on the model and the clarity of the label names.
- For deterministic classification, consider `rule-engine` with jq conditions instead.
- Latency is similar to `ollama-generate`; set an appropriate `timeout:`.
