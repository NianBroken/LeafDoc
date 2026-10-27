import { useTreeStore } from '@/features/tree/state/tree'
import { Navigate, useLocation } from 'react-router'

export default function RootRedirect() {
  const location = useLocation()
  const { tree } = useTreeStore()

  if (!tree || !tree.children || tree.children.length === 0) return null

  const first = tree.children[0]
  return <Navigate to={`/${first.path}`} replace state={location.state} />
}
