// 浏览器入口载入全局样式与本地字体，将应用挂载到 HTML 提供的 root 元素。
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from '@/app/App.tsx'
import '@/index.css'

// 逐项导入界面所用字重及斜体，字体文件由构建系统纳入静态资源。
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/inter/700.css'
import '@fontsource/inter/800.css'
import '@fontsource/inter/900.css'
import '@fontsource/inter/400-italic.css'
import '@fontsource/inter/500-italic.css'
import '@fontsource/inter/600-italic.css'
import '@fontsource/inter/700-italic.css'

// 非空断言属于类型提示，运行时要求页面确实存在 root 元素。
// StrictMode 在开发环境执行额外检查，应用中的副作用清理需遵守 React 生命周期。
createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
