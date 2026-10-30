import { useResolvedDesignMode } from './useResolvedDesignMode'
import { Moon, Sun } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/shared/components/ui/button'
import { useDesignModeStore } from '@/features/designtoggle/designmode'
import { TooltipWrapper } from '@/shared/components/TooltipWrapper'
import { useAppMode } from '@/shared/lib/useAppMode'
import { useIsMobile } from '@/shared/lib/useIsMobile'

export default function DesignToggle() {
  const { t } = useTranslation('viewer')
  const mode = useResolvedDesignMode()
  const setMode = useDesignModeStore((s) => s.setMode)
  const label =
    mode === 'dark'
      ? t('designToggle.switchToLight')
      : t('designToggle.switchToDark')
  const isMobile = useIsMobile()
  const appMode = useAppMode()

  if (isMobile && appMode === 'edit') return null

  return (
    <TooltipWrapper label={label} align="center">
      <Button
        variant="outline"
        size="icon"
        aria-label={label}
        onClick={() => {
          setMode(mode === 'dark' ? 'light' : 'dark')
        }}
      >
        <Moon className={mode === 'light' ? 'visible' : 'hidden'} />
        <Sun className={mode === 'dark' ? 'visible' : 'hidden'} />
      </Button>
    </TooltipWrapper>
  )
}
