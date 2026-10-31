// 密码重置页面复用设置密码表单，成功时展示确认页面，不在此自动登录。
import { SetPasswordForm } from '@/features/auth/SetPasswordForm'

// 路由入口将 reset 模式传入，令牌、错误和提交状态由共享表单保存。
export default function ResetPasswordPage() {
  return <SetPasswordForm mode="reset" />
}
