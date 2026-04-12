# Binary WebSocket Frames Guide

Binary frames (Opcode `0x2`) are essential for high-performance applications, media streaming, and low-level protocol debugging. Unlike text frames, they carry raw byte arrays and are not subject to UTF-8 validation.

## 1. xwebs Native Support

`xwebs` provides several ways to interact with binary frames directly from the REPL or via automation.

### :sendb (Send Binary)
The primary command for sending binary data. It automatically handles encoding detection.

- **Hex**: `:sendb 48656c6c6f` (Hex for "Hello")
- **Base64**: `:sendb base64:SGVsbG8=` (Base64 for "Hello")

### :ping / :pong
Binary payloads are often used in control frames for latency measurement or specialized heartbeats.
- `:ping hex:010203`
- `:pong base64:AQID`

---

## 2. External Tools Ecosystem

### websocat (The Power User Choice)
The most robust tool for binary pipes.
- **File Pipe**: `websocat -b ws://url < image.png`
- **Hex Pipe**: `echo "deadbeef" | xxd -r -p | websocat -b ws://url`

### wscat
Simple tool for WebSocket testing. Use `--file` for binary data.
- `wscat -c ws://url --file data.bin`

### curl
Recent versions of `curl` support basic WebSocket handshaking.
- `curl --include --no-buffer --header "Connection: Upgrade" --header "Upgrade: websocket" ws://url`

---

## 3. Real-World Scenarios

### Case A: Protocol Buffers (gRPC-Web/Twirp)
Many modern APIs use Protobuf over WebSockets for efficient communication. Bytes are usually Base64-encoded in the protocol but handled as binary by the WebSocket layer.

**Step-by-Step CLI Workflow:**
1. **Define your message** (`user.proto`):
   ```proto
   message User { string name = 1; int32 id = 2; }
   ```
2. **Generate binary payload from a text representation**:
   ```bash
   echo 'name: "Alice" id: 123' | protoc --encode=User user.proto > payload.bin
   ```
3. **Convert to Base64 (Mac/Linux)**:
   ```bash
   B64_PAYLOAD=$(base64 < payload.bin)
   ```
4. **Send via xwebs**:
   - **Interactive**: In REPL, type `:sendb base64:<PASTE_B64_HERE>`
   - **Automated**: `echo ":sendb base64:$B64_PAYLOAD" | xwebs connect URL`

### Case B: Binary JSON (CBOR/MessagePack)
To save bandwidth (up to 40%), JSON is often swapped for CBOR.
- **Xwebs Trigger**: `--on-match '{"match":{"binary":true}, "run":"cbor-to-json --input -"}'`

### Case C: Media Streaming (WebRTC Data Channels / H.264)
Streaming raw video chunks over a persistent socket to reduce overhead.
- **Verification**: Use `--verbose` in `xwebs` to verify the `len=` metadata of high-frequency frames.

---

## 4. 20 Practical Examples

### Interactive xwebs Commands (REPL)
1. **Basic Hex**: `:sendb 01020304`
2. **Basic Base64**: `:sendb base64:AQIDBA==`
3. **Empty Frame**: `:sendb ""` (Sends a 0-byte binary frame)
4. **Binary Heartbeat**: `:ping hex:FF`
5. **Control ACK**: `:pong hex:FE`
6. **Sensor Tuple**: `:sendb 00010002` (Simulating [id, value] as 16-bit ints)

### Shell Automation (Pipes)
7. **Scripted Send**: `echo ":sendb $(base64 -i file.bin)" | xwebs connect URL --quiet`
8. **Streaming Logs**: `tail -f binary.log | websocat -b URL`
9. **Raw Byte Injection**: `printf "\x01\x02\x03" | websocat -b URL`
10. **Random Noise Test**: `head -c 10 /dev/urandom | websocat -b URL`

### Server-Side Handlers (xwebs serve)
11. **Detect Binary**: `--on-match '{"match":{"binary":true}, "run":"echo Binary frame detected"}'`
12. **Pipeline to Capture**: `--on-match '{"match":{"binary":true}, "run":"cat >> capture.bin"}'`
13. **Live Hex Dump**: `--on-match '{"match":{"binary":true}, "run":"xxd"}'`
14. **Protobuf Inspector**: `--on-match '{"match":{"binary":true}, "run":"protoc --decode_raw"}'`

### Advanced / Edge Cases
15. **Large Payload**: `:sendb $(python3 -c "print('AA'*1000)")` (Testing fragmentation)
16. **Nul-terminated String**: `:sendb 48656c6c6f00` (Sending "Hello\0")
17. **Gzip Header Test**: `:sendb hex:1F8B08` (Verifying server handles magic numbers)
18. **Length Validation**: Use `{{ .Msg.Raw | len }}` in a template to verify size.
19. **Binary Broadcast**: `xwebs relay --on-match '{"match":{"binary":true}, "builtin":"relay"}'`
20. **Stress Loop**: `for i in {1..100}; do echo ":sendb 01"; done | xwebs connect URL`


