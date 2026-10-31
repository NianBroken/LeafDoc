// 设置根路由按登记顺序选择当前用户可见且由应用内部渲染的首个分区。
import {
  isSectionVisible,
  settingsSections,
  useSettingsSectionContext,
} from '@/shared/lib/registries/settingsSections'
import { Navigate } from 'react-router'

export default function SettingsIndexRedirect() {
  const ctx = useSettingsSectionContext()

  // 外部地址函数返回非空值的分区只提供外链，不作为内部重定向目标。
  // 找不到可用分区时回到站点根路由，所有重定向均替换当前历史记录。
  const target = settingsSections.find(
    (section) => isSectionVisible(section, ctx) && !section.externalHref?.(ctx),
  )

  return <Navigate to={target ? `/settings/${target.path}` : '/'} replace />
}
