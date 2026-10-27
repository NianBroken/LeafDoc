// Bootstrap sessions render only the password form until the server confirms completion.
import { ChangeOwnPasswordPanel } from '@/features/settings/account/ChangeOwnPasswordPanel'
import { useTranslation } from 'react-i18next'
export default function ForcePasswordChange() {
  const { t } = useTranslation('settings')
  return (
    <main className="mx-auto max-w-lg p-8">
      <h1 className="mb-6 text-2xl">{t('account.forcePasswordChange')}</h1>
      <ChangeOwnPasswordPanel />
    </main>
  )
}
