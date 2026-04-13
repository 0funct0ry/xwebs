import { useState, useEffect } from 'react'

function App() {
  const [status, setStatus] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const fetchStatus = async () => {
    try {
      const response = await fetch('/api/status')
      if (!response.ok) throw new Error('Failed to fetch status')
      const data = await response.json()
      setStatus(data)
      setError(null)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 3000)
    return () => clearInterval(interval)
  }, [])

  if (loading && !status) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-sky-500"></div>
      </div>
    )
  }

  return (
    <div className="max-w-6xl mx-auto px-4 py-8 md:py-12">
      <header className="flex flex-col md:flex-row md:items-center justify-between mb-12 gap-6">
        <div>
          <h1 className="text-4xl font-black bg-gradient-to-r from-sky-400 to-indigo-400 bg-clip-text text-transparent tracking-tight">
            xwebs Server
          </h1>
          <p className="text-slate-400 mt-2 font-medium">Real-time monitoring and control dashboard</p>
        </div>
        <div className="flex items-center gap-4">
          <span className={`status-badge ${status?.status === 'running' ? 'status-running' : 'status-stopped'}`}>
            {status?.status || 'Unknown'}
          </span>
          <div className="text-sm font-mono text-slate-500 bg-slate-800/50 px-3 py-1 rounded-md border border-slate-700/50">
            v1.0.0
          </div>
        </div>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-12">
        <div className="glass-card p-6 border-l-4 border-l-sky-500">
          <p className="text-slate-400 text-sm font-bold uppercase tracking-wider mb-1">Uptime</p>
          <p className="text-3xl font-mono font-bold text-white">{status?.uptime || '0s'}</p>
        </div>
        <div className="glass-card p-6 border-l-4 border-l-emerald-500">
          <p className="text-slate-400 text-sm font-bold uppercase tracking-wider mb-1">Active Connections</p>
          <p className="text-3xl font-mono font-bold text-white">{status?.connections || 0}</p>
        </div>
        <div className="glass-card p-6 border-l-4 border-l-indigo-500">
          <p className="text-slate-400 text-sm font-bold uppercase tracking-wider mb-1">Memory Usage</p>
          <p className="text-3xl font-mono font-bold text-white">4.2MB</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <section>
          <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-sky-500"></span>
            Endpoints
          </h2>
          <div className="glass-card overflow-hidden">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-800/50">
                  <th className="p-4 text-xs font-bold uppercase text-slate-400 border-b border-slate-700/50">Path</th>
                  <th className="p-4 text-xs font-bold uppercase text-slate-400 border-b border-slate-700/50">Type</th>
                </tr>
              </thead>
              <tbody>
                {status?.paths?.map((path, i) => (
                  <tr key={i} className="hover:bg-slate-800/30 transition-colors">
                    <td className="p-4 font-mono text-emerald-400 text-sm">{path}</td>
                    <td className="p-4">
                      <span className="bg-slate-700/50 text-slate-300 px-2 py-0.5 rounded text-[10px] font-bold">WS</span>
                    </td>
                  </tr>
                ))}
                <tr className="hover:bg-slate-800/30 transition-colors border-t border-slate-800/50">
                  <td className="p-4 font-mono text-amber-400 text-sm">/api/*</td>
                  <td className="p-4">
                    <span className="bg-slate-700/50 text-slate-300 px-2 py-0.5 rounded text-[10px] font-bold">REST</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <section>
          <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
            Recent Clients
          </h2>
          <div className="glass-card p-4">
            {status?.connections > 0 ? (
              <p className="text-slate-400">Connection details available in the Clients tab.</p>
            ) : (
              <div className="py-8 text-center bg-slate-800/20 rounded-lg border border-dashed border-slate-700/50">
                <p className="text-slate-500">No active connections</p>
              </div>
            )}
          </div>
        </section>
      </div>

      <footer className="mt-24 pt-8 border-t border-slate-800/50 text-center text-slate-500 text-sm">
        <p>© 2026 xwebs. Premium WebSocket Toolchain.</p>
      </footer>
    </div>
  )
}

export default App
