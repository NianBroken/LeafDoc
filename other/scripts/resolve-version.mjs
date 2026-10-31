// 项目版本只读取集中元数据，不接受环境变量或 Git 标签覆盖。
import fs from 'node:fs';
import path from 'node:path';
import { findProjectRoot } from './project-root.mjs';
const metadata = JSON.parse(fs.readFileSync(path.join(findProjectRoot(), 'other/project.json'), 'utf8'));
process.stdout.write(metadata.version + '\n');
