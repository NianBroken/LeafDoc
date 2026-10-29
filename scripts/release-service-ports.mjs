// Development and browser runners reclaim the same fixed listeners on every platform.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = fileURLToPath(new URL("../", import.meta.url));
const allocation = JSON.parse(
  fs.readFileSync(
    path.join(root, "internal/core/serviceports/ports.json"),
    "utf8",
  ),
);

export function releaseFixedServicePorts(services) {
  if (
    !services.length ||
    services.some((service) => !Object.hasOwn(allocation, service))
  ) {
    throw new Error(
      "Explicit names from the fixed service allocation are required",
    );
  }
  if (process.platform === "win32") {
    const result = spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        path.join(root, "scripts/release-service-ports.ps1"),
        "-Services",
        services.join(","),
      ],
      { stdio: "inherit", windowsHide: true },
    );
    if (result.error) throw result.error;
    if (result.status !== 0)
      throw new Error("Could not release the fixed service ports");
    return;
  }
  for (const service of services) {
    for (const protocol of ["TCP", "UDP"]) {
      const args = ["-t", `-i${protocol}:${allocation[service]}`];
      if (protocol === "TCP") args.push("-sTCP:LISTEN");
      const result = spawnSync("lsof", args, { encoding: "utf8" });
      if (result.error) throw result.error;
      if (result.status !== 0 && result.status !== 1)
        throw new Error(result.stderr.trim());
      for (const value of new Set(
        result.stdout.trim().split(/\s+/).filter(Boolean),
      )) {
        const owner = Number(value);
        if (
          !Number.isSafeInteger(owner) ||
          owner <= 1 ||
          owner === process.pid
        ) {
          throw new Error("Invalid listener process owner");
        }
        try {
          process.kill(owner, "SIGKILL");
        } catch (error) {
          if (error.code !== "ESRCH") throw error;
        }
      }
    }
  }
}

if (
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
  releaseFixedServicePorts(process.argv.slice(2));
}
