// 本脚本为 Windows、shell 构建与前端元数据输出同一个版本值。
// 优先使用裁去边缘空白的 APP_VERSION，否则查询最近标签。
import { execFileSync } from 'node:child_process'
import { findProjectRoot } from './project-root.mjs'

const root = findProjectRoot()
let version = process.env.APP_VERSION?.trim()
// 显式环境值为空时才执行 Git 查询，不验证版本文本是否符合特定版本格式。
if (!version) {
  try {
    version = execFileSync('git', ['describe', '--tags', '--abbrev=0'], {
      cwd: root,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    }).trim()
  } catch {
    // Git 缺失、无标签或查询失败均使用固定初始版本，不把 Git 诊断写到标准输出。
    version = 'v0.1.0'
  }
}
// 标准输出只包含版本及换行，供其他脚本直接读取。
process.stdout.write(version + '\n')
