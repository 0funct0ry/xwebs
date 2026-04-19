package server

import (
	"math/rand/v2"
	"sort"
)

// RandomTemplate returns a random template from the collection.
func RandomTemplate() string {
	keys := GetAvailableStyles()
	return CannedTemplates[keys[rand.IntN(len(keys))]]
}

// GetAvailableStyles returns the list of available template styles.
func GetAvailableStyles() []string {
	keys := make([]string, 0, len(CannedTemplates))
	for k := range CannedTemplates {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// CannedTemplates contains 20 diverse, high-quality HTML templates.
var CannedTemplates = map[string]string{
	"modern": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Modern Dashboard</title>
    <style>
        :root { --bg: #0f172a; --card: #1e293b; --accent: #38bdf8; --text: #f8fafc; --muted: #94a3b8; }
        body { font-family: 'Inter', system-ui, sans-serif; background: var(--bg); color: var(--text); margin: 0; display: flex; height: 100vh; }
        .sidebar { width: 260px; background: #020617; border-right: 1px solid #1e293b; padding: 2rem; display: flex; flex-direction: column; }
        .main { flex: 1; padding: 2.5rem; overflow-y: auto; }
        .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2rem; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1.5rem; margin-bottom: 2rem; }
        .card { background: var(--card); padding: 1.5rem; border-radius: 1rem; border: 1px solid rgba(255,255,255,0.05); }
        .card h3 { margin: 0; color: var(--muted); font-size: 0.875rem; text-transform: uppercase; letter-spacing: 0.05em; }
        .card .value { font-size: 1.875rem; font-weight: 700; margin-top: 0.5rem; color: var(--accent); }
        #logs { background: #000; border-radius: 0.75rem; padding: 1.5rem; height: 300px; overflow-y: auto; font-family: monospace; border: 1px solid #334155; }
        .log-in { color: #10b981; } .log-out { color: #38bdf8; } .log-sys { color: #64748b; font-style: italic; }
        .input-area { margin-top: 1.5rem; position: relative; }
        input { width: 100%; background: #020617; border: 1px solid #334155; padding: 1rem; color: #fff; border-radius: 0.5rem; outline: none; }
        input:focus { border-color: var(--accent); }
    </style>
</head>
<body>
    <div class="sidebar">
        <h2 style="color: var(--accent)">xwebs</h2>
        <nav style="margin-top: 2rem; display: flex; flex-direction: column; gap: 1rem;">
            <div style="color: var(--muted)">Dashboard</div>
            <div style="color: var(--muted)">Analytics</div>
            <div style="color: var(--muted)">Settings</div>
        </nav>
    </div>
    <div class="main">
        <div class="header">
            <h1>Live Monitor</h1>
            <div id="status" style="padding: 0.5rem 1rem; border-radius: 9999px; background: rgba(255,255,255,0.05); font-size: 0.875rem;">Connecting...</div>
        </div>
        <div class="stats">
            <div class="card"><h3>Connections</h3><div class="value">Active</div></div>
            <div class="card"><h3>Latency</h3><div class="value">-- ms</div></div>
            <div class="card"><h3>Messages</h3><div class="value">0</div></div>
        </div>
        <div id="logs"></div>
        <div class="input-area"><input type="text" id="input" placeholder="Broadcasting message..."></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        const status = document.getElementById('status');
        let socket;
        function addLog(type, m) {
            const d = document.createElement('div'); d.className = 'log-'+type; d.textContent = (type==='in'?'← ':type==='out'?'→ ':'')+m;
            logs.appendChild(d); logs.scrollTop = logs.scrollHeight;
        }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => { status.textContent = 'Connected'; status.style.color = '#10b981'; addLog('sys', 'Connection established'); };
            socket.onclose = () => { status.textContent = 'Disconnected'; status.style.color = '#ef4444'; setTimeout(connect, 3000); };
            socket.onmessage = e => addLog('in', e.data);
            socket.onerror = () => addLog('sys', 'WebSocket Error');
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); addLog('out', input.value); input.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"terminal": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Cyber Terminal</title>
    <style>
        body { background: #000; color: #0f0; font-family: 'Courier New', monospace; margin: 0; display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
        .crt { position: absolute; top: 0; left: 0; right: 0; bottom: 0; background: linear-gradient(rgba(18, 16, 16, 0) 50%, rgba(0, 0, 0, 0.25) 50%), linear-gradient(90deg, rgba(255, 0, 0, 0.06), rgba(0, 255, 0, 0.02), rgba(0, 0, 255, 0.06)); background-size: 100% 2px, 3px 100%; pointer-events: none; z-index: 2; }
        .scanline { width: 100%; height: 100px; z-index: 3; background: linear-gradient(0deg, rgba(0, 0, 0, 0) 0%, rgba(255, 255, 255, 0.2) 50%, rgba(0, 0, 0, 0) 100%); opacity: 0.1; position: absolute; bottom: 100%; animation: scanline 10s linear infinite; }
        @keyframes scanline { 0% { bottom: 100%; } 100% { bottom: -100px; } }
        #output { flex: 1; overflow-y: auto; padding: 2rem; z-index: 1; }
        .prompt-area { padding: 1rem 2rem; background: #000; display: flex; border-top: 1px solid #060; }
        input { background: transparent; border: none; color: #0f0; font-family: inherit; font-size: 1.2rem; flex: 1; outline: none; margin-left: 10px; }
    </style>
</head>
<body>
    <div class="crt"></div><div class="scanline"></div>
    <div id="output"><div>XWEBS v1.0.0 SESSION START</div><div style="color: #0c0">------------------------------</div></div>
    <div class="prompt-area"><span>></span><input type="text" id="input" autofocus></div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const output = document.getElementById('output');
        const input = document.getElementById('input');
        let socket;
        function print(m, color='#0f0') {
            const d = document.createElement('div'); d.style.color = color; d.textContent = '['+new Date().toLocaleTimeString()+'] '+m;
            output.appendChild(d); output.scrollTop = output.scrollHeight;
        }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => print('CONNECTION STATUS: SECURE', '#0a0');
            socket.onclose = () => { print('CONNECTION LOST. RETRYING...', '#f00'); setTimeout(connect, 3000); };
            socket.onmessage = e => print('<<< ' + e.data, '#0f0');
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); print('>>> ' + input.value, '#0c0'); input.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"glass": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Glassmorphic Chat</title>
    <style>
        body { margin: 0; background: linear-gradient(45deg, #0f172a, #334155, #1e293b); background-size: 400% 400%; animation: gradient 15s ease infinite; height: 100vh; display: flex; align-items: center; justify-content: center; font-family: sans-serif; }
        @keyframes gradient { 0% { background-position: 0% 50%; } 50% { background-position: 100% 50%; } 100% { background-position: 0% 50%; } }
        .window { width: 400px; height: 600px; background: rgba(255, 255, 255, 0.1); backdrop-filter: blur(10px); border: 1px solid rgba(255, 255, 255, 0.2); border-radius: 20px; box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37); display: flex; flex-direction: column; overflow: hidden; }
        .header { padding: 1.5rem; border-bottom: 1px solid rgba(255, 255, 255, 0.1); color: white; display: flex; justify-content: space-between; }
        #chats { flex: 1; padding: 1.5rem; overflow-y: auto; display: flex; flex-direction: column; gap: 0.75rem; }
        .msg { padding: 0.75rem 1rem; border-radius: 15px; max-width: 80%; font-size: 0.9rem; }
        .msg.in { align-self: flex-start; background: rgba(255, 255, 255, 0.1); color: white; }
        .msg.out { align-self: flex-end; background: #38bdf8; color: white; }
        .input-bar { padding: 1.5rem; background: rgba(0,0,0,0.1); }
        input { width: 100%; border: none; background: rgba(255,255,255,0.05); padding: 0.75rem 1rem; border-radius: 10px; color: white; outline: none; }
    </style>
</head>
<body>
    <div class="window">
        <div class="header"><span>WebSocket Live</span><div id="status" style="width: 10px; height: 10px; background: #ef4444; border-radius: 50%;"></div></div>
        <div id="chats"></div>
        <div class="input-bar"><input type="text" id="input" placeholder="Say something..."></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const chats = document.getElementById('chats');
        const input = document.getElementById('input');
        const status = document.getElementById('status');
        let socket;
        function addMsg(type, m) {
            const d = document.createElement('div'); d.className = 'msg '+type; d.textContent = m;
            chats.appendChild(d); chats.scrollTop = chats.scrollHeight;
        }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => status.style.background = '#10b981';
            socket.onclose = () => { status.style.background = '#ef4444'; setTimeout(connect, 3000); };
            socket.onmessage = e => addMsg('in', e.data);
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); addMsg('out', input.value); input.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"minimal": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Minimalist</title>
    <style>
        body { font-family: system-ui; background: #fff; color: #000; padding: 4rem; max-width: 600px; margin: 0 auto; line-height: 1.6; }
        h1 { font-weight: 900; font-size: 3rem; margin: 0; }
        .status { color: #f00; font-weight: bold; }
        #output { margin-top: 2rem; border-top: 1px solid #eee; padding-top: 1rem; }
        input { width: 100%; border: 1px solid #eee; padding: 1rem; font-size: 1rem; margin-top: 1rem; outline: none; }
        input:focus { border-color: #000; }
    </style>
</head>
<body>
    <h1>Minimal.</h1>
    <div id="status" class="status">connecting...</div>
    <div id="output"></div>
    <input type="text" id="input" placeholder="Type something.">
    <script>
        const wsUrl = "{{.WSURL}}";
        const o = document.getElementById('output');
        const i = document.getElementById('input');
        const s = document.getElementById('status');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => { s.textContent = 'online'; s.style.color = '#0c0'; };
            socket.onclose = () => { s.textContent = 'offline'; s.style.color = '#f00'; setTimeout(connect, 3000); };
            socket.onmessage = e => { const d = document.createElement('div'); d.textContent = e.data; o.prepend(d); };
        }
        i.onkeypress = e => { if (e.key === 'Enter' && i.value) { socket.send(i.value); i.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"iot": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - IoT Sensor Monitor</title>
    <style>
        body { background: #1a1a1a; color: #fff; font-family: 'Segoe UI', sans-serif; margin: 0; padding: 2rem; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 2rem; }
        .sensor { background: #2d2d2d; border-radius: 15px; padding: 2rem; text-align: center; box-shadow: 0 4px 6px rgba(0,0,0,0.3); }
        .gauge { width: 150px; height: 150px; margin: 0 auto 1.5rem; position: relative; }
        .gauge-svg { transform: rotate(-90deg); }
        .gauge-bg { fill: none; stroke: #333; stroke-width: 4; }
        .gauge-fill { fill: none; stroke: #00ff88; stroke-width: 4; stroke-dasharray: 100 100; transition: stroke-dasharray 0.5s ease; }
        .value { font-size: 2.5rem; font-weight: bold; }
        .label { color: #888; text-transform: uppercase; letter-spacing: 2px; }
        #logs { margin-top: 2rem; background: #000; height: 150px; overflow-y: auto; padding: 1rem; font-family: monospace; font-size: 0.8rem; opacity: 0.7; }
    </style>
</head>
<body>
    <h1 style="margin-bottom: 2.5rem;">Network Sensor Node #1</h1>
    <div class="grid">
        <div class="sensor"><div class="gauge"><svg class="gauge-svg" viewBox="0 0 36 36"><path class="gauge-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/><path id="temp-gauge" class="gauge-fill" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/></svg></div><div class="value" id="temp">--°C</div><div class="label">Temperature</div></div>
        <div class="sensor"><div class="gauge"><svg class="gauge-svg" viewBox="0 0 36 36"><path class="gauge-bg" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/><path id="hum-gauge" class="gauge-fill" style="stroke: #00c2ff;" d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"/></svg></div><div class="value" id="hum">--%</div><div class="label">Humidity</div></div>
    </div>
    <div id="logs"></div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => {
                const d = document.createElement('div'); d.textContent = e.data; logs.prepend(d);
                if (!isNaN(e.data)) {
                    const val = Math.min(100, Math.max(0, parseFloat(e.data)));
                    document.getElementById('temp').textContent = val.toFixed(1) + '°C';
                    document.getElementById('temp-gauge').style.strokeDasharray = val + ' 100';
                }
            };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        connect();
    </script>
</body>
</html>`,

	"trading": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - High Speed Trading</title>
    <style>
        body { background: #0b0e11; color: #eaecef; font-family: 'Roboto', sans-serif; margin: 0; padding: 1rem; overflow: hidden; height: 100vh; display: flex; flex-direction: column; }
        .ticker { display: flex; gap: 2rem; background: #1e2329; padding: 0.5rem 1rem; border-bottom: 1px solid #2b3139; white-space: nowrap; }
        .symbol { font-weight: bold; } .up { color: #02c076; } .down { color: #f84960; }
        .main { flex: 1; display: grid; grid-template-columns: 1fr 300px; gap: 1px; background: #2b3139; }
        .chart { background: #0b0e11; padding: 2rem; }
        #orderbook { background: #161a1e; padding: 1rem; font-family: monospace; font-size: 0.85rem; overflow-y: auto; }
        .row { display: grid; grid-template-columns: 1fr 1fr 1fr; margin-bottom: 2px; }
        #input-area { background: #1e2329; padding: 1rem; border-top: 1px solid #2b3139; }
        input { background: #2b3139; border: none; padding: 0.75rem; color: #fff; width: 300px; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="ticker">
        <div><span class="symbol">XWBS/USD</span> <span id="price" class="up">0.0000</span></div>
        <div><span class="symbol">BTC/USD</span> <span class="down">64,231.50</span></div>
    </div>
    <div class="main">
        <div class="chart"><h1>Market Overview</h1><div id="graph" style="height: 300px; border-bottom: 2px solid #2b3139;"></div></div>
        <div id="orderbook"><div style="color: #848e9c; margin-bottom: 10px;">ORDER BOOK</div></div>
    </div>
    <div id="input-area"><input type="text" id="input" placeholder="Execute Market Order..."></div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const book = document.getElementById('orderbook');
        const price = document.getElementById('price');
        const input = document.getElementById('input');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => {
                const row = document.createElement('div'); row.className = 'row';
                const p = parseFloat(e.data);
                if (!isNaN(p)) {
                    price.textContent = p.toFixed(4);
                    price.className = p > 50 ? 'up' : 'down';
                }
                row.innerHTML = '<span>'+new Date().toLocaleTimeString()+'</span> <span class="up">'+e.data+'</span> <span>1.0</span>';
                book.prepend(row); if(book.children.length > 30) book.lastChild.remove();
            };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"social": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Social Stream</title>
    <style>
        body { background: #f0f2f5; font-family: system-ui; padding: 2rem; display: flex; justify-content: center; }
        #stream { width: 500px; display: flex; flex-direction: column; gap: 1rem; }
        .post { background: #fff; border-radius: 8px; box-shadow: 0 1px 2px rgba(0,0,0,0.1); padding: 1rem; animation: slideIn 0.3s ease-out; }
        @keyframes slideIn { from { opacity:0; transform:translateY(20px); } to { opacity:1; transform:translateY(0); } }
        .author { font-weight: bold; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 10px; }
        .avatar { width: 40px; height: 40px; border-radius: 50%; background: #ccc; }
        .content { color: #1c1e21; margin-bottom: 1rem; }
        .footer { border-top: 1px solid #ebedf0; padding-top: 0.5rem; display: flex; gap: 1rem; color: #65676b; font-size: 0.9rem; }
        #composer { background: #fff; border-radius: 8px; padding: 1rem; margin-bottom: 1rem; box-shadow: 0 1px 2px rgba(0,0,0,0.1); }
        input { width: 100%; height: 40px; background: #f0f2f5; border: none; border-radius: 20px; padding: 0 1rem; outline: none; box-sizing: border-box; }
    </style>
</head>
<body>
    <div id="stream">
        <div id="composer"><input type="text" id="input" placeholder="What's on your mind?"></div>
        <div id="posts"></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const posts = document.getElementById('posts');
        const input = document.getElementById('input');
        let socket;
        function addPost(m) {
            const p = document.createElement('div'); p.className = 'post';
            p.innerHTML = '<div class="author"><div class="avatar"></div>Anonymous User</div><div class="content">'+m+'</div><div class="footer"><span>Like</span><span>Comment</span><span>Share</span></div>';
            posts.prepend(p);
        }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => addPost(e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); addPost(input.value); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"cyberpunk": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Cyberpunk Edge</title>
    <style>
        body { background: #050505; color: #ff0055; font-family: sans-serif; text-transform: uppercase; letter-spacing: 2px; overflow: hidden; height: 100vh; padding: 2rem; margin: 0; box-sizing: border-box; }
        h1 { color: #00ffff; text-shadow: 0 0 10px #00ffff; font-size: 4rem; margin: 0; }
        .border { position: absolute; border: 2px solid #ff0055; width: calc(100% - 4rem); height: calc(100% - 4rem); pointer-events: none; }
        #console { height: 60%; overflow-y: auto; margin-top: 2rem; border-left: 5px solid #00ffff; padding-left: 1rem; }
        .line { margin-bottom: 0.5rem; text-shadow: 0 0 5px #ff0055; }
        input { background: #ff0055; color: #000; border: none; padding: 1rem; width: 400px; font-weight: 900; margin-top: 2rem; outline: none; }
        input:focus { background: #00ffff; }
    </style>
</head>
<body>
    <div class="border"></div>
    <h1>XWEBS.CORE</h1>
    <div id="status" style="color: #00ffff">LINK: ESTABLISHING...</div>
    <div id="console"></div>
    <input type="text" id="input" placeholder="SEND DATA PKT">
    <script>
        const wsUrl = "{{.WSURL}}";
        const c = document.getElementById('console');
        const s = document.getElementById('status');
        const i = document.getElementById('input');
        let socket;
        function log(m) { const l = document.createElement('div'); l.className='line'; l.textContent='> ' + m; c.appendChild(l); c.scrollTop=c.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => s.textContent = 'LINK: ONLINE';
            socket.onclose = () => { s.textContent = 'LINK: CRITICAL ERROR'; setTimeout(connect, 3000); };
            socket.onmessage = e => log(e.data);
        }
        i.onkeypress = e => { if (e.key==='Enter' && i.value) { socket.send(i.value); log('OUT: ' + i.value); i.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"neumorphism": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Neumorphic UI</title>
    <style>
        body { background: #e0e5ec; font-family: sans-serif; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .container { width: 350px; background: #e0e5ec; border-radius: 50px; padding: 3rem; box-shadow: 20px 20px 60px #bec3c9, -20px -20px 60px #ffffff; }
        .well { height: 200px; margin-bottom: 2rem; border-radius: 20px; box-shadow: inset 6px 6px 12px #bec3c9, inset -6px -6px 12px #ffffff; padding: 1rem; overflow-y: auto; color: #444; }
        input { width: 100%; border: none; padding: 1rem; border-radius: 10px; background: #e0e5ec; box-shadow: 6px 6px 12px #bec3c9, -6px -6px 12px #ffffff; outline: none; margin-bottom: 2rem; }
        button { width: 100%; border: none; padding: 1rem; border-radius: 10px; background: #e0e5ec; box-shadow: 6px 6px 12px #bec3c9, -6px -6px 12px #ffffff; cursor: pointer; color: #38bdf8; font-weight: bold; }
        button:active { box-shadow: inset 4px 4px 8px #bec3c9, inset -4px -4px 8px #ffffff; }
    </style>
</head>
<body>
    <div class="container">
        <h2 style="text-align: center; color: #444; margin-bottom: 2rem;">Soft Hub</h2>
        <div id="output" class="well"></div>
        <input type="text" id="input" placeholder="Type a message">
        <button id="send">SEND MESSAGE</button>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const o = document.getElementById('output');
        const i = document.getElementById('input');
        const btn = document.getElementById('send');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => { const d = document.createElement('div'); d.textContent = e.data; o.appendChild(d); o.scrollTop=o.scrollHeight; };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        function send() { if (i.value) { socket.send(i.value); i.value=''; } }
        btn.onclick = send; i.onkeypress = e => { if (e.key==='Enter') send(); };
        connect();
    </script>
</body>
</html>`,

	"hacker": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>xwebs - Matrix Hacker</title>
    <style>
        body { background: black; color: #00ff00; font-family: 'Courier New', monospace; margin: 0; overflow: hidden; }
        canvas { position: absolute; top: 0; left: 0; z-index: -1; }
        .ui { padding: 2rem; background: rgba(0,0,0,0.8); height: 100vh; width: 100vw; display: flex; flex-direction: column; box-sizing: border-box; }
        #logs { flex: 1; overflow-y: auto; border: 1px solid #00ff00; padding: 1rem; margin-bottom: 1rem; }
        .input-group { display: flex; align-items: center; border: 1px solid #00ff00; padding: 0.5rem; }
        input { background: none; border: none; color: #00ff00; flex: 1; font-size: 1.2rem; outline: none; }
    </style>
</head>
<body>
    <canvas id="canvas"></canvas>
    <div class="ui">
        <h1 style="margin: 0 0 1rem 0;">XWEBS_TERMINAL_V1.0</h1>
        <div id="logs"></div>
        <div class="input-group"><span>[HACK@XWEBS] ~ $</span><input type="text" id="input" autofocus></div>
    </div>
    <script>
        const canvas = document.getElementById('canvas');
        const ctx = canvas.getContext('2d');
        canvas.height = window.innerHeight; canvas.width = window.innerWidth;
        const matrix = "XWEBS010101ABCDEFGHIJKLMNOPQRSTUVWXYZ";
        const columns = canvas.width / 20;
        const drops = [];
        for (let x = 0; x < columns; x++) drops[x] = 1;
        function draw() {
            ctx.fillStyle = "rgba(0, 0, 0, 0.05)"; ctx.fillRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = "#0F0"; ctx.font = "20px arial";
            for (let i = 0; i < drops.length; i++) {
                const text = matrix[Math.floor(Math.random() * matrix.length)];
                ctx.fillText(text, i * 20, drops[i] * 20);
                if (drops[i] * 20 > canvas.height && Math.random() > 0.975) drops[i] = 0;
                drops[i]++;
            }
        }
        setInterval(draw, 35);
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function print(m) { const d = document.createElement('div'); d.textContent = 'ACCESS_GRANTED: ' + m; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => print(e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); print('PKT_SENT: ' + input.value); input.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"ios": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Mobile Messenger</title>
    <style>
        body { margin: 0; background: #fff; font-family: -apple-system, sans-serif; display: flex; justify-content: center; }
        .phone { width: 375px; height: 100vh; border: 1px solid #eee; display: flex; flex-direction: column; }
        .nav { height: 90px; background: #f9f9f9; border-bottom: 1px solid #ddd; display: flex; align-items: flex-end; padding: 1rem; box-sizing: border-box; }
        .nav h1 { margin: 0; font-size: 1.2rem; }
        #chats { flex: 1; overflow-y: auto; padding: 1rem; background: #fff; display: flex; flex-direction: column; gap: 0.5rem; }
        .bubble { padding: 0.75rem 1rem; border-radius: 20px; max-width: 80%; font-size: 1rem; }
        .in { background: #e9e9eb; color: black; align-self: flex-start; }
        .out { background: #007aff; color: white; align-self: flex-end; }
        .input-bar { height: 80px; padding: 1rem; border-top: 1px solid #ddd; background: #f9f9f9; display: flex; gap: 10px; align-items: center; box-sizing: border-box; }
        input { flex: 1; height: 36px; border: 1px solid #ddd; border-radius: 18px; padding: 0 15px; outline: none; }
    </style>
</head>
<body>
    <div class="phone">
        <div class="nav"><h1>iWebs Messenger</h1></div>
        <div id="chats"></div>
        <div class="input-bar"><input type="text" id="input" placeholder="iMessage"></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const chats = document.getElementById('chats');
        const input = document.getElementById('input');
        let socket;
        function add(m, type) { const b = document.createElement('div'); b.className = 'bubble ' + type; b.textContent = m; chats.appendChild(b); chats.scrollTop = chats.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => add(e.data, 'in');
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); add(input.value, 'out'); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"win95": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Windows 95 Client</title>
    <style>
        body { background: #008080; font-family: 'MS Sans Serif', Tahoma, sans-serif; margin: 0; padding: 2rem; }
        .window { background: #c0c0c0; border: 2px solid; border-color: #fff #808080 #808080 #fff; width: 500px; }
        .title-bar { background: #000080; color: #fff; padding: 3px 5px; font-weight: bold; display: flex; align-items: center; }
        .content { padding: 1rem; }
        #logs { background: #fff; height: 200px; border: 2px solid; border-color: #808080 #fff #fff #808080; padding: 0.5rem; overflow-y: scroll; margin-bottom: 1rem; font-family: 'Courier New', monospace; font-size: 0.8rem; }
        .input-group { display: flex; gap: 10px; }
        input { border: 2px solid; border-color: #808080 #fff #fff #808080; flex: 1; padding: 4px; outline: none; }
        button { background: #c0c0c0; border: 2px solid; border-color: #fff #808080 #808080 #fff; padding: 4px 10px; cursor: pointer; outline: none; }
        button:active { border-color: #808080 #fff #fff #808080; }
    </style>
</head>
<body>
    <div class="window">
        <div class="title-bar"><span>XWEBS Terminal</span></div>
        <div class="content">
            <div id="logs">Connecting to server...</div>
            <div class="input-group"><input type="text" id="input"><button id="btn">Send</button></div>
        </div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        const btn = document.getElementById('btn');
        let socket;
        function log(m) { const d = document.createElement('div'); d.textContent = m; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => log('System: Connection successful');
            socket.onclose = () => { log('System: Connection lost. Reconnecting...'); setTimeout(connect, 2000); };
            socket.onmessage = e => log('Incoming: ' + e.data);
        }
        function send() { if (input.value) { socket.send(input.value); log('Outgoing: ' + input.value); input.value = ''; } }
        btn.onclick = send; input.onkeypress = e => { if (e.key === 'Enter') send(); };
        connect();
    </script>
</body>
</html>`,

	"saas": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - SaaS Landing Page</title>
    <style>
        body { font-family: 'Inter', sans-serif; margin: 0; color: #1e293b; background: white; }
        .navbar { padding: 1.5rem 5rem; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #f1f5f9; }
        .hero { padding: 5rem; text-align: center; }
        .hero h1 { font-size: 4rem; font-weight: 800; letter-spacing: -2px; margin: 0 0 1rem 0; }
        .hero p { font-size: 1.25rem; color: #64748b; max-width: 600px; margin: 0 auto 3rem auto; }
        .demo { width: 900px; margin: 0 auto; background: #0f172a; border-radius: 1rem; height: 400px; display: flex; flex-direction: column; overflow: hidden; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.25); }
        .demo-head { padding: 1rem; border-bottom: 1px solid #334155; display: flex; gap: 8px; }
        .dot { width: 12px; height: 12px; border-radius: 50%; }
        #logs { flex: 1; padding: 2rem; color: #38bdf8; font-family: monospace; overflow-y: auto; text-align: left; }
        .demo-foot { padding: 1rem; border-top: 1px solid #334155; }
        input { width: 100%; background: transparent; border: none; color: #fff; font-size: 1rem; outline: none; }
    </style>
</head>
<body>
    <div class="navbar"><div style="font-weight: 900; font-size: 1.5rem;">XWEBS</div><div style="color: #38bdf8; font-weight: 600;">Pricing &rarr;</div></div>
    <div class="hero">
        <h1>Connect anything.</h1>
        <p>The enterprise-grade WebSocket engine that scales with your ambition. High throughput, ultra-low latency.</p>
        <div class="demo">
            <div class="demo-head"><div class="dot" style="background: #ff5f56;"></div><div class="dot" style="background: #ffbd2e;"></div><div class="dot" style="background: #27c93f;"></div><span style="color: #64748b; margin-left: auto;">live-terminal</span></div>
            <div id="logs">Waiting for connection...</div>
            <div class="demo-foot"><input type="text" id="input" placeholder="Type a message to see xwebs in action..."></div>
        </div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function print(m) { const d = document.createElement('div'); d.textContent = '> ' + m; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => { logs.innerHTML = '<div style="color: #10b981">STREAM ESTABLISHED. READY.</div>'; };
            socket.onmessage = e => print(e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); print('EMIT: ' + input.value); input.value = ''; } };
        connect();
    </script>
</body>
</html>`,

	"lab": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Lab Notebook</title>
    <style>
        body { background: #fdfdfd; background-image: radial-gradient(#e0e0e0 1px, transparent 1px); background-size: 20px 20px; font-family: 'Georgia', serif; padding: 4rem; color: #333; }
        .page { background: white; max-width: 800px; margin: 0 auto; box-shadow: 0 0 10px rgba(0,0,0,0.05); padding: 4rem; border: 1px solid #ddd; min-height: 800px; position: relative; }
        h1 { border-bottom: 2px solid #555; padding-bottom: 1rem; margin-bottom: 2rem; font-style: italic; }
        .meta { display: flex; justify-content: space-between; margin-bottom: 3rem; font-size: 0.9rem; color: #666; }
        #logs { display: flex; flex-direction: column; gap: 1rem; }
        .entry { border-left: 2px solid #ccc; padding-left: 1rem; font-size: 1.1rem; }
        .time { font-size: 0.8rem; color: #888; display: block; }
        .input-area { position: absolute; bottom: 4rem; width: calc(100% - 8rem); border-top: 1px dashed #ccc; padding-top: 1rem; }
        input { width: 100%; border: none; font-size: 1.2rem; font-family: inherit; font-style: italic; color: #007aff; outline: none; }
    </style>
</head>
<body>
    <div class="page">
        <h1>Experimental Log: WebSocket Connectivity</h1>
        <div class="meta"><span>Subject: XWEBS_ALPHA_01</span><span id="date"></span></div>
        <div id="logs"></div>
        <div class="input-area"><input type="text" id="input" placeholder="Annotate log..."></div>
    </div>
    <script>
        document.getElementById('date').textContent = new Date().toLocaleDateString();
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function print(m) { const e = document.createElement('div'); e.className='entry'; e.innerHTML='<span class="time">' + new Date().toLocaleTimeString() + '</span>' + m; logs.appendChild(e); }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => print('System: Subroutine initiated.');
            socket.onmessage = e => print('Observed: ' + e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); print('Note: ' + input.value); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"space": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Space HUD</title>
    <style>
        body { background: #000; margin: 0; overflow: hidden; font-family: 'Space Mono', monospace; color: #fff; }
        .stars { position: absolute; width: 100%; height: 100%; background: radial-gradient(circle at center, #111 0%, #000 100%); z-index: -1; }
        .hud { padding: 4rem; height: 100vh; display: flex; flex-direction: column; box-sizing: border-box; }
        .grid { flex: 1; display: grid; grid-template-columns: 1fr 1fr; gap: 4rem; }
        .box { border: 1px solid rgba(0, 255, 255, 0.3); background: rgba(0, 255, 255, 0.05); padding: 2rem; position: relative; }
        .box::after { content: ''; position: absolute; top: -5px; left: -5px; width: 20px; height: 20px; border-top: 2px solid cyan; border-left: 2px solid cyan; }
        #logs { height: 100%; overflow-y: auto; color: cyan; font-size: 0.9rem; }
        .radar { width: 200px; height: 200px; border: 2px solid cyan; border-radius: 50%; margin: 0 auto; position: relative; overflow: hidden; }
        .radar-sweep { position: absolute; width: 100%; height: 100%; top: 0; left: 0; background: conic-gradient(from 0deg, rgba(0, 255, 255, 0.5), transparent 90deg); border-radius: 50%; animation: sweep 2s linear infinite; }
        @keyframes sweep { to { transform: rotate(360deg); } }
        input { background: none; border: 1px solid cyan; color: cyan; padding: 1rem; margin-top: 2rem; width: 100%; box-sizing: border-box; font-family: inherit; }
    </style>
</head>
<body>
    <div class="stars"></div>
    <div class="hud">
        <h1 style="color: cyan; margin: 0 0 2rem 0; font-size: 2rem;">DEEP_SPACE_COMM_LINK</h1>
        <div class="grid">
            <div class="box"><div id="logs">SIGNAL STATUS: WEAK...</div></div>
            <div class="box" style="display: flex; flex-direction: column; items: center; justify-content: center;">
                <div class="radar"><div class="radar-sweep"></div></div>
                <div style="text-align: center; margin-top: 2rem;">SCANNING FOR INBOUND PACKETS</div>
            </div>
        </div>
        <input type="text" id="input" placeholder="TRANSMIT_TO_STATION">
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => logs.innerHTML = 'STREAM: ONLINE<br>STATION_STABLE';
            socket.onmessage = e => { const d = document.createElement('div'); d.textContent = 'RECV: ' + e.data; logs.prepend(d); };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); const d = document.createElement('div'); d.textContent = 'SEND: ' + input.value; logs.prepend(d); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"gallery": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Art Gallery</title>
    <style>
        body { background: #111; color: white; font-family: 'Didot', serif; margin: 0; padding: 4rem; text-align: center; }
        h1 { font-size: 4rem; font-weight: 100; color: #888; }
        .frame { border: 20px solid #222; max-width: 900px; margin: 4rem auto; padding: 4rem; background: #000; min-height: 400px; display: flex; flex-direction: column; justify-content: center; align-items: center; }
        #canvas { font-size: 3rem; color: #fff; line-height: 1.2; text-transform: uppercase; letter-spacing: 0.2em; font-style: italic; }
        .footer { margin-top: 4rem; color: #555; }
        input { background: none; border-bottom: 1px solid #333; border-top: none; border-left: none; border-right: none; color: #fff; padding: 1rem; width: 400px; text-align: center; font-family: inherit; font-size: 1.5rem; outline: none; transition: border-color 1s; }
        input:focus { border-color: #888; }
    </style>
</head>
<body>
    <h1>Digital Ephemera</h1>
    <div class="frame"><div id="canvas">Waiting for signal</div></div>
    <input type="text" id="input" placeholder="contribute to the void">
    <div class="footer">&copy; 2026 xwebs exhibition</div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const canvas = document.getElementById('canvas');
        const input = document.getElementById('input');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => { canvas.style.opacity = 0; setTimeout(() => { canvas.textContent = e.data; canvas.style.opacity = 1; }, 500); };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); input.value=''; } };
        canvas.style.transition = 'opacity 0.5s';
        connect();
    </script>
</body>
</html>`,

	"corporate": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Enterprise Hub</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, sans-serif; background: #f4f7f6; color: #2c3e50; margin: 0; }
        .header { background: #ffffff; padding: 1.5rem 3rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); display: flex; justify-content: space-between; align-items: center; }
        .hero { background: linear-gradient(135deg, #2c3e50, #4ca1af); color: white; padding: 4rem 3rem; text-align: center; }
        .panel { max-width: 1000px; margin: -2rem auto 4rem; background: white; border-radius: 8px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); padding: 2rem; }
        .status-bar { display: flex; gap: 2rem; margin-bottom: 2rem; border-bottom: 1px solid #eee; padding-bottom: 1rem; }
        .stat { flex: 1; } .stat div { font-weight: bold; font-size: 1.5rem; }
        #logs { height: 300px; overflow-y: auto; background: #f8f9fa; border: 1px solid #e1e4e8; border-radius: 4px; padding: 1rem; font-family: 'Consolas', monospace; font-size: 0.9rem; }
        .input-group { margin-top: 1.5rem; display: flex; gap: 10px; }
        input { flex: 1; padding: 0.75rem; border: 1px solid #ddd; border-radius: 4px; outline: none; }
        button { background: #4ca1af; color: white; border: none; padding: 0 2rem; border-radius: 4px; cursor: pointer; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header"><div style="font-size: 1.5rem; font-weight: bold; color: #4ca1af;">XWEBS_CORP</div><div id="status">OFFLINE</div></div>
    <div class="hero"><h1>WebSocket Infrastructure for Teams</h1><p>Secure, Managed, and Scalable Digital Communication Hub</p></div>
    <div class="panel">
        <div class="status-bar">
            <div class="stat"><label>Uptime</label><div style="color: #4ca1af;">99.9%</div></div>
            <div class="stat"><label>Nodes</label><div id="nodes">--</div></div>
            <div class="stat"><label>Inbound</label><div id="count">0</div></div>
        </div>
        <div id="logs">Establishing secure handshake...</div>
        <div class="input-group"><input type="text" id="input"><button id="btn">TRANSMIT</button></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        const btn = document.getElementById('btn');
        let count = 0;
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => { document.getElementById('status').textContent = 'CONNECTED'; document.getElementById('nodes').textContent = 'ACTIVE'; logs.innerHTML=''; };
            socket.onmessage = e => { count++; document.getElementById('count').textContent = count; const d = document.createElement('div'); d.textContent = '['+new Date().toLocaleTimeString()+'] RECV: ' + e.data; logs.prepend(d); };
            socket.onclose = () => { document.getElementById('status').textContent = 'RETRYING'; setTimeout(connect, 3000); };
        }
        function send() { if (input.value) { socket.send(input.value); const d = document.createElement('div'); d.textContent = '['+new Date().toLocaleTimeString()+'] SEND: ' + input.value; logs.prepend(d); input.value=''; } }
        btn.onclick = send; input.onkeypress = e => { if (e.key === 'Enter') send(); };
        connect();
    </script>
</body>
</html>`,

	"material": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Material Design</title>
    <style>
        body { margin: 0; font-family: 'Roboto', sans-serif; background: #f5f5f5; }
        .app-bar { background: #6200ee; color: white; height: 64px; padding: 0 24px; display: flex; align-items: center; box-shadow: 0 2px 4px rgba(0,0,0,0.2); }
        .card { width: 90%; max-width: 600px; margin: 32px auto; background: white; border-radius: 4px; box-shadow: 0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24); padding: 24px; }
        #logs { margin-top: 16px; border-top: 1px solid #eee; padding-top: 16px; height: 300px; overflow-y: auto; }
        .input-group { margin-top: 24px; position: relative; }
        input { width: 100%; border: none; border-bottom: 1px solid #9e9e9e; padding: 8px 0; font-size: 16px; outline: none; transition: border-bottom 0.2s; }
        input:focus { border-bottom: 2px solid #6200ee; }
        .fab { position: fixed; bottom: 32px; right: 32px; width: 56px; height: 56px; background: #03dac6; border-radius: 50%; box-shadow: 0 3px 5px rgba(0,0,0,0.2); display: flex; align-items: center; justify-content: center; cursor: pointer; color: white; font-size: 24px; }
    </style>
</head>
<body>
    <div class="app-bar"><h2>XWEBS - Material Client</h2></div>
    <div class="card">
        <h3>Live Stream</h3>
        <div id="logs">Connecting...</div>
        <div class="input-group"><input type="text" id="input" placeholder="Message content"></div>
    </div>
    <div class="fab" id="btn">+</div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        const btn = document.getElementById('btn');
        let socket;
        function log(m) { const d = document.createElement('div'); d.textContent = m; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => { logs.innerHTML = ''; log('Socket opened.'); };
            socket.onmessage = e => log('Incoming: ' + e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        function send() { if (input.value) { socket.send(input.value); log('Sent: ' + input.value); input.value=''; } }
        btn.onclick = send; input.onkeypress = e => { if (e.key === 'Enter') send(); };
        connect();
    </script>
</body>
</html>`,

	"retro": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Retro Dashboard</title>
    <style>
        body { background: #222; color: #fff; font-family: 'Courier New', Courier, monospace; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
        .monitor { width: 600px; height: 400px; background: #000; border: 15px solid #444; border-radius: 30px; box-shadow: inset 0 0 20px rgba(0,255,0,0.2); padding: 1rem; display: flex; flex-direction: column; }
        .header { color: #0f0; border-bottom: 2px solid #0f0; padding-bottom: 5px; margin-bottom: 10px; font-weight: bold; }
        #logs { flex: 1; overflow-y: auto; color: #0f0; font-size: 1.2rem; }
        .input-line { color: #0f0; display: flex; }
        input { background: none; border: none; color: #0f0; font-family: inherit; font-size: 1.2rem; flex: 1; outline: none; margin-left: 5px; }
    </style>
</head>
<body>
    <div class="monitor">
        <div class="header">XWEBS_TERMINAL_V0.1.A</div>
        <div id="logs">READY.</div>
        <div class="input-line"><span>></span><input type="text" id="input"></div>
    </div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => { const d = document.createElement('div'); d.textContent = e.data; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; };
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key === 'Enter' && input.value) { socket.send(input.value); input.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"minimal_neon": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Minimal Neon</title>
    <style>
        body { background: #000; color: #fff; font-family: sans-serif; height: 100vh; display: flex; align-items: center; justify-content: center; flex-direction: column; margin: 0; }
        #output { font-size: 5vw; font-weight: 900; text-shadow: 0 0 10px #38bdf8, 0 0 20px #38bdf8; text-align: center; }
        input { position: fixed; bottom: 4rem; background: none; border: none; border-bottom: 1px solid #38bdf8; color: #38bdf8; font-size: 1.5rem; text-align: center; outline: none; padding: 1rem; width: 60%; }
    </style>
</head>
<body>
    <div id="output">LINK_READY</div>
    <input type="text" id="input" placeholder="BROADCAST">
    <script>
        const wsUrl = "{{.WSURL}}";
        const o = document.getElementById('output');
        const i = document.getElementById('input');
        let socket;
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onmessage = e => { o.style.color = '#38bdf8'; o.textContent = e.data; };
            socket.onclose = () => { o.style.color = '#f00'; o.textContent = 'LINK_LOST'; setTimeout(connect, 3000); };
        }
        i.onkeypress = e => { if (e.key==='Enter' && i.value) { socket.send(i.value); i.value=''; } };
        connect();
    </script>
</body>
</html>`,

	"terminal_amber": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>xwebs - Vintage Amber</title>
    <style>
        body { background: #1a1005; color: #ffb000; font-family: 'Courier New', monospace; padding: 2rem; margin: 0; height: 100vh; box-sizing: border-box; display: flex; flex-direction: column; text-shadow: 0 0 5px #ffb000; }
        #logs { flex: 1; overflow-y: auto; overflow-x: hidden; }
        .input-line { margin-top: 1rem; display: flex; }
        input { background: none; border: none; color: #ffb000; font-family: inherit; font-size: 1.2rem; flex: 1; outline: none; caret-color: #ffb000; }
    </style>
</head>
<body>
    <div id="logs">ESTABLISHING TELETYPE LINK...</div>
    <div class="input-line"><span>*</span><input type="text" id="input" autofocus></div>
    <script>
        const wsUrl = "{{.WSURL}}";
        const logs = document.getElementById('logs');
        const input = document.getElementById('input');
        let socket;
        function print(m) { const d = document.createElement('div'); d.textContent = m; logs.appendChild(d); logs.scrollTop = logs.scrollHeight; }
        function connect() {
            socket = new WebSocket(wsUrl);
            socket.onopen = () => print('CONNECTED TO CORE.');
            socket.onmessage = e => print('<<< ' + e.data);
            socket.onclose = () => setTimeout(connect, 3000);
        }
        input.onkeypress = e => { if (e.key==='Enter' && input.value) { socket.send(input.value); print('>>> ' + input.value); input.value=''; } };
        connect();
    </script>
</body>
</html>`,
}
