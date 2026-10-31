// 统一占满父容器的纵向滚动区域，滚动条外观由共享 CSS 类提供。
import { ReactNode } from 'react'

// hidden 只添加隐藏样式，子组件仍保留在 React 树中，不触发卸载。
export default function ScrollableContainer({
  children,
  hidden,
}: {
  children: ReactNode
  hidden?: boolean
}) {
  return (
    <div
      className={`custom-scrollbar h-full w-full overflow-y-auto p-2 ${hidden ? 'hidden' : ''}`}
    >
      {children}
    </div>
  )
}
