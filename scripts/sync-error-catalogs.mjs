// Frontend error catalogues are the source used by both API and browser translations.
import { readFileSync, writeFileSync, readdirSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'

const root = fileURLToPath(new URL('../', import.meta.url))
const locales = path.join(root, 'ui', 'src', 'shared', 'locales')
const destination = path.join(root, 'internal', 'core', 'localization')
for (const entry of readdirSync(locales, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue
  writeFileSync(path.join(destination, `${entry.name}.json`), readFileSync(path.join(locales, entry.name, 'errors.json'), 'utf8').replace(/\r\n/g, '\n'), 'utf8')
}
