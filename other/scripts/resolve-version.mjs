// Resolve one release version for Windows, shell builds and frontend metadata.
import { execFileSync } from 'node:child_process'
import { findProjectRoot } from './project-root.mjs'

const root = findProjectRoot()
let version = process.env.APP_VERSION?.trim()
if (!version) {
  try {
    version = execFileSync('git', ['describe', '--tags', '--abbrev=0'], {
      cwd: root,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    }).trim()
  } catch {
    // Untagged source archives use the same initial release as shell builds.
    version = 'v0.1.0'
  }
}
process.stdout.write(version + '\n')
