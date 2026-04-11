# Testing xwebs Server with Chrome DevTools

This guide walks you through using Google Chrome's built-in DevTools to test and interact with an `xwebs serve` instance, verifying routing paths and handler execution without needing external CLI tools or writing a custom frontend.

## Prerequisites

Start an `xwebs` server locally. For this guide, we'll start one with multiple paths and a simple echo handler.

1. Create a simple handler file (`echo.yaml`):

```yaml
handlers:
  - name: echo
    match: "*"
    respond: "Echo: {{.Message}}"
```

2. Start the server:

```bash
xwebs serve --port 8999 --path /ws1 --path /ws2 --handlers echo.yaml --verbose
```

## Connecting via Chrome Console

You can establish and interact with WebSocket connections directly from the Chrome DevTools Console. 

*Note: Some websites enforce strict Content Security Policies (CSP) that block outgoing WebSocket connections. It is highly recommended to open a blank page (`about:blank`) before opening DevTools.*

1. Open Chrome and navigate to `about:blank`.
2. Open DevTools (Right-click -> **Inspect**, or press `F12` / `Cmd+Option+I` on macOS).
3. Navigate to the **Console** tab.

### Step 1: Connecting to a Path

Create a WebSocket connection to the first path `/ws1` and set up basic logging:

```javascript
// Create a new WebSocket connection
const ws1 = new WebSocket('ws://localhost:8999/ws1');

// Setup event listeners to monitor the connection state
ws1.onopen = () => console.log('✅ Connected to /ws1');
ws1.onmessage = (event) => console.log('📩 Received on /ws1:', event.data);
ws1.onclose = (event) => console.log(`❌ Disconnected from /ws1 (Code: ${event.code})`);
ws1.onerror = (err) => console.error('⚠️ Error on /ws1:', err);
```

Press Enter. Once the connection is established, you should see `✅ Connected to /ws1` printed in the console.

### Step 2: Verifying Multiple Paths

Now, let's verify that the second path `/ws2` is also active:

```javascript
const ws2 = new WebSocket('ws://localhost:8999/ws2');
ws2.onopen = () => console.log('✅ Connected to /ws2');
```

You should see `✅ Connected to /ws2`.

Try connecting to an invalid path to verify the server correctly rejects it with a 404:

```javascript
const ws3 = new WebSocket('ws://localhost:8999/invalid');
// This will result in an error: WebSocket connection to 'ws://localhost:8999/invalid' failed
```

### Step 3: Triggering Handlers

Since we loaded `echo.yaml` (which replies with `"Echo: {{.Message}}"`), the server should echo back anything we send.

Send a message over the `/ws1` connection:

```javascript
ws1.send('Hello from Chrome DevTools!');
```

You should immediately see the handler's response logged in the console:
`📩 Received on /ws1: Echo: Hello from Chrome DevTools!`

## Inspecting Frames in the Network Tab

While the Console is great for interactive testing, Chrome DevTools provides a dedicated viewer for inspecting WebSocket frames in detail.

1. Navigate to the **Network** tab in DevTools.
2. Filter the network requests by clicking the **WS** (WebSocket) filter tab at the top.
3. If you connected via the console *before* opening the Network tab, you might need to refresh the page and reconnect to capture the initial HTTP handshake.
4. Click on the active connection (e.g., `ws1` or `localhost`) in the Name column.
5. In the details pane that opens, click the **Messages** tab.

Here, you will see a live, color-coded table of all sent (⬆) and received (⬇) frames. 
- **Light Green rows**: Messages sent by the client (Chrome).
- **White rows**: Messages received from the server (`xwebs`).

You can click on individual frames to inspect their raw data and length at the bottom of the pane. This is extremely useful for debugging complex JSON handler payloads and ensuring the server is formatting responses correctly.

## Testing Graceful Disconnect

### Client-Initiated Close
To test a graceful shutdown from the client-side, close the connection with a specific code and reason:

```javascript
ws1.close(1000, "Client finished testing");
```

If your `xwebs serve` instance is running with `--verbose`, you will see the server acknowledge the graceful close code `1000` and the custom reason in your terminal.

### Server-Initiated Close
To test server-side graceful shutdown:
1. Ensure `ws2` is still connected in your Chrome console.
2. Go to your terminal where `xwebs serve` is running and press `Ctrl+C`.
3. Switch back to Chrome. You will see the `onclose` event trigger in the Console: `❌ Disconnected from /ws2 (Code: 1000)`.

This confirms `xwebs serve` correctly drains and gracefully closes client connections upon receiving termination signals.
