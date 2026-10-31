#!/usr/bin/env bash
# 从当前 Ubuntu 发行版对应的官方签名仓库安装兼容引擎和 Compose 插件。
set -euo pipefail
. "$(dirname "$0")/lib/deployment.sh"
leafdoc_require_root
. /etc/os-release
case "$ID:$VERSION_ID" in ubuntu:18.04|ubuntu:20.04|ubuntu:22.04) ;; *) echo 'Unsupported Ubuntu release.' >&2; exit 1;; esac
apt-get update -o Acquire::Retries=3
DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl gnupg
install -d -m 755 /etc/apt/keyrings
key="$(mktemp)"
trap 'rm -f "$key"' EXIT
curl --fail --silent --show-error https://download.docker.com/linux/ubuntu/gpg -o "$key"
gpg --batch --yes --dearmor -o /etc/apt/keyrings/docker.gpg "$key"
chmod 644 /etc/apt/keyrings/docker.gpg
printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu %s stable\n' "$(dpkg --print-architecture)" "$VERSION_CODENAME" > /etc/apt/sources.list.d/leafdoc-docker.list
apt-get update
# apt 根据发行版仓库解析兼容包，不把新发行版包混入旧系统。
DEBIAN_FRONTEND=noninteractive apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
systemctl enable --now docker
docker version
docker compose version
install -d -m 755 /var/lib/leafdoc-deployment
dpkg-query -W docker-ce docker-ce-cli containerd.io docker-compose-plugin > /var/lib/leafdoc-deployment/docker-versions.txt
