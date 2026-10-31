// 每个前端测试文件预载 DOM 匹配器，并在每例执行前把共享翻译实例设为英语。
import '@testing-library/jest-dom'

// 仅在 window 存在且没有存储对象时安装内存替代，已有浏览器存储保持原样。
// 替代对象只实现下列四个方法，Map 在本模块生命周期内保留，不在每例前清空。
if (typeof window !== 'undefined' && !window.localStorage) {
  const storage = new Map<string, string>()

  Object.defineProperty(window, 'localStorage', {
    value: {
      clear: () => storage.clear(),
      getItem: (key: string) => storage.get(key) ?? null,
      removeItem: (key: string) => storage.delete(key),
      setItem: (key: string, value: string) => {
        storage.set(key, value)
      },
    },
    configurable: true,
  })
}

// 组件断言使用英语标签，专门的语言场景在自身测试中再选择所需语言。
import i18next from '@/shared/lib/i18n'
import { beforeEach } from 'vitest'
beforeEach(async () => {
  await i18next.changeLanguage('en')
})
