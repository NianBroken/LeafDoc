// 邀请页面复用设置密码表单，模式决定调用接受邀请接口并在成功后建立会话。
import { SetPasswordForm } from '@/features/auth/SetPasswordForm'

// 路由入口只选择模式，令牌读取、输入校验和成功导航由共享表单处理。
export default function AcceptInvitePage() {
  return <SetPasswordForm mode="invite" />
}
