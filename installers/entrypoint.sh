#!/bin/sh
# 新数据卷首次启动时生成安装令牌，已有实例配置与令牌保持原样。
set -eu
data_dir=${LEAFDOC_DATA_DIR:-/app/data}
mkdir -p "$data_dir"
if [ ! -e "$data_dir/installation.json" ] && [ ! -e "$data_dir/setup-token" ]; then
    openssl rand -hex 32 > "$data_dir/setup-token.next"
    mv "$data_dir/setup-token.next" "$data_dir/setup-token"
fi
exec /docker-entrypoint.sh "$@"
