---
title: Binary Frames
description: How to send, receive, and match WebSocket binary frames in xwebs — including :sendb syntax, handler matching, and server-side send.
---

# Binary Frames

Binary frames (WebSocket opcode `0x2`) carry raw byte arrays with no UTF-8 validation. They're used for Protocol Buffers, CBOR/MessagePack, media streaming, and low-level protocol debugging.

---

## Sending Binary from the REPL

The `:sendb` command sends a binary frame. It auto-detects the encoding format:

```
:sendb <hex>              Hex string → binary frame (e.g., :sendb 48656c6c6f)
:sendb base64:<data>      Base64 prefix → decoded binary frame (e.g., :sendb base64:SGVsbG8=)
```

Examples:

```
xwebs> :sendb 48656c6c6f
→ sent binary frame (5 bytes)

xwebs> :sendb base64:SGVsbG8=
→ sent binary frame (5 bytes)

xwebs> :sendb deadbeef
→ sent binary frame (4 bytes)
```

**No file path option:** pipe file content via shell if needed:

```bash
echo ":sendb base64:$(base64 -i payload.bin)" | xwebs connect ws://localhost:8080
```

---

## Binary Ping/Pong Control Frames

Binary payloads in control frames are used for specialized heartbeats and latency measurement:

```
:ping hex:010203          Ping with hex payload
:pong base64:AQID         Pong with Base64 payload
```

---

## Matching Binary Frames in Handlers

Use `match.binary` to route by frame type:

```yaml
- name: handle-binary
  match:
    binary: true          # matches only binary frames (opcode 0x2)
  run: xxd                # hex dump the frame payload

- name: handle-text
  match:
    binary: false         # matches only text frames (opcode 0x1)
  run: jq '.'
```

Combine with other matchers using `all:`:

```yaml
- name: large-binary
  match:
    all:
      - binary: true
      - template: '{{gt .MessageLen 1024}}'
  run: ./process-large-binary.sh
```

---

## Server-Side `:send` and `:broadcast` with Binary

From the server REPL:

```
:send -b <client-id> 48656c6c6f           Send hex bytes to a specific client
:send -b <client-id> base64:SGVsbG8=      Send Base64 bytes to a specific client
:broadcast -b 48656c6c6f                  Broadcast hex bytes to all clients
:broadcast -b base64:SGVsbG8=             Broadcast Base64 bytes to all clients
```

---

## Real-World Scenarios

### Protocol Buffers

```bash
# 1. Define your message
# user.proto: message User { string name = 1; int32 id = 2; }

# 2. Encode to binary
echo 'name: "Alice" id: 123' | protoc --encode=User user.proto > payload.bin

# 3. Send via xwebs
echo ":sendb base64:$(base64 < payload.bin)" | xwebs connect ws://localhost:8080
```

Handler to receive and decode:

```yaml
- name: handle-protobuf
  match:
    binary: true
  run: |
    cat <<< "{{.MessageBytes | base64Encode}}" | base64 -d \
      | protoc --decode=User user.proto
  respond: '{"decoded":"{{.Stdout | trim}}"}'
```

### CBOR / MessagePack

```bash
# Send a MessagePack-encoded payload
python3 -c "import msgpack; print(msgpack.packb({'type':'event','id':42}).hex())" \
  | xargs -I{} xwebs connect ws://localhost:8080 --send ":sendb {}"
```

### File Upload Over WebSocket

```bash
# Client side: send a file as binary
xwebs connect ws://localhost:8080 --script - <<'EOF'
:sendb base64:$(base64 -i image.png)
:expect .status == "received"
EOF
```

Server handler:

```yaml
- name: receive-file
  match:
    binary: true
  builtin: file-write
  path: /uploads/{{uuid}}.bin
  respond: '{"status":"received","size":{{.MessageLen}}}'
```

---

## External Tools

When `:sendb` isn't enough, these tools integrate well with xwebs:

### websocat

```bash
# Send a binary file
websocat -b ws://localhost:8080 < image.png

# Send hex-decoded bytes
echo "deadbeef" | xxd -r -p | websocat -b ws://localhost:8080
```

### wscat

```bash
# Send a binary file
wscat -c ws://localhost:8080 --file payload.bin
```

### curl

```bash
# Basic WebSocket upgrade (recent curl versions)
curl --include --no-buffer \
  --header "Connection: Upgrade" \
  --header "Upgrade: websocket" \
  ws://localhost:8080
```

---

## Output Formatting for Binary

In the REPL, switch to hex dump mode to inspect binary frames:

```
xwebs> :format hex
xwebs> :sendb base64:SGVsbG8=
← 00000000  48 65 6c 6c 6f                                    |Hello|
```

Switch back to raw:

```
xwebs> :format raw
```
