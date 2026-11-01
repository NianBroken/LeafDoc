#!/bin/sh
# 系统级安装及生命周期入口，不构建源码，不依赖宿主应用运行库。
set -eu
action=${1:-status}
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
install_dir=/opt/leafdoc
config_dir=/etc/leafdoc
data_dir=/var/lib/leafdoc
[ "$(id -u)" = 0 ] || { echo '请使用 sudo 执行系统级安装管理。' >&2; exit 1; }
docker info >/dev/null
docker compose version >/dev/null
compose() { docker compose --project-name leafdoc --env-file "$config_dir/environment" -f "$install_dir/compose.yaml" "$@"; }
case "$action" in
install|upgrade)
  # 路径前缀作为实例参数持久保存，采用可直接写入环境文件的 URL 路径字符。
  case "${LEAFDOC_BASE_PATH:-}" in ''|/*) ;; *) echo '路径前缀必须为空或以 / 开头' >&2; exit 1;; esac
  case "${LEAFDOC_BASE_PATH:-}" in *[!A-Za-z0-9/_.~%-]*) echo '路径前缀包含无效字符，请使用 URL 路径编码' >&2; exit 1;; esac
  cd "$root"
  sha256sum -c SHA256SUMS
  docker load -i images.tar
  install -d -m 755 "$install_dir"
  install -d -m 700 "$config_dir" "$data_dir" "$config_dir/tls"
  install -m 644 compose.yaml "$install_dir/compose.yaml"
  install -m 755 install.sh "$install_dir/manage.sh"
  install -m 644 manifest.json "$install_dir/manifest.json"
  # 版本资料可替换，用户部署参数和实例数据在升级时原样保留。
  if [ -f "$config_dir/environment" ]; then
    grep -v '^LEAFDOC_IMAGE=\|^LEAFDOC_CADDY_IMAGE=' "$config_dir/environment" > "$config_dir/environment.next"
    grep '^LEAFDOC_IMAGE=\|^LEAFDOC_CADDY_IMAGE=' release.env >> "$config_dir/environment.next"
    mv "$config_dir/environment.next" "$config_dir/environment"
  else
    install -m 600 release.env "$config_dir/environment"
    printf 'LEAFDOC_BASE_PATH=%s\n' "${LEAFDOC_BASE_PATH:-}" >> "$config_dir/environment"
  fi
  if [ ! -e "$config_dir/application.env" ]; then install -m 600 /dev/null "$config_dir/application.env"; fi
  . "$config_dir/environment"
  docker run --rm --user 0 --entrypoint sh \
    --mount "type=bind,source=$config_dir,target=/config" \
    --mount "type=bind,source=$data_dir,target=/instance-data" \
    -e "LEAFDOC_FRONTEND_PORT=$LEAFDOC_FRONTEND_PORT" \
    -e "LEAFDOC_BACKEND_PORT=$LEAFDOC_BACKEND_PORT" \
    -e "LEAFDOC_BASE_PATH=${LEAFDOC_BASE_PATH:-}" \
    -e "LEAFDOC_DOMAIN=${LEAFDOC_DOMAIN:-}" \
    -e "LEAFDOC_HOSTS=${LEAFDOC_HOSTS:-$(hostname -I)}" \
    "$LEAFDOC_IMAGE" /prepare-instance.sh
  docker run --rm --mount "type=bind,source=$config_dir,target=/etc/caddy" "$LEAFDOC_CADDY_IMAGE" caddy fmt --overwrite /etc/caddy/Caddyfile
  compose config --quiet
  compose up -d
  ready=false
  for attempt in $(seq 1 60); do
    # 先读取内核监听状态，避免在端口尚未创建时发起必然失败的健康请求。
    if compose exec -T app sh -c 'port=$(printf "%04X" "$1"); awk -v port="$port" '\''$4 == "0A" && substr($2, length($2)-3) == port { found=1 } END { exit !found }'\'' /proc/net/tcp /proc/net/tcp6' sh "$LEAFDOC_BACKEND_PORT"; then
      if compose exec -T app wget -q -O /dev/null "http://127.0.0.1:$LEAFDOC_BACKEND_PORT${LEAFDOC_BASE_PATH:-}/api/health"; then ready=true; break; fi
    fi
    sleep 1
  done
  [ "$ready" = true ] || { echo '应用未在限定时间内就绪，请检查 logs。' >&2; exit 1; }
  if [ ! -f "$data_dir/installation.json" ]; then printf '首次安装令牌: '; cat "$data_dir/setup-token"; printf '\n'; fi
  printf '本机和局域网入口: https://主机地址:%s\n' "$LEAFDOC_FRONTEND_PORT"
  ;;
start) compose up -d;;
stop) compose stop;;
restart) compose restart;;
status) compose ps;;
logs) compose logs --tail 100;;
ca) cp "$config_dir/tls/local-ca.crt" "$config_dir/local-ca.crt"; chmod 644 "$config_dir/local-ca.crt"; printf '本机和客户端可导入此 CA: %s/local-ca.crt\n' "$config_dir";;
uninstall) compose down;;
purge)
  [ "${2:-}" = '--confirm-delete-instance' ] || { echo '清除实例需要 --confirm-delete-instance，外部数据库不会删除。' >&2; exit 1; }
  compose down -v
  rm -rf -- "$data_dir" "$config_dir" "$install_dir"
  ;;
*) echo '操作: install upgrade start stop restart status logs ca uninstall purge' >&2; exit 1;;
esac
