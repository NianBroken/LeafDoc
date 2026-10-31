// 此页面并行读取密钥和用户列表，将所属账号标识映射为名称并提供创建与撤销入口。
import {
  SettingsTable,
  SettingsTableCell,
} from '@/shared/components/SettingsTable'
import { mapApiError } from '@/shared/lib/api/errors'
import { useApiKeyStore } from '@/features/apikeys/state/apikeys'
import { useUserStore } from '@/features/users/state/users'
import { useDateTimeFormat } from '@/shared/lib/useDateTimeFormat'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSetTitle } from '@/features/viewer/setTitle'
import { CreateApiKeyButton } from '@/features/apikeys/CreateApiKeyButton'
import { DeleteApiKeyButton } from '@/features/apikeys/DeleteApiKeyButton'

// 时间由共享格式器使用用户偏好展示，空值以占位文字表示。
export default function ApiKeysManagement() {
  const { t } = useTranslation('apikeys')
  const { formatDateTime } = useDateTimeFormat()
  const formatTimestamp = (value?: string) => formatDateTime(value) || '—'
  const { apiKeys, loadApiKeys, reset } = useApiKeyStore()
  const { users, loadUsers } = useUserStore()
  const [loading, setLoading] = useState(true)
  useSetTitle({ title: t('page.title') })

  useEffect(() => {
    // 两个请求都成功才正常结束加载，任一失败显示错误；已有成功请求不回滚。
    Promise.all([loadApiKeys(), loadUsers()])
      .catch((err) => {
        console.warn(err)
        const mapped = mapApiError(err, t('page.loadErrorFallback'))
        toast.error(mapped.message)
      })
      .finally(() => {
        setLoading(false)
      })

    return () => {
      // 页面离开或依赖变化时清空密钥状态，不清空用户列表，也不取消在途加载。
      reset()
    }
  }, [loadApiKeys, loadUsers, reset, t])

  const usernameById = useMemo(
    // 用户列表变化时重建查找表，避免每行重复遍历整个用户数组。
    () => new Map(users.map((u) => [u.id, u.username])),
    [users],
  )
  // 找不到用户记录时显示原始标识，仍能识别密钥所属对象。
  const usernameFor = (userId: string) => usernameById.get(userId) ?? userId

  return (
    <div className="settings">
      <h1 className="settings__title">{t('page.heading')}</h1>

      <div className="settings__header-actions">
        <CreateApiKeyButton />
      </div>

      <div className="settings__table-card">
        <div className="settings__table-scroll">
          <SettingsTable columns={6}>
            <thead className="settings__table-head">
              <tr>
                <th className="settings__table-header-cell">
                  {t('table.name')}
                </th>
                <th className="settings__table-header-cell">
                  {t('table.owner')}
                </th>
                <th className="settings__table-header-cell">
                  {t('table.role')}
                </th>
                <th className="settings__table-header-cell">
                  {t('table.expires')}
                </th>
                <th className="settings__table-header-cell">
                  {t('table.lastUsed')}
                </th>
                <th className="settings__table-header-cell">
                  {t('table.actions')}
                </th>
              </tr>
            </thead>
            <tbody>
              {/* 加载、空列表和数据行按 loading 与列表长度互斥显示。 */}
              {loading && (
                <tr>
                  <SettingsTableCell
                    colSpan={6}
                    className="settings__table-body-message"
                  >
                    {t('table.loading')}
                  </SettingsTableCell>
                </tr>
              )}
              {!loading && apiKeys.length === 0 && (
                <tr>
                  <SettingsTableCell
                    colSpan={6}
                    className="settings__table-body-message"
                  >
                    {t('table.empty')}
                  </SettingsTableCell>
                </tr>
              )}
              {!loading &&
                apiKeys.length > 0 &&
                apiKeys.map((apiKey) => (
                  <tr key={apiKey.id} className="settings__table-row">
                    <SettingsTableCell className="settings__table-cell">
                      {apiKey.name}
                    </SettingsTableCell>
                    <SettingsTableCell className="settings__table-cell">
                      {usernameFor(apiKey.userId)}
                    </SettingsTableCell>
                    <SettingsTableCell className="settings__table-cell">
                      <span
                        className={`settings__role-pill ${
                          apiKey.role === 'admin'
                            ? 'settings__role-pill--admin'
                            : 'settings__role-pill--default'
                        }`}
                      >
                        {t(`role.${apiKey.role}`, {
                          defaultValue: apiKey.role,
                        })}
                      </span>
                    </SettingsTableCell>
                    <SettingsTableCell className="settings__table-cell">
                      {formatTimestamp(apiKey.expiresAt)}
                    </SettingsTableCell>
                    <SettingsTableCell className="settings__table-cell">
                      {formatTimestamp(apiKey.lastUsedAt)}
                    </SettingsTableCell>
                    <SettingsTableCell
                      className="settings__actions-cell"
                      content="controls"
                    >
                      <div className="settings__actions">
                        {/* 已撤销记录只显示状态，仍有效的记录提供撤销确认入口。 */}
                        {apiKey.revokedAt ? (
                          <span className="settings__table-body-message">
                            {t('table.revoked')}
                          </span>
                        ) : (
                          <DeleteApiKeyButton apiKey={apiKey} />
                        )}
                      </div>
                    </SettingsTableCell>
                  </tr>
                ))}
            </tbody>
          </SettingsTable>
        </div>
      </div>
    </div>
  )
}
