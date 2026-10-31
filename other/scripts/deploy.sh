#!/usr/bin/env bash
# Ubuntu Compose 管理入口。配置保存在系统目录，业务数据使用命名卷。
set -euo pipefail
. "$(dirname "$0")/lib/project-root.sh"
. "$(dirname "$0")/lib/deployment.sh"
root="$(leafdoc_project_root)"
action="${1:-status}"
config_dir="${LEAFDOC_CONFIG_DIR:-/etc/leafdoc-compose}"
environment="$config_dir/environment"
case "$action" in
init)
    leafdoc_require_root
    shift
    exec python3 "$root/other/scripts/configure-deployment.py" --mode docker --directory "$config_dir" "$@"
    ;;
build)
    exec node "$root/other/scripts/container.mjs" build
    ;;
start|stop|restart|status|logs|config)
    leafdoc_load_environment "$environment"
    export LEAFDOC_CONFIG_DIR="$config_dir"
    args=(--project-name leafdoc --env-file "$environment" -f "$root/other/compose.yaml")
    if [ "${LEAFDOC_TLS_MODE:-local}" = public ]; then args+=(-f "$root/other/compose.public.yaml"); fi
    case "$action" in
    start)
        docker compose "${args[@]}" up -d
        exec python3 "$root/other/scripts/verify-deployment.py" --url "https://localhost:$LEAFDOC_FRONTEND_PORT" --ca "$config_dir/tls/local-ca.crt" --environment "$environment"
        ;;
    stop) exec docker compose "${args[@]}" stop;;
    restart) docker compose "${args[@]}" restart; exec bash "$0" start;;
    status) exec docker compose "${args[@]}" ps;;
    logs) exec docker compose "${args[@]}" logs --tail 100;;
    config) exec docker compose "${args[@]}" config --quiet;;
    esac
    ;;
*) echo 'Commands: init, build, start, stop, restart, status, logs, config' >&2; exit 1;;
esac
