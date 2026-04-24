---
title: xml-to-json
description: Parse an XML incoming message and convert it to a JSON representation for downstream processing.
---

# `xml-to-json` — XML to JSON

**Mode:** `[*]` (client and server)

Parses the incoming message as XML and converts it to a JSON object, replacing `{{.Message}}` with the JSON result. This enables XML-speaking clients or upstream systems to interact with JSON-native handler pipelines.

---

## Key Config Fields

`xml-to-json` has no configuration fields beyond the standard handler fields. The incoming message is the XML input.

---

## Example

**Convert incoming XML to JSON, then apply jq:**

```yaml
handlers:
  - name: xml-bridge
    match:
      regex: "^<"
    builtin: xml-to-json
    respond: '{"type": "converted", "data": {{.Message | jq ".root.item" | toJSON}}}'
```

**Pipeline: convert then store:**

```yaml
handlers:
  - name: xml-ingest
    match:
      regex: "^<"
    pipeline:
      - builtin: xml-to-json
      - builtin: sqlite
        db: "data/events.db"
        query: 'INSERT INTO events (data) VALUES ("{{.Message | shellEscape}}")'
    respond: '{"inserted": true}'
```

---

## Edge Cases

- Malformed XML causes the handler to error; no response is sent.
- XML attributes are typically represented as `@attr` keys in the JSON output; XML namespaces appear as prefixed keys.
- The JSON representation of XML is lossless for round-tripping but may produce verbose output with nested `#text` keys for mixed content.
- Very large XML documents may be slow to parse; set `timeout:` accordingly.
