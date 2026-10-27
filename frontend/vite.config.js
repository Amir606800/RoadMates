import { readFileSync } from 'node:fs'
import { execFileSync } from 'node:child_process'
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = { ...loadEnv(mode, process.cwd(), 'VITE_'), ...process.env }
  const version = env.VITE_APP_VERSION || readFileSync(new URL('../VERSION', import.meta.url), 'utf8').trim()
  let commit = env.VITE_APP_COMMIT
  if (!commit) {
    try { commit = execFileSync('git', ['rev-parse', '--short=7', 'HEAD'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim() }
    catch { commit = 'unknown' }
  }
  const metadata = { app_version: version, commit }
  const proxy = { target: env.VITE_PROXY_TARGET || 'http://localhost:8080', changeOrigin: true }
  return {
    plugins: [react(), {
      name: 'roadmates-version',
      generateBundle() { this.emitFile({ type: 'asset', fileName: 'version.json', source: JSON.stringify(metadata) + '\n' }) },
      configureServer(server) {
        server.middlewares.use('/version.json', (_req, res) => {
          res.setHeader('Content-Type', 'application/json')
          res.setHeader('Cache-Control', 'no-cache')
          res.end(JSON.stringify(metadata))
        })
      },
    }],
    define: {
      'import.meta.env.VITE_APP_VERSION': JSON.stringify(version),
      'import.meta.env.VITE_APP_COMMIT': JSON.stringify(commit),
    },
    server: {
      host: true,
      port: 5173,
      strictPort: true,
      proxy: { '/api': proxy, '/healthz': proxy, '/readyz': proxy },
    },
  }
})
