---
title: http-graphql
description: Execute a GraphQL query or mutation over HTTP; response in {{.HttpBody}}, errors in {{.GraphQLErrors}}.
---

# `http-graphql` — GraphQL HTTP Client

**Mode:** `[*]` (client and server)

Sends a GraphQL query or mutation to an HTTP endpoint using the standard `application/json` POST format. The response data is in `{{.HttpBody}}` and any GraphQL errors are in `{{.GraphQLErrors}}`.

---

## Key Config Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string | Yes | — | GraphQL endpoint URL. Template expression. |
| `query` | string | Yes | — | GraphQL query or mutation string. Template expression. |
| `variables` | string | No | — | JSON object of query variables. Template expression. |
| `headers` | map | No | — | HTTP headers (e.g. `Authorization`). |
| `timeout` | string | No | `30s` | Request timeout. |

---

## Template Variables Injected

| Variable | Type | Description |
|----------|------|-------------|
| `{{.HttpBody}}` | string | Full GraphQL JSON response body. |
| `{{.HttpStatus}}` | int | HTTP status code. |
| `{{.GraphQLErrors}}` | string | JSON array of GraphQL errors, or empty string if none. |

---

## Example

```yaml
handlers:
  - name: gql-query
    match:
      jq: '.type == "user_lookup"'
    builtin: http-graphql
    url: "https://api.example.com/graphql"
    query: |
      query GetUser($id: ID!) {
        user(id: $id) { id name email }
      }
    variables: '{"id": "{{.Message | jq ".user_id"}}"}'
    headers:
      Authorization: "Bearer {{env "GQL_TOKEN"}}"
    respond: '{"user": {{.HttpBody | jq ".data.user" | toJSON}}}'
```

---

## Edge Cases

- GraphQL-level errors (in the `errors` field of the response) are extracted into `{{.GraphQLErrors}}` even when `{{.HttpStatus}}` is 200.
- Network and HTTP-level errors still cause the handler to fail.
- The `query:` field is sent as a plain string in the JSON body — do not double-encode it.
- For subscriptions over WebSocket, use a separate `xwebs connect` with `--subprotocol graphql-ws`.
