// 唯一发行入口：快照当前源码，重新编译镜像，导出可离线安装的完整服务包。
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { spawnSync, execFileSync } from 'node:child_process';
import { findProjectRoot } from './project-root.mjs';

const root = findProjectRoot();
const output = path.join(root, 'installers');
const metadata = JSON.parse(fs.readFileSync(path.join(root, 'other/project.json'), 'utf8'));
const platform = 'linux/amd64';
const gateway = 'caddy:2-alpine';
const excluded = /(^|\/)(node_modules|dist|installers|data|test-results|releases|\.git)(\/|$)|(^|\/)\.env($|\.)|\.exe$|\.tsbuildinfo$/;
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'leafdoc-package-'));
const snapshot = path.join(temporary, 'source');
const bundle = path.join(temporary, 'bundle');

// 所有命令直接传递参数数组，避免中文路径和 shell 元字符改变执行语义。
function run(command, args, options = {}) {
  const result = spawnSync(command, args, { cwd: root, stdio: 'inherit', windowsHide: true, ...options });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} 执行失败，退出码 ${result.status}`);
}
function digest(file) { return crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex'); }
function files() {
  return execFileSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z'], { cwd: root, encoding: 'utf8' })
    .split('\0').filter(Boolean).filter(file => !excluded.test(file) && fs.existsSync(path.join(root, file))).sort();
}
function inventory() { return files().map(file => ({ file, sha256: digest(path.join(root, file)) })); }

try {
  if (!/^[A-Za-z][A-Za-z0-9_-]*$/.test(metadata.name) || !/^\d+\.\d+\.\d+$/.test(metadata.version)) throw new Error('项目名称或版本格式无效');
  if (fs.existsSync(output) && (fs.lstatSync(output).isSymbolicLink() || fs.realpathSync(output) !== path.resolve(output))) throw new Error('installers 不得指向其他目录');
  fs.mkdirSync(output, { recursive: true });
  for (const entry of fs.readdirSync(output)) fs.rmSync(path.join(output, entry), { recursive: true, force: true });
  run('docker', ['info', '--format', '{{.OSType}}']);
  const before = inventory();
  const sourceHash = crypto.createHash('sha256').update(JSON.stringify(before)).digest('hex');
  for (const { file } of before) {
    const source = path.join(root, file);
    if (fs.lstatSync(source).isSymbolicLink()) throw new Error(`源码快照不接受符号链接: ${file}`);
    const target = path.join(snapshot, file); fs.mkdirSync(path.dirname(target), { recursive: true }); fs.copyFileSync(source, target);
  }
  const d = new Date();
  const pad = (v, n = 2) => String(v).padStart(n, '0');
  const timestamp = `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}${pad(Math.floor(d.getMilliseconds() / 10))}`;
  const name = `${metadata.name}_${metadata.version}_${timestamp}`;
  const image = `${metadata.name.toLowerCase()}:${metadata.version}-${timestamp}`;
  run('docker', ['build', '--platform', platform, '--no-cache', '-f', path.join(snapshot, 'other/Dockerfile'), '--build-arg', `APP_VERSION=${metadata.version}`, '--build-arg', `APP_NAME=${metadata.name}`, '--label', `org.opencontainers.image.revision=${sourceHash}`, '-t', image, snapshot]);
  run('docker', ['pull', '--platform', platform, gateway]);
  fs.mkdirSync(bundle);
  run('docker', ['save', '-o', path.join(bundle, 'images.tar'), image, gateway]);
  const ports = JSON.parse(fs.readFileSync(path.join(root, 'backend/internal/core/serviceports/ports.json'), 'utf8'));
  fs.copyFileSync(path.join(root, 'other/scripts/install-bundle.sh'), path.join(bundle, 'install.sh'));
  fs.copyFileSync(path.join(root, 'other/docker-packaging.md'), path.join(bundle, 'README.md'));
  fs.writeFileSync(path.join(bundle, 'release.env'), `LEAFDOC_IMAGE=${image}\nLEAFDOC_CADDY_IMAGE=${gateway}\nLEAFDOC_FRONTEND_PORT=${ports.frontend}\nLEAFDOC_BACKEND_PORT=${ports.backend}\n`);
  fs.writeFileSync(path.join(bundle, 'compose.yaml'), fs.readFileSync(path.join(root, 'other/compose.bundle.yaml')));
  fs.writeFileSync(path.join(bundle, 'manifest.json'), JSON.stringify({ ...metadata, platform, builtAt: d.toISOString(), sourceHash, files: before, images: [image, gateway] }, null, 2));
  fs.writeFileSync(path.join(bundle, 'SHA256SUMS'), fs.readdirSync(bundle).map(file => `${digest(path.join(bundle, file))}  ${file}\n`).join(''));
  if (JSON.stringify(before) !== JSON.stringify(inventory())) throw new Error('构建期间源码发生变化，请重新打包');
  run('docker', ['run', '--rm', '--user', '0', '--entrypoint', 'sh', '--mount', `type=bind,source=${temporary},target=/work`, image, '-ec', `set -o pipefail; cd /work/bundle && tar -cf - . | gzip -9 > /work/${name}.tar.gz`]);
  fs.copyFileSync(path.join(temporary, `${name}.tar.gz`), path.join(output, `${name}.tar.gz`));
  console.log(`产物: ${path.join(output, name + '.tar.gz')}\nSHA256: ${digest(path.join(output, name + '.tar.gz'))}\n源码摘要: ${sourceHash}`);
} finally { fs.rmSync(temporary, { recursive: true, force: true }); }
