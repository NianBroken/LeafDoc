#!/usr/bin/env bash
set -euo pipefail

# Resolve the same release metadata used by Windows and frontend builds.
exec node "$(dirname "$0")/resolve-version.mjs"
