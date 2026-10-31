// 维护对话框类型与渲染函数映射，打开状态由独立的全局对话框存储维护。

import { JSX } from 'react/jsx-runtime'

// Dialog 通过类型字符串匹配入口，render 接收未知属性并由应用装配层转交组件。
export interface Dialog {
  type: string
  render: (props: unknown) => JSX.Element
}

// DialogRegistry 保留注册次序，拒绝类型重复，不在注册时实例化组件。
export class DialogRegistry {
  private dialogs: Map<string, Dialog> = new Map()

  // 保存原对象引用，已占用类型抛错，不覆盖原有注册内容。
  register(dialog: Dialog) {
    if (this.dialogs.has(dialog.type)) {
      throw new Error(`Dialog with type ${dialog.type} is already registered.`)
    }
    this.dialogs.set(dialog.type, dialog)
  }

  // 精确按类型查找，未知类型返回 undefined。
  getDialog(type: string): Dialog | undefined {
    return this.dialogs.get(type)
  }

  // 返回新数组，后续登记不会自动进入先前取得的数组，内部对象不深复制。
  getAllDialogs(): Dialog[] {
    return Array.from(this.dialogs.values())
  }
}
