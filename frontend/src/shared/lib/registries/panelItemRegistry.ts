// 保存面板描述与渲染函数，注册只建立映射，实际启用条件和模式由消费组件检查。

import type { AppMode } from '@/shared/lib/useAppMode'
import { JSX } from 'react'

// PanelItem 以稳定 ID 标识面板，标签、图标和内容在消费方调用时生成。
export interface PanelItem {
  id: string
  // 标签保存为函数，渲染时读取当前语言，避免模块注册时固定为初始语言文字。
  label: () => string
  hotkey?: string
  modes?: AppMode[]
  isEnabled?: () => boolean
  icon: () => JSX.Element
  render: (props: unknown) => JSX.Element
}

// PanelItemRegistry 按 ID 去重并保留插入次序，不提供注销或自动替换操作。
export class PanelItemRegistry {
  private items: Map<string, PanelItem> = new Map()

  // 重复 ID 直接抛错，成功时保存调用方对象引用，不复制或执行其渲染函数。
  register(item: PanelItem) {
    if (this.items.has(item.id)) {
      throw new Error(`Panel item with id ${item.id} is already registered.`)
    }
    this.items.set(item.id, item)
  }

  // 返回已登记对象引用，未知 ID 返回 undefined。
  getItem(id: string): PanelItem | undefined {
    return this.items.get(id)
  }

  // 创建当前映射值的数组快照，数组独立但内部面板对象仍共享引用。
  getAllItems(): PanelItem[] {
    return Array.from(this.items.values())
  }
}
