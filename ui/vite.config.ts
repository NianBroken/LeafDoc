import react from '@vitejs/plugin-react'
import path from 'path'
import fs from 'node:fs'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { defineConfig, type ConfigEnv } from 'vite'

const ports = JSON.parse(
  fs.readFileSync(
    path.resolve(
      import.meta.dirname,
      '../internal/core/serviceports/ports.json',
    ),
    'utf8',
  ),
) as Record<string, number>

const backendTarget =
  process.env.LEAFDOC_BACKEND_URL?.trim() || `http://localhost:${ports.backend}`

// The shared resolver supplies identical release metadata on every platform.
const appVersion = execFileSync(
  process.execPath,
  [path.resolve(import.meta.dirname, '../scripts/resolve-version.mjs')],
  { encoding: 'utf8' },
).trim()

// Mermaid's published ESM distribution retains its own lazy diagram modules.
// Serving those modules intact avoids rebundling large generated layout engines.
function mermaidDistribution() {
  const root = path.dirname(createRequire(import.meta.url).resolve('mermaid'))
  const prefix = '/static/mermaid/'
  return {
    name: 'mermaid-distribution',
    generateBundle() {
      const emit = (directory: string, relative: string) => {
        for (const entry of fs.readdirSync(directory, {
          withFileTypes: true,
        })) {
          const name = path.posix.join(relative, entry.name)
          const file = path.join(directory, entry.name)
          if (entry.isDirectory()) emit(file, name)
          else if (entry.name.endsWith('.mjs'))
            this.emitFile({
              type: 'asset',
              fileName: prefix.slice(1) + name,
              source: fs.readFileSync(file),
            })
        }
      }
      this.emitFile({
        type: 'asset',
        fileName: prefix.slice(1) + 'mermaid.esm.min.mjs',
        source: fs.readFileSync(path.join(root, 'mermaid.esm.min.mjs')),
      })
      emit(
        path.join(root, 'chunks', 'mermaid.esm.min'),
        'chunks/mermaid.esm.min',
      )
    },
    configureServer(server: import('vite').ViteDevServer) {
      server.middlewares.use((request, response, next) => {
        const url = request.url?.split('?')[0] ?? ''
        if (!url.startsWith(prefix)) return next()
        const file = path.resolve(root, url.slice(prefix.length))
        if (
          !file.startsWith(root + path.sep) ||
          !file.endsWith('.mjs') ||
          !fs.existsSync(file)
        ) {
          response.statusCode = 404
          response.end()
          return
        }
        response.setHeader('Content-Type', 'text/javascript')
        fs.createReadStream(file).pipe(response)
      })
    },
  } satisfies import('vite').Plugin
}

// Go's //go:embed skips files whose names start with "_" or "." when embedding
// a directory, so a chunk like "_baseUniq-*.js" would be missing from the
// binary and 404 at runtime. Strip such prefixes from chunk names.
export function chunkFileName(chunk: { name: string }): string {
  return `static/${chunk.name.replace(/^[._]+/, '')}-[hash].js`
}

// https://vite.dev/config/
export default defineConfig(({ command }: ConfigEnv) => ({
  // Relative base so asset paths in the built HTML are ./static/... instead of /static/...
  // Go rewrites these references to absolute paths at serve time so they resolve correctly
  // when index.html is served for deep SPA routes. Lazy chunks resolve via import.meta.url
  // and therefore work correctly under any sub-path without further server-side patching.
  base: command === 'build' ? './' : '/',
  plugins: [
    react(),
    mermaidDistribution(),
    {
      name: 'release-metadata',
      transformIndexHtml() {
        return [
          {
            tag: 'meta',
            attrs: { name: 'app-version', content: appVersion },
            injectTo: 'head',
          },
        ]
      },
    },
  ],
  preview: { port: ports.frontend, strictPort: true },
  server: {
    port: ports.frontend,
    strictPort: true,
    proxy: {
      '/defaults': { target: backendTarget, changeOrigin: true, secure: false },
      '/avatars': { target: backendTarget, changeOrigin: true, secure: false },
      '/assets': {
        target: backendTarget,
        changeOrigin: true,
        secure: false,
      },
      '/branding': {
        target: backendTarget,
        changeOrigin: true,
        secure: false,
      },
      '/api': {
        target: backendTarget,
        changeOrigin: true,
        secure: false,
      },
    },
  },
  optimizeDeps: {
    include: ['mermaid', 'dagre-d3-es'],
  },
  build: {
    assetsDir: 'static', // <--- here you change the target directory
    rollupOptions: {
      output: {
        chunkFileNames: chunkFileName,
        entryFileNames: 'static/[name]-[hash].js',
        assetFileNames: 'static/[name].[hash][extname]',
        codeSplitting: {
          groups: [
            {
              name: 'translations',
              test: /src[/\\]locales[/\\].*\.json$/,
              maxSize: 200000,
              priority: 20,
            },
            {
              name: 'framework',
              test: /node_modules[/\\](react|react-dom|scheduler)[/\\]/,
              includeDependenciesRecursively: true,
              priority: 10,
            },
            {
              name: 'math',
              test: /node_modules[/\\]katex[/\\]/,
              includeDependenciesRecursively: true,
              priority: 15,
            },
            {
              name: 'highlight',
              test: /node_modules[/\\](highlight\.js|lowlight)[/\\]/,
              includeDependenciesRecursively: true,
              priority: 15,
            },
            {
              name: 'editor',
              test: /node_modules[/\\](@codemirror|@lezer)[/\\]/,
              includeDependenciesRecursively: true,
              priority: 15,
            },
            {
              name: 'parser',
              test: /node_modules[/\\]@lezer[/\\]/,
              includeDependenciesRecursively: true,
              priority: 25,
            },
            {
              name: 'markdown',
              test: /node_modules[/\\](remark[^/\\]*|rehype[^/\\]*|hast[^/\\]*|unist[^/\\]*|mdast[^/\\]*|micromark[^/\\]*|parse5|unified)[/\\]/,
              includeDependenciesRecursively: true,
              priority: 5,
            },
          ],
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
}))
