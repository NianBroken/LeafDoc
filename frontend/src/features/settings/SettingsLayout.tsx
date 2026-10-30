import { useExitModeButton } from '@/features/toolbar/useExitModeButton'
import { useLastWikiLocationStore } from '@/shared/state/lastWikiLocation'
import { settingsSections } from '@/shared/lib/registries/settingsSections'
import { useSetTitle } from '@/features/viewer/setTitle'
import { useCallback } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'

export default function SettingsLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation('settings')
  // Every settings route supplies its current section title to the browser tab.
  const section = settingsSections.find((entry) =>
    location.pathname.endsWith(`/settings/${entry.path}`),
  )
  useSetTitle({
    title: section ? t(section.labelKey, { ns: section.ns }) : t('nav.title'),
  })
  const lastWikiLocation = useLastWikiLocationStore((s) => s.location)

  const onExit = useCallback(() => {
    if (lastWikiLocation) {
      navigate(
        {
          pathname: lastWikiLocation.pathname,
          search: lastWikiLocation.search,
        },
        { state: lastWikiLocation.state },
      )
      return
    }
    navigate('/')
  }, [navigate, lastWikiLocation])

  useExitModeButton({
    id: 'exit-settings',
    labelKey: 'exit',
    ns: 'settings',
    hotkeyId: 'settings.exit',
    onExit,
  })

  return <Outlet />
}
