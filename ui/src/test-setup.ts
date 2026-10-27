import '@testing-library/jest-dom'

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

// Existing component fixtures assert English labels; localization scenarios select their own language.
import i18next from '@/shared/lib/i18n'
import { beforeEach } from 'vitest'
beforeEach(async () => {
  await i18next.changeLanguage('en')
})
