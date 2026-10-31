// 本模块从给定位置向上寻找项目标记，让源码检出与容器快照共用定位入口。
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

// findProjectRoot 默认从本模块目录开始。URL 参数按文件地址转换后取父目录，
// 普通路径按宿主路径规则解析；同时发现前端清单和定位脚本才认定为项目根。
export function findProjectRoot(start = import.meta.dirname) {
  let directory = start instanceof URL ? path.dirname(fileURLToPath(start)) : path.resolve(start);
  for (;;) {
    if (existsSync(path.join(directory, 'frontend/package.json')) &&
        existsSync(path.join(directory, 'other/scripts/project-root.mjs'))) return directory;
    // 到达文件系统根仍未命中时明确失败，不继续重复检查同一目录。
    const parent = path.dirname(directory);
    if (parent === directory) throw new Error('LeafDoc source root was not found.');
    directory = parent;
  }
}

// 作为命令直接运行时只输出根路径，被其他模块导入时不写标准输出。
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.stdout.write(findProjectRoot() + '\n');
}
