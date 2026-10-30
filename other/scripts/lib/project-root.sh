#!/usr/bin/env bash
# Resolve the checkout through the same authority as the Windows and Node launchers.
leafdoc_project_root() {
    node "$(dirname "${BASH_SOURCE[0]}")/../project-root.mjs"
}
