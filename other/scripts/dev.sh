#!/usr/bin/env bash
# Windows development uses the same persistent Linux application image as production.
set -euo pipefail
. "$(dirname "$0")/lib/project-root.sh"
root="$(leafdoc_project_root)"
exec pwsh -NoProfile -File "$root/other/scripts/start.ps1" "$@"
