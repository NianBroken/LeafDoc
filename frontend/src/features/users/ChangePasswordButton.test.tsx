// 使用管理员会话分别渲染其他用户与当前用户，核验按钮是否存在，不提交密码请求。
import '@/shared/lib/i18n'
import { User } from '@/shared/lib/api/users'
import { useSessionStore } from '@/shared/state/session'
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { ChangePasswordButton } from '@/features/users/ChangePasswordButton'

const admin: User = {
  id: 'admin-id',
  username: 'admin',
  email: 'admin@example.com',
  role: 'admin',
  mustSetPassword: false,
}

const other: User = {
  id: 'other-id',
  username: 'jane',
  email: 'jane@example.com',
  role: 'editor',
  mustSetPassword: false,
}

describe('ChangePasswordButton', () => {
  beforeEach(() => {
    useSessionStore.setState({ user: admin })
  })

  // 目标 ID 与会话用户不同，管理列表提供修改目标密码的按钮。
  it('renders for another user', () => {
    render(<ChangePasswordButton user={other} />)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  // 当前账号在账号设置修改自身密码，这个列表入口对同 ID 用户不渲染按钮。
  it('does not render for the current user (own password is changed in account settings)', () => {
    render(<ChangePasswordButton user={admin} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})
