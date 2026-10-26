import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import { createHash } from 'node:crypto'
import { readFileSync, readdirSync, renameSync, rmSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'

// A query string can still refer to a reused resource path in a desktop webview.
// Give each set of UI bytes its own directory, including relative CSS/JS URLs.
const uiSource = resolve(__dirname, 'public/one-browser')
const uiRevision = createHash('sha256')
for (const name of readdirSync(uiSource).sort()) {
  uiRevision.update(name).update(readFileSync(resolve(uiSource, name)))
}
const uiDirectory = 'one-browser-' + uiRevision.digest('hex').slice(0, 12)

function packagedUI() {
  return {
    name: 'one-browser-packaged-ui',
    apply: 'build' as const,
    writeBundle(options: { dir?: string }) {
      const output = resolve(__dirname, options.dir || 'dist')
      const source = resolve(output, 'one-browser')
      const document = resolve(source, 'index.html')
      const css = readFileSync(resolve(source, 'one-browser-current.css'), 'utf8')
      const marker = '/* One Browser 2.0.6: shared dark surfaces.'
      const darkOffset = css.indexOf(marker)
      if (darkOffset < 0) throw new Error('Missing packaged dark appearance rules')
      // Keep the critical dark rules atomic with the document. Even if a webview
      // supplies an old external sheet, it cannot omit these dark-only rules.
      const critical = css.slice(darkOffset)
      let html = readFileSync(document, 'utf8')
      html = html.replace('<html lang="zh-CN">', '<html lang="zh-CN" data-ui-revision="' + uiDirectory + '">')
      html = html.replace('</head>', '<style id="one-browser-dark-appearance">' + critical + '</style>\n</head>')
      writeFileSync(document, html)
      for (const name of readdirSync(output)) {
        if (/^one-browser-[a-f0-9]{12}$/.test(name)) rmSync(resolve(output, name), { recursive: true })
      }
      renameSync(source, resolve(output, uiDirectory))
    },
  }
}

const defaultDevPort = 5218

function resolveBoolean(rawValue: string | undefined, fallbackValue: boolean) {
  const raw = String(rawValue ?? '').trim().toLowerCase()
  if (!raw) {
    return fallbackValue
  }
  if (raw === '1' || raw === 'true' || raw === 'yes' || raw === 'on') {
    return true
  }
  if (raw === '0' || raw === 'false' || raw === 'no' || raw === 'off') {
    return false
  }
  return fallbackValue
}

function resolveDevPort() {
  const raw = Number.parseInt(process.env.FRONTEND_PORT || '', 10)
  if (Number.isInteger(raw) && raw > 0 && raw <= 65535) {
    return raw
  }
  return defaultDevPort
}

const devPort = resolveDevPort()
const disableHmr = resolveBoolean(process.env.FRONTEND_DISABLE_HMR, false)

export default defineConfig(({ command }) => ({
  define: { __ONE_BROWSER_UI_DOCUMENT__: JSON.stringify(command === 'build' ? '/' + uiDirectory + '/index.html' : '/one-browser/index.html') },
  plugins: [react(), packagedUI()],
  server: {
    port: devPort,
    strictPort: true,
    host: '127.0.0.1',
    cors: true,
    hmr: disableHmr
      ? false
      : {
          host: '127.0.0.1',
          protocol: 'ws',
        },
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom'],
        },
      },
    },
  },
}))
