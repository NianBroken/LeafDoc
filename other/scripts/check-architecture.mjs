// 本脚本解析前端源码的静态导入与导出，检查共享层依赖、跨功能公开入口和运行依赖循环。
// 只读取源码及接口清单，不执行模块，也不尝试分析任意运行时动态导入。
import fs from "node:fs";
import path from "node:path";
import { findProjectRoot } from "./project-root.mjs";
import { createRequire } from "node:module";
const root = findProjectRoot();
const source = path.join(root, "frontend", "src");
// 从前端包位置加载 TypeScript，使用与项目依赖相同的解析器。
const ts = createRequire(path.join(root, "frontend", "package.json"))("typescript");
const featureInterfaces = JSON.parse(
  fs.readFileSync(path.join(root, "frontend", "feature-interfaces.json"), "utf8"),
);
const graph = new Map();
const violations = [];
// 递归枚举 TS 与 TSX，跳过测试、声明文件和测试辅助路径。
// 只扫描顶层 import/export 声明，不包含函数体中的 require 或 import 调用。
function collect(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      collect(file);
      continue;
    }
    if (
      !/\.tsx?$/.test(file) ||
      /\.test\./.test(file) ||
      /test-setup|test-utils|[\\/]testutil[\\/]|\.d\.ts$/.test(file)
    )
      continue;
    const code = ts.createSourceFile(
      file,
      fs.readFileSync(file, "utf8"),
      ts.ScriptTarget.Latest,
      true,
    );
    const dependencies = [];
    for (const node of code.statements) {
      if (!ts.isImportDeclaration(node) && !ts.isExportDeclaration(node))
        continue;
      const specifier = node.moduleSpecifier;
      if (!specifier || !ts.isStringLiteral(specifier)) continue;
      const value = specifier.text;
      if (!value.startsWith("@/") && !value.startsWith(".")) continue;
      const target = value.startsWith("@/")
        ? path.join(source, value.slice(2))
        : path.resolve(path.dirname(file), value);
      // 按指定候选顺序寻找文件，不执行 TypeScript 的完整模块解析规则。
      const resolved = [
        target,
        target + ".ts",
        target + ".tsx",
        path.join(target, "index.ts"),
        path.join(target, "index.tsx"),
      ].find(
        (candidate) =>
          fs.existsSync(candidate) && fs.statSync(candidate).isFile(),
      );
      if (!resolved || !/\.tsx?$/.test(resolved)) continue;
      if (
        file.startsWith(path.join(source, "shared") + path.sep) &&
        !resolved.startsWith(path.join(source, "shared") + path.sep)
      )
        violations.push(
          `${path.relative(source, file)} imports ${path.relative(source, resolved)}`,
        );
      // 跨功能依赖须命中目标功能公开清单，共享层只能依赖共享目录。
      // 这些边界判断先于类型导入过滤，因此类型引用也需要遵守边界。
      const fromParts = path.relative(source, file).split(path.sep);
      const toParts = path.relative(source, resolved).split(path.sep);
      if (
        fromParts[0] === "features" &&
        toParts[0] === "features" &&
        fromParts[1] !== toParts[1]
      ) {
        const entry = toParts
          .slice(2)
          .join("/")
          .replace(/\.tsx?$/, "")
          .replace(/\/index$/, "");
        if (!featureInterfaces[toParts[1]]?.includes(entry))
          violations.push(
            `${path.relative(source, file)} imports private feature module ${path.relative(source, resolved)}`,
          );
      }
      // 完全属于类型的声明不加入运行依赖图，混合导入只要含运行值仍记录整条模块边。
      if (node.isTypeOnly || node.importClause?.isTypeOnly) continue;
      if (
        node.importClause?.namedBindings &&
        ts.isNamedImports(node.importClause.namedBindings) &&
        !node.importClause.name &&
        node.importClause.namedBindings.elements.every(
          (element) => element.isTypeOnly,
        )
      )
        continue;
      dependencies.push(resolved);
    }
    graph.set(file, dependencies);
  }
}
// 图建立完毕后用深度优先遍历查找回到当前递归栈的边。
collect(source);
const visiting = new Set(),
  visited = new Set();
// visit 分开记录当前递归栈和已经检查完成的节点，避免把普通共享依赖误判为环。
// 遇到环只记录路径，不终止其余检查，便于一次呈现多个违规点。
function visit(file, stack) {
  if (visiting.has(file)) {
    violations.push(
      "cycle: " +
        [...stack.slice(stack.indexOf(file)), file]
          .map((item) => path.relative(source, item))
          .join(" -> "),
    );
    return;
  }
  if (visited.has(file)) return;
  visiting.add(file);
  for (const dependency of graph.get(file) ?? [])
    visit(dependency, [...stack, file]);
  visiting.delete(file);
  visited.add(file);
}
for (const file of graph.keys()) visit(file, []);
// 集中输出发现的违规并设置失败退出码，无违规时报告已检查模块数量。
if (violations.length) {
  process.stderr.write(violations.join("\n") + "\n");
  process.exitCode = 1;
} else
  process.stdout.write(
    `Architecture verified: ${graph.size} source modules.\n`,
  );
