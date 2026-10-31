// 内部分区路由同时检查可见性与外链配置，不满足内部展示条件时回到设置根页。
import {
  isSectionVisible,
  useSettingsSectionContext,
  type SettingsSection,
} from '@/shared/lib/registries/settingsSections'
import { type ReactNode } from 'react'
import { Navigate } from 'react-router'

type Props = {
  section: SettingsSection
  children: ReactNode
}

export default function SettingsSectionGuard({ section, children }: Props) {
  const ctx = useSettingsSectionContext()

  if (!isSectionVisible(section, ctx)) {
    return <Navigate to="/settings" replace />
  }

  // 非空外部地址意味着该分区入口用于离站链接，此守卫不渲染其内部组件。
  // 这里仅导航到设置根页，不直接执行外部跳转；根页继续选择可用内部分区。
  if (section.externalHref?.(ctx)) {
    return <Navigate to="/settings" replace />
  }

  return <>{children}</>
}
