# Sourced by check-toolchain-versions.sh and doctor.sh, which run from the
# repository root: the one place that parses the two version sources.

# The full Go version from go.mod's toolchain line, e.g. 1.27.3; empty
# without a toolchain go<major>.<minor>.<patch> line.
toolchain_go_version() {
    local file="${1:-go.mod}"
    sed -n -E 's/^toolchain go([0-9]+\.[0-9]+\.[0-9]+).*/\1/p' "$file" 2>/dev/null
}

# The Node major from the first non-blank line of .nvmrc, without a leading
# v: 26, v26 and 26.10.0 give 26. Empty if .nvmrc is missing or has none.
toolchain_node_major() {
    local file="${1:-.nvmrc}"
    awk 'NF { sub(/^v/, "", $1); sub(/[^0-9].*/, "", $1); print $1; exit }' "$file" 2>/dev/null
}
