// 根据实际浅深色主题显示切换按钮，点击后将选择明确保存为相反主题。
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
    // 提示文字描述点击后的目标主题，与当前主题使用的图标共同表达操作。
    mode === 'dark'
      ? t('designToggle.switchToLight')
      : t('designToggle.switchToDark')
  const isMobile = useIsMobile()
  const appMode = useAppMode()

  // 移动端编辑界面不渲染此入口，其他组合继续显示切换按钮。
  if (isMobile && appMode === 'edit') return null

  return (
    <TooltipWrapper label={label} align="center">
      <Button
        variant="outline"
        size="icon"
        aria-label={label}
        onClick={() => {
          // 跟随系统模式也按解析后的颜色切换，点击会结束当前的 system 选择。
          setMode(mode === 'dark' ? 'light' : 'dark')
        }}
      >
        <Moon className={mode === 'light' ? 'visible' : 'hidden'} />
        <Sun className={mode === 'dark' ? 'visible' : 'hidden'} />
      </Button>
    </TooltipWrapper>
  )
}
