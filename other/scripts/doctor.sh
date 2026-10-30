#!/usr/bin/env bash
# Host checks require only the dispatcher and Linux Docker engine.
set -euo pipefail
command -v node >/dev/null
command -v docker >/dev/null
node --version
docker version --format '{{.Client.Version}}'
test "$(docker info --format '{{.OSType}}')" = linux
echo 'Host ready; Go, npm and frontend dependencies run inside Linux images.'
