// 本脚本把构建、依赖安装与验证命令交给 Linux 容器执行。
// 宿主只负责定位项目、读取版本配置及调用 Docker，不在宿主安装项目依赖。
import { spawnSync, execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { findProjectRoot } from './project-root.mjs';
// 项目位置动态定位，镜像标签与命令名称集中声明，未给动作时执行应用构建。
const root = findProjectRoot();
const image = 'leafdoc:local';
const tools = 'leafdoc-tools:local';
const action = process.argv[2] || 'build';
// execute 同步执行 Docker 并继承输出，Windows 下隐藏额外命令窗口。
// 启动错误与非零退出码均终止后续流程，不把失败包装为验证成功。
function execute(args) {
  const result = spawnSync('docker', args, { cwd: root, stdio: 'inherit', windowsHide: true });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`Docker command failed with exit code ${result.status ?? 1}`);
}
// 所有动作共用 Linux 容器前置检查，Docker 不可访问或处于其他容器模式时直接拒绝。
const system = spawnSync('docker', ['info', '--format', '{{.OSType}}'], { encoding: 'utf8', windowsHide: true });
if (system.error) throw system.error;
if (system.status !== 0 || system.stdout.trim() !== 'linux') throw new Error('Docker must be running Linux containers');
// build 和 release 在此使用同一构建路径，不由这个入口发布到远端仓库。
if (action === 'build' || action === 'release') {
  const version = execFileSync(process.execPath, [fileURLToPath(new URL('./resolve-version.mjs', import.meta.url))], { encoding: 'utf8' }).trim();
  execute(['build', '-f', 'other/Dockerfile', '--build-arg', `APP_VERSION=${version}`, '-t', image, '.']);
} else {
  // 每个验证动作映射为容器内命令，shell 的错误退出使串联检查在失败处结束。
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
  // 生成动作使用随机名称保留退出后的容器，复制两份输出后在 finally 中删除容器。
  // 文件复制依次执行，不为宿主上的两份生成文件提供整体回滚。
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
  // 只接受命令表中显式登记的动作，用户输入不作为任意 shell 命令拼接执行。
  if (!Object.hasOwn(commands, action)) throw new Error(`Unknown container action: ${action}`);
  execute(['build', '-f', 'other/Dockerfile.tools', '-t', tools, '.']);
  let executionImage = tools;
  // 后端 lint 单独构建带固定工具版本的镜像，版本取自项目版本文件。
  if (action === 'lint') {
    const version = readFileSync(new URL('../.golangci-lint-version', import.meta.url), 'utf8').trim();
    executionImage = 'leafdoc-lint:local';
    execute(['build', '-f', 'other/Dockerfile.lint', '--build-context', `lint-toolchain=docker-image://golangci/golangci-lint:${version}`, '-t', executionImage, '.']);
  }
  // 验证容器退出即删除，Go 构建缓存通过命名卷跨调用复用，源码来自工具镜像构建。
  execute(['run', '--rm', '--mount', 'type=volume,source=leafdoc-linux-go-cache,target=/root/.cache/go-build', '-e', 'GOOS=linux', executionImage, 'sh', '-ec', commands[action]]);
}
