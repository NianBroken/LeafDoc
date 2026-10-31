// 本文件封装当前用户头像操作，并为指定用户拼接浏览器可用头像地址。
import { withBasePath } from '@/shared/lib/routePath'
import { fetchWithAuth } from '@/shared/lib/api/client'

// getAvatarStatus 禁用请求缓存并读取自定义头像存在标记，不根据图片加载回退推断状态。
export function getAvatarStatus(): Promise<{ hasCustomAvatar: boolean }> {
  return fetchWithAuth('/api/user/avatar', { cache: 'no-store' })
}

// 以 file 表单字段上传头像，等待请求完成，不保存或返回响应正文。
export async function uploadAvatar(file: File): Promise<void> {
  const formData = new FormData()
  formData.append('file', file)

  await fetchWithAuth('/api/user/avatar', {
    method: 'POST',
    body: formData,
    headers: {}, // FormData 的分隔标记与内容类型由浏览器生成。
  })
}

// 删除当前用户头像，函数不在此修改头像版本号或本地用户状态。
export async function deleteAvatar(): Promise<void> {
  await fetchWithAuth('/api/user/avatar', {
    method: 'DELETE',
  })
}

/**
 * 拼接带部署前缀的头像路径，version 不为 undefined 时追加 v 查询参数。
 * 不编码 userId，不发送请求，也不自动生成版本号，参数来源由调用方保证。
 */
export function avatarUrl(userId: string, version?: number): string {
  const path = withBasePath(`/avatars/${userId}`)
  return version !== undefined ? `${path}?v=${version}` : path
}
