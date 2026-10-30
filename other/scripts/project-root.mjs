// Locate a source checkout or container snapshot independently of the caller's working directory.
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function findProjectRoot(start = import.meta.dirname) {
  let directory = start instanceof URL ? path.dirname(fileURLToPath(start)) : path.resolve(start);
  for (;;) {
    if (existsSync(path.join(directory, 'frontend/package.json')) &&
        existsSync(path.join(directory, 'other/scripts/project-root.mjs'))) return directory;
    const parent = path.dirname(directory);
    if (parent === directory) throw new Error('LeafDoc source root was not found.');
    directory = parent;
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.stdout.write(findProjectRoot() + '\n');
}
