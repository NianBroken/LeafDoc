// 根据当前前端会话决定编辑控件的只读提示，不替代服务端权限校验。
import { useSessionStore } from '@/shared/state/session'

// 未登录和 viewer 返回只读，其余非空用户都返回可编辑，不额外验证角色枚举。
export function useIsReadOnly() {
  const user = useSessionStore((s) => s.user)

  // 无用户状态时界面按只读呈现，与站点是否允许公开阅读分别处理。
  if (!user) {
    return true
  }

  // 明确的阅读者角色不显示编辑能力。
  if (user.role === 'viewer') {
    return true
  }

  // 其余角色统一落入可编辑分支，角色有效性由会话来源保证。
  return false
}
