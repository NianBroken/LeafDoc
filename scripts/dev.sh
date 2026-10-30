#!/usr/bin/env bash
# Windows development uses the same persistent Linux application image as production.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
exec pwsh -NoProfile -File "$root/scripts/start.ps1" "$@"
