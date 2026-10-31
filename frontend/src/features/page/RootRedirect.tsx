// 根路由等待页面树，得到顶层节点后把访问交给该节点的路由。
import { useTreeStore } from '@/features/tree/state/tree'
import { Navigate, useLocation } from 'react-router'

// 只选择 children 中第一项，不在这里递归查找页面或按标题排序。
export default function RootRedirect() {
  const location = useLocation()
  const { tree } = useTreeStore()

  // 数据未加载或没有顶层节点时不渲染内容，也不跳转到错误页。
  if (!tree || !tree.children || tree.children.length === 0) return null

  const first = tree.children[0]
  // 替换当前历史记录并转交原导航状态，让目标页面继续处理同一次访问携带的信息。
  return <Navigate to={`/${first.path}`} replace state={location.state} />
}
