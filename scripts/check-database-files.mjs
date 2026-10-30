// Runtime and source checks permit one schema file and reject embedded database artifacts.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../', import.meta.url))
const schema = 'internal/core/database/default.sql'
const excluded = new Set(['.git', 'node_modules'])
const forbidden = /\.(?:db(?:-wal|-shm|-journal)?|sqlite[0-9]*(?:-wal|-shm|-journal)?|db3|mdb|ldb|rdb|aof|sql)$/i
const failures = []
let schemaCount = 0

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
    const relative = path.relative(root, file).split(path.sep).join('/')
    if (relative === schema) { schemaCount++; continue }
    if (forbidden.test(entry.name)) { failures.push(relative); continue }
    const fd = fs.openSync(file, 'r')
    try {
      const header = Buffer.alloc(16)
      fs.readSync(fd, header, 0, header.length, 0)
      if (header.toString('ascii') === 'SQLite format 3\0') failures.push(relative)
    } finally { fs.closeSync(fd) }
  }
}
inspect(root)
if (schemaCount !== 1 || failures.length) {
  throw new Error(`Database file policy failed: schema count=${schemaCount}; ${failures.join(', ')}`)
}
console.log('Database file policy verified: one default SQL; no local database artifacts.')
