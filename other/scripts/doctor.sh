#!/usr/bin/env bash
# 本脚本检查宿主能找到 Node 与 Docker，输出版本后确认 Docker 处于 Linux 模式。
# 不编译应用、不安装依赖，也不验证当前业务容器或登录流程是否健康。
set -euo pipefail
# 严格模式使缺少命令、版本查询失败或容器模式不符直接以失败结束。
command -v node >/dev/null
command -v docker >/dev/null
node --version
docker version --format '{{.Client.Version}}'
test "$(docker info --format '{{.OSType}}')" = linux
echo 'Host ready; Go, npm and frontend dependencies run inside Linux images.'
