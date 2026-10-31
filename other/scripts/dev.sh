#!/usr/bin/env bash
# 此入口定位项目后调用 PowerShell 启动脚本，开发场景也运行持久 Linux 应用容器。
# 所有调用参数按原边界转交，实际容器、网关和端口处理由 start.ps1 执行。
set -euo pipefail
# 共享定位函数先于路径使用加载，任何失败在严格 shell 模式下终止。
. "$(dirname "$0")/lib/project-root.sh"
root="$(leafdoc_project_root)"
exec pwsh -NoProfile -File "$root/other/scripts/start.ps1" "$@"
