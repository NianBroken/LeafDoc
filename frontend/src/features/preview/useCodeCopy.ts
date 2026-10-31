import copy from 'copy-to-clipboard'
import {
  ReactNode,
  isValidElement,
  useCallback,
  useEffect,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

// 从 React 子节点递归提取纯文本，复制代码和高亮节点内容都使用相同结果规则。
export function readTextContent(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node)
  }

  if (Array.isArray(node)) {
    return node.map(readTextContent).join('')
  }

  if (isValidElement<{ children?: ReactNode }>(node)) {
    return readTextContent(node.props.children)
  }

  return ''
}

export function useCodeCopy(code: string) {
  // 复制成功后短暂显示成功状态，两秒后自动复位；失败只显示错误提示。
  const { t } = useTranslation('viewer')
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!copied) return

    const timeoutId = window.setTimeout(() => {
      setCopied(false)
    }, 2000)

    return () => {
      // 状态变化或组件卸载时清除尚未触发的计时器。
      window.clearTimeout(timeoutId)
    }
  }, [copied])

  const copyCode = useCallback(() => {
    // 复制库返回 false 时不改变成功状态，成功后更新按钮状态并显示提示。
    if (!copy(code)) {
      toast.error(t('codeBlock.copyErrorToast'))
      return
    }

    setCopied(true)
    toast.success(t('codeBlock.copiedToast'))
  }, [code, t])

  return { copied, copyCode }
}
