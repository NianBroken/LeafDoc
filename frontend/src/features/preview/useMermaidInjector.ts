import { isDynamicImportChunkError } from '@/shared/lib/dynamicImportError'
import i18next from '@/shared/lib/i18n'
import { withBasePath } from '@/shared/lib/routePath'
import { useEffect, useRef } from 'react'

// Mermaid 模块按需加载并缓存 Promise，图表代码、主题或行号变化时才更新 SVG。
type MermaidDefault = (typeof import('mermaid'))['default']

let mermaidPromise: Promise<MermaidDefault> | null = null

function loadMermaid(): Promise<MermaidDefault> {
  // 资源地址经过站点基础路径转换，动态导入结果由模块级 Promise 在组件间共享。
  if (!mermaidPromise) {
    const url = withBasePath('/static/mermaid/mermaid.esm.min.mjs')
    mermaidPromise = import(/* @vite-ignore */ url).then((m) => m.default)
  }
  return mermaidPromise
}

let initializedTheme: 'default' | 'dark' | null = null

async function ensureMermaidInitialized(
  theme: 'default' | 'dark',
): Promise<MermaidDefault> {
  // 同一个 Mermaid 实例每次最多按需初始化到当前主题，配置固定禁用脚本标签。
  const mermaid = await loadMermaid()

  if (initializedTheme !== theme) {
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'antiscript',
      theme,
      deterministicIds: true,
      deterministicIDSeed: 'leafdoc',
    })
    mermaid.setParseErrorHandler((err) => {
      console.warn('Mermaid parse error:', err)
    })
    initializedTheme = theme
  }

  return mermaid
}

export type MermaidRenderError = {
  message: string
  isChunkLoadError: boolean
}

export type MermaidInjectorOps = {
  containerRef: React.RefObject<HTMLDivElement | null>
  code: string
  dataLine?: string
  theme: 'default' | 'dark'
  onError: (error: MermaidRenderError | null) => void
}

function djb2(str: string) {
  // 对字符串字符码滚动计算无符号散列，用于比较代码输入而不是验证数据完整性。
  let h = 5381
  for (let i = 0; i < str.length; i++) h = ((h << 5) + h) ^ str.charCodeAt(i)
  return (h >>> 0).toString(36)
}

function normalizeCode(code: string) {
  // 统一换行，移除首尾空行，再按非空行共同缩进裁剪每行前缀。
  const lines = code.replace(/\r\n/g, '\n').split('\n')
  // remove leading/trailing empty lines
  while (lines.length && lines[0].trim() === '') lines.shift()
  while (lines.length && lines[lines.length - 1].trim() === '') lines.pop()
  // determine common indent
  const indents = lines
    .filter((l) => l.trim() !== '')
    .map((l) => l.match(/^(\s+)/)?.[1].length ?? 0)
  const min = indents.length ? Math.min(...indents) : 0
  // Remove common indent
  const out = lines.map((l) => l.slice(min)).join('\n')
  return out
}

export function useMermaidInjector({
  containerRef,
  code,
  dataLine,
  theme,
  onError,
}: MermaidInjectorOps) {
  const lastHashRef = useRef<string | null>(null)
  const lastDataLineRef = useRef<string | undefined>(undefined)
  const lastThemeRef = useRef<'default' | 'dark' | null>(null)

  useEffect(() => {
    // 主题变化使代码缓存失效，后续效果将按新主题重新生成 SVG。
    if (lastThemeRef.current !== theme) {
      lastHashRef.current = null
      lastThemeRef.current = theme
    }
  }, [theme])

  useEffect(() => {
    if (!containerRef) return
    if (!containerRef.current) return

    let cancelled = false

    async function inject() {
      const el = containerRef.current
      if (!el) return
      const normalizedCode = normalizeCode(code)

      const codeHash = djb2(normalizedCode)
      // 代码未变时只同步 data-line，避免再次解析并生成整张图。
      if (lastHashRef.current === codeHash) {
        if (lastDataLineRef.current !== dataLine) {
          // 直接更新已生成 SVG 的行号属性。
          const oldSVG = el.querySelector('svg')
          if (oldSVG) {
            if (dataLine != null) {
              oldSVG.setAttribute('data-line', String(dataLine))
            } else {
              oldSVG.removeAttribute('data-line')
            }
          }
          lastDataLineRef.current = dataLine
        }
        onError(null)
        return // No need to re-render
      }
      // 临时画布用于 Mermaid 测量，缺少该节点时记录告警并保留当前画面。
      const sandbox = document.getElementById('mermaid-renderer')
      if (!sandbox) {
        console.warn('Mermaid renderer element not found')
        return
      }
      try {
        const mermaid = await ensureMermaidInitialized(theme)
        await mermaid.parse(normalizedCode)
        const { svg } = await mermaid.render(
          `mermaid-${codeHash}-${dataLine || '0'}`,
          normalizedCode,
          sandbox,
        )

        if (cancelled) return

        // 以 HTML 模板解析 SVG 字符串，避免 foreignObject 中的 HTML 空元素触发 XML 解析错误。
        // 模板内容保持惰性，Mermaid 生成配置使用 antiscript 安全级别处理 HTML 标签。
        const template = document.createElement('template')
        // eslint-disable-next-line no-restricted-syntax
        template.innerHTML = svg
        const newSvg = template.content.querySelector(
          'svg',
        ) as SVGSVGElement | null
        if (!newSvg)
          throw new Error(i18next.t('mermaid.noSvgOutput', { ns: 'viewer' }))
        newSvg.setAttribute('width', '100%')
        newSvg.removeAttribute('height')
        newSvg.setAttribute('preserveAspectRatio', 'xMinYMin meet')
        if (dataLine != null) newSvg.setAttribute('data-line', String(dataLine))

        const oldSVG = el.querySelector('svg')
        if (oldSVG) {
          el.replaceChild(newSvg, oldSVG)
        } else {
          el.appendChild(newSvg)
        }

        // 记录已渲染代码和行号，供下次效果判断是否只需调整属性。
        lastHashRef.current = codeHash
        lastDataLineRef.current = dataLine

        // 同步外层容器的行号，供预览行跳转逻辑定位图表来源。
        if (dataLine != null) {
          el.setAttribute('data-line', String(dataLine))
        } else {
          el.removeAttribute('data-line')
        }

        // 解除渲染前预留的最小高度。
        el.style.minHeight = ''
        onError(null)
      } catch (error) {
        if (cancelled) return

        lastHashRef.current = null
        // 下次效果继续尝试渲染，错误消息和动态模块加载错误类型交给调用方显示。
        const message = error instanceof Error ? error.message : String(error)
        console.warn('Mermaid diagram could not be rendered:', message)
        onError({
          message,
          isChunkLoadError: isDynamicImportChunkError(message),
        })
      }
    }

    const raf1 = requestAnimationFrame(inject)
    return () => {
      cancelled = true
      if (raf1) cancelAnimationFrame(raf1)
    }
  }, [containerRef, code, dataLine, theme, onError])
}
