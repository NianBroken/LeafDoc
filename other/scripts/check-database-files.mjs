// 本脚本检查项目树中的数据库文件，只允许固定位置的默认 SQL，
// 并按扩展名与 SQLite 文件头定位不允许保留的本地数据库产物。
import fs from 'node:fs'
import path from 'node:path'
import { findProjectRoot } from './project-root.mjs'

// 根目录由共享定位器取得，不依赖调用脚本时的工作目录。
const root = findProjectRoot()
const schema = 'backend/internal/core/database/default.sql'
// 只跳过这两个目录名，运行数据等其他目录仍会被递归检查。
const excluded = new Set(['.git', 'node_modules'])
const forbidden = /\.(?:db(?:-wal|-shm|-journal)?|sqlite[0-9]*(?:-wal|-shm|-journal)?|db3|mdb|ldb|rdb|aof|sql)$/i
const failures = []
let schemaCount = 0

// 同步递归检查目录，不跟随符号链接，链接自身仍按名称匹配禁止扩展名。
// 文件系统读取错误直接中止脚本，不将无法读取的文件当作已通过。
function inspect(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name)
    if (entry.isSymbolicLink()) {
      if (forbidden.test(entry.name)) failures.push(path.relative(root, file))
      continue
    }
    if (entry.isDirectory()) {
      if (!excluded.has(entry.name)) inspect(file)
      continue
    }
    // 策略中的相对路径统一为正斜杠，兼容 Windows 的目录分隔符。
    const relative = path.relative(root, file).split(path.sep).join('/')
    if (relative === schema) { schemaCount++; continue }
    if (forbidden.test(entry.name)) { failures.push(relative); continue }
    // 默认 SQL 与禁止扩展名已经处理，其余文件读取前十六字节辨认 SQLite。
    const fd = fs.openSync(file, 'r')
    try {
      const header = Buffer.alloc(16)
      fs.readSync(fd, header, 0, header.length, 0)
      if (header.toString('ascii') === 'SQLite format 3\0') failures.push(relative)
    } finally { fs.closeSync(fd) } // 读取或内容判断失败也关闭已打开的文件句柄。
  }
}
// 固定默认 SQL 必须恰好出现一次，违规路径在遍历结束后合并报告。
inspect(root)
if (schemaCount !== 1 || failures.length) {
  throw new Error(`Database file policy failed: schema count=${schemaCount}; ${failures.join(', ')}`)
}
console.log('Database file policy verified: one default SQL; no local database artifacts.')
