// 本文件封装 API 密钥的列表、创建和删除请求，认证与错误处理复用统一客户端。
import type { UserRole } from '@/shared/contracts/auth'
import { fetchWithAuth } from '@/shared/lib/api/client'

// 密钥角色使用项目共用的用户角色类型，不单独维护另一组名称。
export type ApiKeyRole = UserRole

// 列表项保存公开元信息与生命周期时间，不包含密钥完整明文。
export type ApiKey = {
  id: string
  name: string
  userId: string
  prefix: string
  role: ApiKeyRole
  createdBy: string
  createdAt: string
  expiresAt?: string
  lastUsedAt?: string
  revokedAt?: string
}

// 创建请求必须指定名称与所属用户，可选角色和过期时间由服务端进一步校验。
export type CreateApiKeyInput = {
  name: string
  userId: string
  role?: ApiKeyRole
  expiresAt?: string
}

// 创建结果同时含公开记录与 secret，调用方负责一次性展示，不由本层持久化。
export type CreateApiKeyResult = {
  key: ApiKey
  secret: string
}

// 读取服务端返回的密钥列表，不进行前端排序或过滤。
export async function getApiKeys(): Promise<ApiKey[]> {
  return await fetchWithAuth<ApiKey[]>('/api/api-keys')
}

// 将创建输入原样编码为 JSON，不在此填入默认角色或到期时间。
export async function createApiKey(
  input: CreateApiKeyInput,
): Promise<CreateApiKeyResult> {
  return await fetchWithAuth<CreateApiKeyResult>('/api/api-keys', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

// 将 id 直接放入请求路径并发送 DELETE，不在本地先移除列表项。
export async function deleteApiKey(id: string) {
  return await fetchWithAuth(`/api/api-keys/${id}`, {
    method: 'DELETE',
  })
}
