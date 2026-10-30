// Dispatches listener ownership to native Windows management or the Linux backend.
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { findProjectRoot } from './project-root.mjs';
const root = findProjectRoot();
export function releaseFixedServicePorts(services) {
  const windows = process.platform === 'win32';
  const command = windows ? 'pwsh' : 'go';
  const args = windows
    ? ['-NoProfile', '-File', path.join(root, 'other/scripts/release-service-ports.ps1'), '-Services', services.join(',')]
    : ['run', './cmd/service-ports', ...services];
  const result = spawnSync(command, args, { cwd: windows ? root : path.join(root, 'backend'), stdio: 'inherit', windowsHide: true });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error('Could not release fixed service ports');
}
if (process.argv[1] && path.resolve(process.argv[1]) === path.join(import.meta.dirname, 'release-service-ports.mjs')) {
  releaseFixedServicePorts(process.argv.slice(2));
}
