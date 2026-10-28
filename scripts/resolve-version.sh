#!/usr/bin/env bash
set -euo pipefail

# Resolves the backend release version for main.Version and release artifacts.
if [ -n "${APP_VERSION:-}" ]; then
  echo "$APP_VERSION"
  exit 0
fi

git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0"
