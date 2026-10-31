// 角色测试核对字符串与允许列表的判断，不构造令牌，也不验证服务端会话。
import { describe, expect, it } from 'vitest'
import { hasRole } from '@/shared/lib/roles'

describe('hasRole', () => {
  it('returns true when the role is in the allowed list', () => {
    // 精确包含的角色通过判断，列表中其他角色不影响结果。
    expect(hasRole('admin', ['admin', 'editor'])).toBe(true)
  })

  it('returns false when the role is not in the allowed list', () => {
    // viewer 没有出现在管理员与编辑者列表中，因此被拒绝。
    expect(hasRole('viewer', ['admin', 'editor'])).toBe(false)
  })

  it('treats an empty allowed list as "any authenticated user"', () => {
    // 空列表不限制已提供的角色，分别用管理员与阅读者覆盖该分支。
    expect(hasRole('admin', [])).toBe(true)
    expect(hasRole('viewer', [])).toBe(true)
  })

  it('returns false when role is undefined, even with an empty allowed list', () => {
    // 缺少角色先于允许列表判断，空列表也不能让 undefined 通过。
    expect(hasRole(undefined, ['admin'])).toBe(false)
    expect(hasRole(undefined, [])).toBe(false)
  })
})
