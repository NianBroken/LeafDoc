// 类名组合先采用 clsx 的条件输入规则，再由 tailwind-merge 处理工具类冲突。
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// 接受字符串、条件对象和数组等 ClassValue，返回可直接传给 className 的字符串。
// 哪些工具类互相覆盖由 tailwind-merge 的规则决定，不在此自定义冲突表。
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
