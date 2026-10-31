// 捕获子树渲染阶段传入 React 错误边界的异常，显示重载、重试和诊断详情。
import { Button } from '@/shared/components/ui/button'
import i18next from '@/shared/lib/i18n'
import { AlertTriangle, RefreshCw } from 'lucide-react'
import { Component, ErrorInfo } from 'react'
import type { ReactNode } from 'react'

// resetKey 变化时清理已有错误，未变化时继续保留当前错误界面。
type Props = {
  children: ReactNode
  resetKey?: string
}

type State = {
  error: Error | null
}

// ErrorBoundary 通过状态切换正常子树与失败界面，不自动重试网络请求。
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  // 非 Error 抛出值转换为 Error，统一后续详情显示格式。
  static getDerivedStateFromError(error: unknown): State {
    return { error: error instanceof Error ? error : new Error(String(error)) }
  }

  // 有错误且重置标识改变时重新渲染子树，让新路径不继续显示旧路径的错误。
  componentDidUpdate(prevProps: Props) {
    if (this.state.error && prevProps.resetKey !== this.props.resetKey) {
      this.setState({ error: null })
    }
  }

  // 记录原错误及 React 组件栈，界面详情使用 Error 自身的堆栈。
  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack)
  }

  // 整页重载重新执行应用入口，并非只清理当前组件状态。
  private handleReload = () => {
    window.location.reload()
  }

  // 局部重试只清理错误状态，若子树再次抛错会重新进入边界。
  private handleReset = () => {
    this.setState({ error: null })
  }

  render() {
    const { error } = this.state

    if (!error) return this.props.children

    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-6 p-8">
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="bg-destructive/10 text-destructive flex h-16 w-16 items-center justify-center rounded-full">
            <AlertTriangle className="h-8 w-8" />
          </div>
          <div className="flex flex-col gap-1">
            <h1 className="text-foreground text-xl font-semibold">
              {i18next.t('errorBoundary.heading', { ns: 'common' })}
            </h1>
            <p className="text-muted-foreground max-w-md text-sm">
              {i18next.t('errorBoundary.body', { ns: 'common' })}
            </p>
          </div>
        </div>

        <div className="flex gap-3">
          <Button onClick={this.handleReload} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            {i18next.t('errorBoundary.reloadButton', { ns: 'common' })}
          </Button>
          <Button variant="outline" onClick={this.handleReset}>
            {i18next.t('errorBoundary.tryRecoverButton', { ns: 'common' })}
          </Button>
        </div>

        <details className="border-border bg-muted-surface/40 w-full max-w-xl rounded-md border">
          <summary className="text-muted-foreground cursor-pointer p-3 text-xs font-medium select-none">
            {i18next.t('errorBoundary.detailsSummary', { ns: 'common' })}
          </summary>
          <pre className="text-destructive overflow-auto p-3 pt-0 text-xs break-all whitespace-pre-wrap">
            {error.stack ?? error.message}
          </pre>
        </details>
      </div>
    )
  }
}
