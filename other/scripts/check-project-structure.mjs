// 校验根目录的三个非隐藏业务目录和启动所需文件是否存在。
// 检查只读取文件系统，不创建、删除或移动项目文件。
import fs from 'node:fs';
import path from 'node:path';
import { findProjectRoot } from './project-root.mjs';

const root = findProjectRoot();
const expected = new Set(['backend', 'frontend', 'other']);
// 隐藏目录不参与根目录集合比较，普通文件也不被当作目录校验。
const actual = new Set(fs.readdirSync(root, { withFileTypes: true }).filter((entry) =>
  entry.isDirectory() && !entry.name.startsWith('.')).map((entry) => entry.name));
const failures = [];
for (const name of expected) if (!actual.has(name)) failures.push(`missing root directory: ${name}`);
for (const name of actual) if (!expected.has(name)) failures.push(`unexpected root directory: ${name}`);
// 入口按路径存在性检查，不解析内容，也不区分普通文件与其他路径类型。
for (const file of ['backend/go.mod', 'backend/go.sum', 'frontend/package.json', 'frontend/package-lock.json', 'other/scripts/start.ps1']) {
  if (!fs.existsSync(path.join(root, file))) failures.push(`missing required path: ${file}`);
}
// 列表中的旧根路径只要存在就计为失败，不检查其路径类型。
for (const name of ['cmd', 'internal', 'ui', 'scripts', 'data']) {
  if (fs.existsSync(path.join(root, name))) failures.push(`legacy root path remains: ${name}`);
}
// 汇总全部已发现的结构问题后失败，只有无问题才输出验证成功。
if (failures.length) throw new Error(failures.join('\n'));
process.stdout.write('Project structure verified.\n');
