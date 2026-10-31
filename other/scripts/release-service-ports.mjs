// 本模块按宿主平台选择固定服务端口的释放入口，服务名由调用方传入。
// 这类操作会停止占用进程，模块导入本身不执行释放。
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { findProjectRoot } from './project-root.mjs';
const root = findProjectRoot();
// Windows 调用 PowerShell 原生进程管理脚本，其他平台运行后端 service-ports 命令。
// 命令同步执行并继承输出，启动失败或非零退出码都向调用方抛错。
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
// 直接执行脚本时把命令行剩余参数作为服务名，被导入时仅暴露函数。
if (process.argv[1] && path.resolve(process.argv[1]) === path.join(import.meta.dirname, 'release-service-ports.mjs')) {
  releaseFixedServicePorts(process.argv.slice(2));
}
