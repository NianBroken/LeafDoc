// 前端构建与本地服务共用端口表、版本入口及页面模板处理插件。
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { runtimeHtml } from './runtimeHtml.ts'
import path from 'path'
import fs from 'node:fs'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { releaseFixedServicePorts } from '../other/scripts/release-service-ports.mjs'
import { defineConfig, type ConfigEnv } from 'vite'

const ports = JSON.parse(
  fs.readFileSync(
    path.resolve(
      import.meta.dirname,
      '../backend/internal/core/serviceports/ports.json',
    ),
    'utf8',
  ),
) as Record<string, number>

const backendTarget =
  process.env.LEAFDOC_BACKEND_URL?.trim() || `http://localhost:${ports.backend}`

// 环境指定的后端地址仍必须使用共享后端端口，端口不符时在加载配置阶段终止。
if (new URL(backendTarget).port !== String(ports.backend)) {
  throw new Error('Backend target must use the fixed backend port')
}
// 版本由共享脚本读取，构建与开发页面使用同一个解析结果。
const appVersion = execFileSync(
  process.execPath,
  [path.resolve(import.meta.dirname, '../other/scripts/resolve-version.mjs')],
  { encoding: 'utf8' },
).trim()

// Mermaid 的模块发行文件保留自身的按需导入路径，插件将入口与分块按原内容提供。
// 构建时将这些模块作为资源输出，开发时由中间件直接读取依赖目录。
function mermaidDistribution() {
  const root = path.dirname(createRequire(import.meta.url).resolve('mermaid'))
  const prefix = '/static/mermaid/'
  return {
    name: 'mermaid-distribution',
    async generateBundle() {
      // 每层目录的条目并发读取，递归保留相对目录，仅输出 .mjs 文件。
      const emit = async (
        directory: string,
        relative: string,
      ): Promise<void> => {
        const entries = await fs.promises.readdir(directory, {
          withFileTypes: true,
        })
        await Promise.all(
          entries.map(async (entry) => {
            const name = path.posix.join(relative, entry.name)
            const file = path.join(directory, entry.name)
            if (entry.isDirectory()) await emit(file, name)
            else if (entry.name.endsWith('.mjs')) {
              const source = await fs.promises.readFile(file)
              this.emitFile({
                type: 'asset',
                fileName: prefix.slice(1) + name,
                source,
              })
            }
          }),
        )
      }
      this.emitFile({
        type: 'asset',
        fileName: prefix.slice(1) + 'mermaid.esm.min.mjs',
        source: await fs.promises.readFile(
          path.join(root, 'mermaid.esm.min.mjs'),
        ),
      })
      await emit(
        path.join(root, 'chunks', 'mermaid.esm.min'),
        'chunks/mermaid.esm.min',
      )
    },
    configureServer(server: import('vite').ViteDevServer) {
      server.middlewares.use((request, response, next) => {
        const url = request.url?.split('?')[0] ?? ''
        if (!url.startsWith(prefix)) return next()
        // 解析后的路径必须仍在依赖目录内部，且是现存 .mjs 文件，否则返回 404。
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

// Go 按目录嵌入文件时跳过点或下划线开头的名称。
// 去掉分块名称的连续前缀，确保这些脚本能进入后端嵌入资源。
export function chunkFileName(chunk: { name: string }): string {
  return `static/${chunk.name.replace(/^[._]+/, '')}-[hash].js`
}

// 配置对象按当前命令区分构建资源基路径，其余插件和端口约束共用。
export default defineConfig(({ command }: ConfigEnv) => ({
  // 构建模板中的入口资源使用相对路径，Go 返回页面时将静态入口定位到实例基础路径。
  // 开发服务使用根路径；按需导入的模块继续依据其自身模块地址解析资源。
  base: command === 'build' ? './' : '/',
  plugins: [
    react(),
    tailwindcss(),
    runtimeHtml(backendTarget),
    mermaidDistribution(),
    {
      name: 'release-metadata',
      configResolved(config) {
        // 在启动监听前拒绝漂移的端口配置，不允许自动切换到其他空闲端口。
        if (
          config.server.port !== ports.frontend ||
          config.preview.port !== ports.frontend ||
          !config.server.strictPort ||
          !config.preview.strictPort
        ) {
          throw new Error(
            'Frontend must use the fixed frontend port with strict port checking',
          )
        }
      },
      configureServer() {
        releaseFixedServicePorts(['frontend'])
      },
      configurePreviewServer() {
        releaseFixedServicePorts(['frontend'])
      },
      transformIndexHtml() {
        // 页面携带解析后的版本，前端无需另行调用 Git 或读取宿主环境。
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
    // 页面由 Vite 提供，业务接口及实例资源交给后端，转发时改写 Host。
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
    assetsDir: 'static', // 构建生成的资源统一位于 static，名称规则由下方输出配置限定。
    rollupOptions: {
      output: {
        chunkFileNames: chunkFileName,
        entryFileNames: 'static/[name]-[hash].js',
        assetFileNames: 'static/[name].[hash][extname]',
        codeSplitting: {
          // 按依赖路径划分翻译、框架、公式、编辑器与 Markdown 模块，组优先级显式声明。
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
