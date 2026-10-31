#!/usr/bin/env bash
# 部署入口共用的路径、环境文件与就绪检查。私有环境文件仅接受变量赋值。
set -euo pipefail
leafdoc_load_environment() {
    local file="$1" line name value
    test -r "$file" || { printf 'Environment file is not readable: %s\n' "$file" >&2; return 1; }
    while IFS= read -r line || [ -n "$line" ]; do
        line="${line%$'\r'}"
        case "$line" in ''|'#'*) continue;; esac
        [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]] || { echo 'Invalid environment assignment.' >&2; return 1; }
        name="${BASH_REMATCH[1]}"; value="${BASH_REMATCH[2]}"
        if [[ "$value" == \"*\" || "$value" == \'*\' ]]; then value="${value:1:${#value}-2}"; fi
        if [ -z "${!name+x}" ]; then export "$name=$value"; fi
    done < "$file"
}
leafdoc_wait_url() {
    local url="$1" ca="${2:-}" attempt
    local args=(--fail --silent --show-error --connect-timeout 3 --max-time 5)
    if [ -n "$ca" ]; then args+=(--cacert "$ca"); fi
    for attempt in {1..120}; do
        if curl "${args[@]}" "$url" > /dev/null 2> /tmp/leafdoc-readiness-$$; then
            rm -f /tmp/leafdoc-readiness-$$; return 0
        fi
        sleep 1
    done
    cat /tmp/leafdoc-readiness-$$ >&2
    rm -f /tmp/leafdoc-readiness-$$
    printf 'Service readiness timed out: %s\n' "$url" >&2
    return 1
}
leafdoc_require_root() {
    [ "$(id -u)" = 0 ] || { echo 'Run this operation as root.' >&2; return 1; }
}
