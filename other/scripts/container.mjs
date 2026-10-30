// Compilation, dependency installation and tests execute exclusively inside Linux containers.
import { spawnSync, execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { findProjectRoot } from './project-root.mjs';
const root = findProjectRoot();
const image = 'leafdoc:local';
const tools = 'leafdoc-tools:local';
const action = process.argv[2] || 'build';
function execute(args) {
  const result = spawnSync('docker', args, { cwd: root, stdio: 'inherit', windowsHide: true });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`Docker command failed with exit code ${result.status ?? 1}`);
}
const system = spawnSync('docker', ['info', '--format', '{{.OSType}}'], { encoding: 'utf8', windowsHide: true });
if (system.error) throw system.error;
if (system.status !== 0 || system.stdout.trim() !== 'linux') throw new Error('Docker must be running Linux containers');
if (action === 'build' || action === 'release') {
  const version = execFileSync(process.execPath, [fileURLToPath(new URL('./resolve-version.mjs', import.meta.url))], { encoding: 'utf8' }).trim();
  execute(['build', '-f', 'other/Dockerfile', '--build-arg', `APP_VERSION=${version}`, '-t', image, '.']);
} else {
  const commands = {
    test: 'cd backend && go test ./...',
    vet: 'cd backend && go vet ./...',
    lint: 'cd backend && golangci-lint run --config ../other/.golangci.yml ./...',
    ui: 'cd frontend && npm run build',
    'ui-check': 'cd frontend && npm test && npm run lint -- --max-warnings=0 && npm run format:check && npm run check:architecture && npm run build',
    verify: 'node other/scripts/check-project-structure.mjs && bash other/scripts/check-private-references.sh && bash other/scripts/check-toolchain-versions.sh && bash other/scripts/check-toolchain-versions.test.sh && node other/scripts/check-database-files.mjs',
    schema: 'cd backend && go run ./cmd/schema-defaults -check',
    bench: 'cd backend && go test -bench=. -benchmem -benchtime=3s ./internal/core/links/... ./internal/core/revision/...',
  };
  if (action === 'schema-write') {
    execute(['build', '-f', 'other/Dockerfile.tools', '-t', tools, '.']);
    const name = `leafdoc-defaults-${randomUUID()}`;
    try {
      execute(['run', '--name', name, '--mount', 'type=volume,source=leafdoc-linux-go-cache,target=/root/.cache/go-build', tools, 'sh', '-ec', 'cd /workspace/backend && go run ./cmd/schema-defaults']);
      for (const file of ['backend/internal/core/database/default.sql', 'frontend/src/shared/lib/instanceDefaults.json']) {
        execute(['cp', `${name}:/workspace/${file}`, path.join(root, file)]);
      }
    } finally {
      execute(['rm', '-f', name]);
    }
    process.exit(0);
  }
  if (!Object.hasOwn(commands, action)) throw new Error(`Unknown container action: ${action}`);
  execute(['build', '-f', 'other/Dockerfile.tools', '-t', tools, '.']);
  let executionImage = tools;
  if (action === 'lint') {
    const version = readFileSync(new URL('../.golangci-lint-version', import.meta.url), 'utf8').trim();
    executionImage = 'leafdoc-lint:local';
    execute(['build', '-f', 'other/Dockerfile.lint', '--build-context', `lint-toolchain=docker-image://golangci/golangci-lint:${version}`, '-t', executionImage, '.']);
  }
  execute(['run', '--rm', '--mount', 'type=volume,source=leafdoc-linux-go-cache,target=/root/.cache/go-build', '-e', 'GOOS=linux', executionImage, 'sh', '-ec', commands[action]]);
}
