// 本脚本把前端各语言目录的 errors.json 复制到后端本地化目录，
// 只把 CRLF 换成 LF，不解析、翻译或合并 JSON 内容。
import { readFileSync, writeFileSync, readdirSync } from 'node:fs'
import { findProjectRoot } from './project-root.mjs'
import path from 'node:path'

// 两端路径都从共享定位器得到，不读取运行实例的语言设置。
const root = findProjectRoot()
const locales = path.join(root, 'frontend', 'src', 'shared', 'locales')
const destination = path.join(root, 'backend', 'internal', 'core', 'localization')
// 每个直接语言子目录产生一份后端错误目录文件；失败时立即终止，不回滚先前已写文件。
for (const entry of readdirSync(locales, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue
  writeFileSync(path.join(destination, `${entry.name}.json`), readFileSync(path.join(locales, entry.name, 'errors.json'), 'utf8').replace(/\r\n/g, '\n'), 'utf8')
}
