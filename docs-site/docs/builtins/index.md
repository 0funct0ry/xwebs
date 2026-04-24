---
title: Builtins Overview
description: All built-in xwebs handler actions — no shell required. Overview table with mode scope and one-line descriptions.
---

# Builtins

Builtins are handler actions that run inside the xwebs process — no shell subprocess required. They're faster than shell commands, work consistently across platforms, and have access to internal server state (connections, KV store, topics).

Use a builtin by setting `builtin: <name>` in your handler config.

## Mode Scope

- `[S]` — Server mode only (`xwebs serve`)
- `[C]` — Client mode only (`xwebs connect`)
- `[*]` — Available in both modes

---

## All Builtins

| Builtin | Mode | Description |
|---------|------|-------------|
| [`echo`](./echo.md) | `[*]` | Echo the received message back to the sender |
| [`broadcast`](./broadcast.md) | `[S]` | Send a message to all connected clients |
| [`broadcast-others`](./broadcast-others.md) | `[S]` | Send to all clients except the sender |
| [`forward`](./forward.md) | `[S]` | Proxy the message to another WebSocket server |
| [`kv-set`](./kv-set.md) | `[S]` | Set a key-value pair in the server KV store |
| [`kv-get`](./kv-get.md) | `[S]` | Retrieve a value from the KV store |
| [`kv-del`](./kv-del.md) | `[S]` | Delete a key from the KV store |
| [`kv-list`](./kv-list.md) | `[S]` | List all keys in the KV store |
| [`sequence`](./sequence.md) | `[*]` | Cycle through a predefined list of responses |
| [`template`](./template.md) | `[*]` | Render a Go template file and send the result |
| [`file-send`](./file-send.md) | `[C]` | Send a local file as a text or binary frame |
| [`file-write`](./file-write.md) | `[*]` | Write message content to a file |
| [`rate-limit`](./rate-limit.md) | `[*]` | Rate limit incoming messages per client or globally |
| [`delay`](./delay.md) | `[*]` | Defer a response by a configurable duration |
| [`drop`](./drop.md) | `[*]` | Silently discard a message |
| [`close`](./close.md) | `[*]` | Send a WebSocket close frame |
| [`http`](./http.md) | `[*]` | Make an HTTP request and inject the response |
| [`log`](./log.md) | `[*]` | Write a structured log entry |
| [`metric`](./metric.md) | `[*]` | Increment a named Prometheus counter |
| [`publish`](./publish.md) | `[S]` | Publish a message to a named topic |
| [`subscribe`](./subscribe.md) | `[S]` | Subscribe the client to a named topic |
| [`unsubscribe`](./unsubscribe.md) | `[S]` | Unsubscribe the client from a topic |
| [`lua`](./lua.md) | `[*]` | Run embedded Lua script for complex logic |
| [`throttle-broadcast`](./throttle-broadcast.md) | `[S]` | Broadcast, skipping recently-messaged clients |
| [`multicast`](./multicast.md) | `[S]` | Send to a named list of client IDs |
| [`sticky-broadcast`](./sticky-broadcast.md) | `[S]` | Broadcast with message retention for late joiners |
| [`round-robin`](./round-robin.md) | `[S]` | Distribute messages across a pool of clients |
| [`sample`](./sample.md) | `[*]` | Pass only 1-in-N messages |
| [`gate`](./gate.md) | `[S]` | Allow message only if a KV condition is met |
| [`once`](./once.md) | `[S]` | Execute exactly once per server lifetime |
| [`debounce`](./debounce.md) | `[*]` | Suppress rapid messages, pass only the final one |
| [`rule-engine`](./rule-engine.md) | `[*]` | Evaluate an ordered list of condition→response rules |
| [`ab-test`](./ab-test.md) | `[S]` | Route messages to one of two handlers by hash |
| [`shadow`](./shadow.md) | `[S]` | Dispatch to a secondary handler silently |
| [`redis-set`](./redis-set.md) | `[*]` | Set a Redis key |
| [`redis-get`](./redis-get.md) | `[*]` | Get a Redis key |
| [`redis-del`](./redis-del.md) | `[*]` | Delete a Redis key |
| [`redis-publish`](./redis-publish.md) | `[*]` | Publish to a Redis Pub/Sub channel |
| [`redis-subscribe`](./redis-subscribe.md) | `[S]` | Subscribe to a Redis channel |
| [`redis-lpush`](./redis-lpush.md) | `[*]` | Push a value onto a Redis list |
| [`redis-rpop`](./redis-rpop.md) | `[*]` | Pop a value from a Redis list |
| [`redis-incr`](./redis-incr.md) | `[*]` | Atomically increment a Redis counter |
| [`webhook`](./webhook.md) | `[*]` | POST message to an HTTP endpoint |
| [`webhook-hmac`](./webhook-hmac.md) | `[*]` | POST with HMAC-SHA256 signature |
| [`http-get`](./http-get.md) | `[*]` | GET a URL and inject the response |
| [`http-graphql`](./http-graphql.md) | `[*]` | Run a GraphQL query or mutation |
| [`ollama-generate`](./ollama-generate.md) | `[*]` | Generate text with an Ollama model |
| [`ollama-chat`](./ollama-chat.md) | `[*]` | Multi-turn chat with an Ollama model |
| [`ollama-embed`](./ollama-embed.md) | `[*]` | Generate an embedding vector |
| [`ollama-classify`](./ollama-classify.md) | `[*]` | Classify a message against labels |
| [`openai-chat`](./openai-chat.md) | `[*]` | Chat with any OpenAI-compatible endpoint |
| [`mqtt-publish`](./mqtt-publish.md) | `[*]` | Publish to an MQTT topic |
| [`mqtt-subscribe`](./mqtt-subscribe.md) | `[S]` | Subscribe to an MQTT topic |
| [`nats-publish`](./nats-publish.md) | `[*]` | Publish to a NATS subject |
| [`nats-subscribe`](./nats-subscribe.md) | `[S]` | Subscribe to a NATS subject |
| [`kafka-produce`](./kafka-produce.md) | `[*]` | Produce a message to a Kafka topic |
| [`kafka-consume`](./kafka-consume.md) | `[S]` | Consume from a Kafka topic |
| [`sqlite`](./sqlite.md) | `[*]` | Execute parameterized SQL against SQLite |
| [`postgres`](./postgres.md) | `[*]` | Execute SQL against PostgreSQL |
| [`append-file`](./append-file.md) | `[*]` | Append message to a file with rotation |
| [`s3-put`](./s3-put.md) | `[*]` | Upload content to S3 |
| [`s3-get`](./s3-get.md) | `[*]` | Fetch an S3 object |
| [`jq-transform`](./jq-transform.md) | `[*]` | Apply a jq expression and re-inject the result |
| [`json-merge`](./json-merge.md) | `[*]` | Deep-merge a JSON template into the message |
| [`xml-to-json`](./xml-to-json.md) | `[*]` | Convert XML message to JSON |
| [`csv-parse`](./csv-parse.md) | `[*]` | Parse CSV message into row array |
| [`schema-validate`](./schema-validate.md) | `[*]` | Validate against JSON Schema |
