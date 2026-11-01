#!/bin/sh
# 只回收传入的固定宿主端口，先停止对应容器，再处理普通监听进程。
set -eu
[ "$(id -u)" = 0 ] || { echo 'Port reclamation requires root.' >&2; exit 1; }
[ "$#" -gt 0 ] || { echo 'At least one fixed port is required.' >&2; exit 1; }
for port in "$@"; do
  case "$port" in ''|*[!0-9]*) echo 'Invalid fixed port.' >&2; exit 1;; esac
  [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || exit 1
done
# 缺失的端口诊断工具通过宿主系统包管理器安装。
if ! command -v ss >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then apt-get update; apt-get install -y iproute2
  elif command -v dnf >/dev/null 2>&1; then dnf install -y iproute
  elif command -v yum >/dev/null 2>&1; then yum install -y iproute
  elif command -v apk >/dev/null 2>&1; then apk add --no-cache iproute2
  else echo 'No supported system package manager for iproute2.' >&2; exit 1; fi
fi
# 按 Docker 的实际宿主绑定精确匹配，不结束 Docker 引擎或共享代理。
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  for container in $(docker ps -q); do
    bindings=$(docker inspect --format '{{range $port, $bindings := .NetworkSettings.Ports}}{{range $bindings}}{{println .HostPort}}{{end}}{{end}}' "$container")
    matched=false
    for binding in $bindings; do
      for port in "$@"; do [ "$binding" != "$port" ] || matched=true; done
    done
    if [ "$matched" = true ]; then docker stop --time 30 "$container"; fi
  done
fi
owners() {
  for port in "$@"; do
    ss -H -ltnp "sport = :$port"
    ss -H -lunp "sport = :$port"
  done
}
pids=$(owners "$@" | sed -n 's/.*pid=\([0-9][0-9]*\).*/\1/p' | sort -u)
for process in $pids; do
  [ "$process" -gt 1 ] && [ "$process" != "$$" ] && [ "$process" != "$PPID" ] || { echo 'Protected listener owner.' >&2; exit 1; }
  name=$(cat "/proc/$process/comm")
  case "$name" in dockerd|containerd|docker-proxy|systemd) echo "Protected shared listener: $name" >&2; exit 1;; esac
  # systemd 服务先停止所属单元，防止进程被自动拉起。
  if command -v systemctl >/dev/null 2>&1; then
    unit=$(sed -n 's|.*/\([^/]*\.service\)$|\1|p' "/proc/$process/cgroup" | head -n 1)
    case "$unit" in ''|init.scope|docker.service|containerd.service) ;; *) systemctl stop "$unit";; esac
  fi
  if kill -0 "$process" 2>/dev/null; then kill -TERM "$process"; fi
done
for attempt in 1 2 3 4 5; do
  [ -n "$(owners "$@")" ] || exit 0
  sleep 1
done
for process in $pids; do
  if kill -0 "$process" 2>/dev/null; then kill -KILL "$process"; fi
done
for attempt in 1 2 3 4 5; do
  [ -n "$(owners "$@")" ] || exit 0
  sleep 1
done
echo 'Fixed host ports remain occupied.' >&2
exit 1
