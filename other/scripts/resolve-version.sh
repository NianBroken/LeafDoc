#!/usr/bin/env bash
set -euo pipefail

# 版本选择全部交给同目录 Node 入口，避免 shell 自行维护另一套默认值。
# exec 用 Node 替换当前 shell，直接保留输出、参数约定和退出状态。
exec node "$(dirname "$0")/resolve-version.mjs"
