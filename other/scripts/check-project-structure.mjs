// Enforce the three project ownership areas and their required entry files.
import fs from 'node:fs';
import path from 'node:path';
import { findProjectRoot } from './project-root.mjs';

const root = findProjectRoot();
const expected = new Set(['backend', 'frontend', 'other']);
const actual = new Set(fs.readdirSync(root, { withFileTypes: true }).filter((entry) =>
  entry.isDirectory() && !entry.name.startsWith('.')).map((entry) => entry.name));
const failures = [];
for (const name of expected) if (!actual.has(name)) failures.push(`missing root directory: ${name}`);
for (const name of actual) if (!expected.has(name)) failures.push(`unexpected root directory: ${name}`);
for (const file of ['backend/go.mod', 'backend/go.sum', 'frontend/package.json', 'frontend/package-lock.json', 'other/scripts/start.ps1']) {
  if (!fs.existsSync(path.join(root, file))) failures.push(`missing required path: ${file}`);
}
for (const name of ['cmd', 'internal', 'ui', 'scripts', 'data']) {
  if (fs.existsSync(path.join(root, name))) failures.push(`legacy root path remains: ${name}`);
}
if (failures.length) throw new Error(failures.join('\n'));
process.stdout.write('Project structure verified.\n');
