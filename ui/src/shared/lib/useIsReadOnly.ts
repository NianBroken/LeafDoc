import { useSessionStore } from '@/shared/state/session'

export function useIsReadOnly() {
  const user = useSessionStore((s) => s.user)

  // Anonymous visitors never receive editing controls.
  if (!user) {
    return true
  }

  // viewer role -> only read
  if (user.role === 'viewer') {
    return true
  }

  // admin and editor roles -> read and write
  return false
}
