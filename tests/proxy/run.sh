#!/usr/bin/env bash
# Use the same cross-platform runner as Windows, with guaranteed private database cleanup.
set -euo pipefail
exec node "$(dirname "$0")/run.mjs" "$@"
