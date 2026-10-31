import i18next from '@/shared/lib/i18n'
import { Component, ErrorInfo, ReactNode } from 'react'

type MarkdownPreviewErrorBoundaryState = {
  // 子树渲染抛错后显示本地化错误提示，直到 resetKey 改变。
  hasError: boolean
}

// 捕获 Markdown 子树的渲染异常，防止单篇文档错误影响外层页面。
export class MarkdownPreviewErrorBoundary extends Component<
  { children: ReactNode; resetKey: string },
  MarkdownPreviewErrorBoundaryState
> {
  state: MarkdownPreviewErrorBoundaryState = { hasError: false }

  static getDerivedStateFromError(): MarkdownPreviewErrorBoundaryState {
    // React 在渲染异常后调用此方法，切换到错误占位视图。
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 保留异常和组件堆栈供排查，界面只显示本地化提示。
    console.error('Markdown preview failed to render', error, errorInfo)
  }

  componentDidUpdate(prevProps: { children: ReactNode; resetKey: string }) {
    // 页面路径或文档内容变化代表新的渲染目标，允许再次尝试显示预览。
    if (this.state.hasError && prevProps.resetKey !== this.props.resetKey) {
      this.setState({ hasError: false })
    }
  }

  render() {
    if (this.state.hasError) {
      // 错误视图从 viewer 语言包读取文案，不暴露内部异常细节。
      return (
        <div className="border-destructive/40 bg-destructive/5 text-destructive rounded-md border p-4 text-sm">
          {i18next.t('markdownPreview.renderError', { ns: 'viewer' })}
        </div>
      )
    }

    // 未发生异常时原样渲染整个 Markdown 子树。
    return this.props.children
  }
}
