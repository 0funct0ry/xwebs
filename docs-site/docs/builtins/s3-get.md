---
title: s3-get
description: Download an object from an S3-compatible bucket; content available as {{.S3Body}}.
---

# `s3-get` — S3 Download

**Mode:** `[*]` (client and server)

Fetches an object from an S3-compatible bucket and injects its content as `{{.S3Body}}`.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `bucket` | string | Yes | — | S3 bucket name. Template expression. |
| `key` | string | Yes | — | Object key to fetch. Template expression. |
| `region` | string | No | `us-east-1` | AWS region. |
| `endpoint` | string | No | — | Custom endpoint for S3-compatible services. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.S3Body}}` | string | Object content as a UTF-8 string (binary objects are base64-encoded). |

---

## Example

```yaml
handlers:
  - name: fetch-from-s3
    match:
      jq: '.type == "fetch_config"'
    builtin: s3-get
    bucket: "my-config-bucket"
    key: 'configs/{{.Message | jq ".name"}}.json'
    respond: '{"config": {{.S3Body | fromJSON | toJSON}}}'
```

---

## Edge Cases

- If the object does not exist, the handler errors (S3 returns 404).
- Binary objects are base64-encoded in `{{.S3Body}}`; decode with `{{.S3Body | base64Decode}}` before processing.
- Very large objects increase memory pressure; prefer fetching only what is needed or use streaming via a `run:` step with `aws s3 cp`.
- AWS credentials follow the standard credential chain — no hardcoding required if running on EC2/ECS with an IAM role.
