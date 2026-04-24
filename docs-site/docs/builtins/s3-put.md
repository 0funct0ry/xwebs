---
title: s3-put
description: Upload content to an S3-compatible object storage bucket.
---

# `s3-put` — S3 Upload

**Mode:** `[*]` (client and server)

Uploads a payload to an S3-compatible bucket (AWS S3, MinIO, DigitalOcean Spaces, etc.). The object key and content are template expressions.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `bucket` | string | Yes | — | S3 bucket name. Template expression. |
| `key` | string | Yes | — | Object key (path within the bucket). Template expression. |
| `content` | string | No | `.Message` | Content to upload. Template expression. |
| `content_type` | string | No | `application/octet-stream` | MIME type for the uploaded object. |
| `region` | string | No | `us-east-1` | AWS region. |
| `endpoint` | string | No | — | Custom endpoint URL for S3-compatible services (e.g. MinIO). |

---

## Example

```yaml
handlers:
  - name: upload-to-s3
    match:
      jq: '.type == "upload"'
    builtin: s3-put
    bucket: "my-data-bucket"
    key: 'uploads/{{now | formatTime "2006/01/02"}}/{{uuid}}.json'
    content: '{{.Message | toJSON}}'
    content_type: "application/json"
    respond: '{"uploaded": true}'
```

---

## Edge Cases

- AWS credentials are read from the standard AWS credential chain (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, IAM role, etc.) — do not hardcode in config.
- Object keys with special characters should be URL-safe; use `{{.Value | urlEncode}}` for user-supplied key segments.
- Large uploads (>5 GB) require multipart upload, which this builtin does not currently support; use a `run:` step with `aws s3 cp` for large objects.
- The `endpoint:` field enables MinIO or other S3-compatible backends; also set `region: us-east-1` as a placeholder when the backend ignores it.
