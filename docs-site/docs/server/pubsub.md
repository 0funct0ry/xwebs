---
title: Pub/Sub Topics
description: How xwebs pub/sub works, the subscribe/unsubscribe/publish builtins, and wire format examples.
---

# Pub/Sub Topics

xwebs includes a built-in publish/subscribe system for server mode. Topics are logical channels — clients subscribe to them and receive messages published to those channels. There is no special protocol: subscriptions and publications happen through regular WebSocket messages matched by handlers.

## How It Works

Topics are created implicitly when the first client subscribes. Each topic maintains a list of subscriber client IDs. When a message is published to a topic, it is fan-out delivered to all current subscribers.

The operator defines:
1. **What incoming message triggers a subscription** — using the `subscribe` builtin
2. **What triggers a publication** — using the `publish` builtin or `:publish` REPL command

The wire format (what the client sends) is entirely under your control.

---

## The `subscribe` Builtin

Subscribes the current client to a named topic.

```yaml
- name: handle-subscribe
  match:
    jq: '.type == "subscribe"'
  builtin: subscribe
  topic: '{{.Message | jq ".channel"}}'
  respond: '{"subscribed":"{{.Message | jq ".channel"}}"}'
```

When a client sends `{"type":"subscribe","channel":"trades"}`, xwebs subscribes that client to the `trades` topic and sends back a confirmation.

The `topic:` field is a Go template — you can derive the topic name from the message content, connection ID, or any other template variable.

---

## The `unsubscribe` Builtin

Removes the current client from a topic.

```yaml
- name: handle-unsubscribe
  match:
    jq: '.type == "unsubscribe"'
  builtin: unsubscribe
  topic: '{{.Message | jq ".channel"}}'
  respond: '{"unsubscribed":"{{.Message | jq ".channel"}}"}'
```

---

## The `publish` Builtin

Publishes a message to all subscribers of a topic.

```yaml
- name: handle-publish
  match:
    jq: '.type == "publish"'
  builtin: publish
  topic: '{{.Message | jq ".channel"}}'
  message: '{{.Message | jq ".data"}}'
```

When a client sends `{"type":"publish","channel":"trades","data":{"price":42.0}}`, the `data` payload is delivered to every subscriber of `trades`.

If no subscribers exist, the message is silently dropped. To publish regardless, set `allow_empty: true` in the builtin config.

---

## Wire Format Examples

xwebs is protocol-agnostic — you choose the wire format. Here are three common patterns:

### JSON-Style (Explicit Type Field)

Clients send structured JSON with a `type` field:

```bash
# Subscribe
{"type":"subscribe","channel":"trades"}
# Response
{"subscribed":"trades"}

# Publish
{"type":"publish","channel":"trades","data":{"symbol":"BTC","price":65000}}

# Unsubscribe
{"type":"unsubscribe","channel":"trades"}
```

Handler config:

```yaml
handlers:
  - name: sub
    match:
      jq: '.type == "subscribe"'
    builtin: subscribe
    topic: '{{.Message | jq ".channel"}}'
    respond: '{"subscribed":"{{.Message | jq ".channel"}}"}'

  - name: pub
    match:
      jq: '.type == "publish"'
    builtin: publish
    topic: '{{.Message | jq ".channel"}}'
    message: '{{.Message | jq ".data" | toJSON}}'

  - name: unsub
    match:
      jq: '.type == "unsubscribe"'
    builtin: unsubscribe
    topic: '{{.Message | jq ".channel"}}'
    respond: '{"unsubscribed":"{{.Message | jq ".channel"}}"}'
```

### Prefix-Style (Simple Text Protocol)

Clients send plain-text commands with a prefix:

```
sub:trades
pub:trades:{"price":42.0}
unsub:trades
```

Handler config:

```yaml
handlers:
  - name: sub
    match: 'sub:*'
    builtin: subscribe
    topic: '{{.Message | trimPrefix "sub:"}}'
    respond: 'subscribed:{{.Message | trimPrefix "sub:"}}'

  - name: pub
    match: 'pub:*'
    builtin: publish
    topic: '{{index (split ":" .Message) 1}}'
    message: '{{index (split ":" .Message) 2}}'

  - name: unsub
    match: 'unsub:*'
    builtin: unsubscribe
    topic: '{{.Message | trimPrefix "unsub:"}}'
    respond: 'unsubscribed:{{.Message | trimPrefix "unsub:"}}'
```

### Sigil-Style (Compact One-Character Prefix)

Ultra-compact protocol where the first character is the command:

```
+trades          # subscribe to trades
-trades          # unsubscribe from trades
>trades:{"p":42} # publish to trades
```

Handler config:

```yaml
handlers:
  - name: sub
    match: '^\\+'
    builtin: subscribe
    topic: '{{.Message | trimPrefix "+"}}'

  - name: unsub
    match: '^-'
    builtin: unsubscribe
    topic: '{{.Message | trimPrefix "-"}}'

  - name: pub
    match: '^>'
    builtin: publish
    topic: '{{index (split ":" (.Message | trimPrefix ">")) 0}}'
    message: '{{index (split ":" (.Message | trimPrefix ">")) 1}}'
```

---

## REPL Commands for Topics

You can manage topics directly from the server REPL without any handler:

```
:topics                           List all active topics
:topic trades                     Show subscriber details for "trades"
:publish trades {"price":65000}   Publish to all subscribers
:subscribe c-a1b2c3 trades        Manually subscribe a client
:unsubscribe c-a1b2c3 trades      Remove a client from a topic
:unsubscribe c-a1b2c3 --all       Remove from all topics
```

---

## Advanced: Sticky Broadcast

The `sticky-broadcast` builtin combines publish with message retention — late-joining subscribers immediately receive the last published message:

```yaml
- name: price-update
  match:
    jq: '.type == "price"'
  builtin: sticky-broadcast
  topic: '{{.Message | jq ".symbol"}}'
  message: '{{.Message | toJSON}}'
```

See [sticky-broadcast](../builtins/sticky-broadcast.md) for full configuration.
