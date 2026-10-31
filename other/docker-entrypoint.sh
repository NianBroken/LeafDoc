#!/usr/bin/env sh
# 容器入口确定默认监听地址，再用应用进程替换当前 shell。
# 严格模式使未处理的命令失败和未定义变量终止执行，带默认值的环境读取除外。
set -eu

# 仅在 DEBUG 精确等于 true 时启用 shell 命令跟踪，其余取值不启用。
if [ "${DEBUG:-false}" = "true" ]; then
  set -x
fi

log() {
  # 日志使用固定入口前缀并写入标准错误，不附加时间戳。
  printf '[entrypoint] %s\n' "$*" >&2
}

log "Starting entrypoint script..."

# 扫描所有参数，仅识别独立的 --host 和 --host= 开头形式。
# 此处只判断参数是否出现，不校验其后是否存在合法的主机值，解析由应用完成。
has_host_arg=false
for arg in "$@"; do
  case "$arg" in
    --host|--host=* )
      has_host_arg=true
      break
      ;;
  esac
done

# 参数未提供主机且环境主机为空时监听所有 IPv4 接口，允许容器端口映射访问。
if [ "$has_host_arg" = false ] && [ -z "${LEAFDOC_HOST:-}" ]; then
  log "No --host argument or LEAFDOC_HOST environment variable found. Setting LEAFDOC_HOST to '0.0.0.0'"
  export LEAFDOC_HOST="0.0.0.0"
fi

# 保留参数边界交给应用，exec 使应用直接接收容器停止信号并决定容器退出状态。
log "Executing /app/leafdoc"
exec /app/leafdoc "$@"
