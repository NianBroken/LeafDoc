import { UserRowActions } from './UserRowActions'
import {
  SettingsTable,
  SettingsTableCell,
  SettingsTableHeader,
} from '@/shared/components/SettingsTable'
import { mapApiError } from '@/shared/lib/api/errors'
import { useUserStore } from '@/features/users/state/users'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useSetTitle } from '@/features/viewer/setTitle'
import { CreateEditUserButton } from '@/features/users/CreateEditUserButton'

export default function UserManagement() {
  const { t } = useTranslation('users')
  const { users, loadUsers, reset } = useUserStore()
  const [loading, setLoading] = useState(true)
  useSetTitle({ title: t('pageTitle') })

  useEffect(() => {
    loadUsers()
      .catch((err) => {
        console.warn(err)
        const mapped = mapApiError(err, t('loadErrorFallback'))
        toast.error(mapped.message)
      })
      .finally(() => {
        setLoading(false)
      })

    return () => {
      reset()
    }
  }, [loadUsers, reset, t])

  return (
    <>
      <div className="settings">
        <h1 className="settings__title">{t('pageTitle')}</h1>

        <div className="settings__header-actions">
          <CreateEditUserButton />
        </div>

        <div className="settings__table-card">
          <div className="settings__table-scroll">
            <SettingsTable columns={4}>
              <thead className="settings__table-head">
                <tr>
                  <SettingsTableHeader>
                    {t('columns.username')}
                  </SettingsTableHeader>
                  <SettingsTableHeader>
                    {t('columns.email')}
                  </SettingsTableHeader>
                  <SettingsTableHeader content="role">
                    {t('columns.role')}
                  </SettingsTableHeader>
                  <SettingsTableHeader content="actions">
                    {t('columns.actions')}
                  </SettingsTableHeader>
                </tr>
              </thead>
              <tbody>
                {loading && (
                  <tr>
                    <SettingsTableCell
                      colSpan={4}
                      className="settings__table-body-message"
                    >
                      {t('loading')}
                    </SettingsTableCell>
                  </tr>
                )}
                {!loading && users.length === 0 && (
                  <tr>
                    <SettingsTableCell
                      colSpan={4}
                      className="settings__table-body-message"
                    >
                      {t('noUsersFound')}
                    </SettingsTableCell>
                  </tr>
                )}
                {!loading &&
                  users.length > 0 &&
                  users.map((user) => (
                    <tr key={user.id} className="settings__table-row">
                      <SettingsTableCell className="settings__table-cell">
                        {user.username}
                      </SettingsTableCell>
                      <SettingsTableCell className="settings__table-cell">
                        {user.email}
                      </SettingsTableCell>
                      <SettingsTableCell className="settings__table-cell">
                        <span
                          className={`settings__role-pill ${
                            user.role === 'admin'
                              ? 'settings__role-pill--admin'
                              : 'settings__role-pill--default'
                          }`}
                        >
                          {t(`roles.${user.role}`)}
                        </span>
                      </SettingsTableCell>
                      <SettingsTableCell className="settings__actions-cell">
                        <UserRowActions user={user} />
                      </SettingsTableCell>
                    </tr>
                  ))}
              </tbody>
            </SettingsTable>
          </div>
        </div>
      </div>
    </>
  )
}
