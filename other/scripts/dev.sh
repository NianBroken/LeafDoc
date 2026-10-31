#!/usr/bin/env bash
# 此入口运行 Ubuntu 容器部署，所有参数按原边界传递。
set -euo pipefail
# 共享定位函数先于路径使用加载，任何失败在严格 shell 模式下终止。
. "$(dirname "$0")/lib/project-root.sh"
root="$(leafdoc_project_root)"
exec bash "$root/other/scripts/deploy.sh" "${@:-start}"
