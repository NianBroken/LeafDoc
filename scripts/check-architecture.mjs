// Verifies static runtime dependencies and keeps shared code independent of features.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createRequire } from "node:module";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = path.join(root, "ui", "src");
const ts = createRequire(path.join(root, "ui", "package.json"))("typescript");
const featureInterfaces = JSON.parse(
  fs.readFileSync(path.join(root, "ui", "feature-interfaces.json"), "utf8"),
);
const graph = new Map();
const violations = [];
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
      // Cross-feature calls use declared public modules; application composition owns component wiring.
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
collect(source);
const visiting = new Set(),
  visited = new Set();
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
if (violations.length) {
  process.stderr.write(violations.join("\n") + "\n");
  process.exitCode = 1;
} else
  process.stdout.write(
    `Architecture verified: ${graph.size} source modules.\n`,
  );
