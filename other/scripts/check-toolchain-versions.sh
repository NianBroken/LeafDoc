#!/bin/bash
# 检查持续集成使用的 Go 与 Node 版本是否和发布镜像一致。Go 版本来自 go.mod 的 toolchain 行，Node 主版本来自 .nvmrc，Dockerfile 另外固定基础镜像标签。比较完整的 Go 版本和 Node 主版本，mise 配置只作为版本引导来源。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
if [[ ! -f "$ROOT/backend/go.mod" && -f "$SCRIPT_DIR/../go.mod" ]]; then
    ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
fi
cd "$ROOT"

if [[ -f backend/go.mod ]]; then
    GO_MANIFEST=backend/go.mod
    NODE_MANIFEST=other/.nvmrc
    SCRIPT_ROOT=other/scripts
    DOCKERFILES=(other/Dockerfile other/Dockerfile.tools)
else
    GO_MANIFEST=go.mod
    NODE_MANIFEST=.nvmrc
    SCRIPT_ROOT=scripts
    DOCKERFILES=(Dockerfile Dockerfile.tools)
fi
# shellcheck source=other/scripts/lib/toolchain-versions.sh
. "$SCRIPT_ROOT/lib/toolchain-versions.sh"
status=0

fail() {
    echo "::error file=$1::$2"
    status=1
}

# 提取每个 FROM 镜像标签中的数字版本，每行输出一个版本。支持 FROM 大小写、--platform 等参数以及 docker.io/library 前缀。
image_versions() {
    awk -v image="$2" '
        { sub(/\r$/, "") }
        tolower($1) == "from" {
            i = 2
            while (i <= NF && $i ~ /^--/) i++
            ref = $i
            sub(/^docker\.io\//, "", ref)
            sub(/^library\//, "", ref)
            if (index(ref, image ":") != 1) next
            tag = substr(ref, length(image) + 2)
            if (match(tag, /^[0-9][0-9.]*/)) print substr(tag, 1, RLENGTH)
        }
    ' "$1"
}

go_version="$(toolchain_go_version "$GO_MANIFEST" || true)"
if [[ -z "$go_version" ]]; then
    fail "$GO_MANIFEST" "$GO_MANIFEST has no toolchain go<major>.<minor>.<patch> line; CI reads the Go version from it."
fi

node_version="$(toolchain_node_major "$NODE_MANIFEST" || true)"
if [[ -z "$node_version" ]]; then
    fail "$NODE_MANIFEST" "$NODE_MANIFEST is missing or has no Node major version; CI reads the Node version from it."
fi

# 检查 Dockerfile 中指定镜像的每个 FROM 行，按点号字段截取后必须与来源文件中的期望值一致。
check() {
    local dockerfile="$1" image="$2" label="$3" want="$4" source="$5" fields="$6"
    local versions version
    [[ -n "$want" ]] || return 0
    versions="$(image_versions "$dockerfile" "$image")"
    if [[ -z "$versions" ]]; then
        fail "$dockerfile" "No $image image in $dockerfile, but $source has $label $want."
        return 0
    fi
    while IFS= read -r version; do
        version="$(cut -d. -f"$fields" <<< "$version")"
        if [[ "$version" != "$want" ]]; then
            fail "$dockerfile" "$label $version in $dockerfile, but $source has $want. Update both together."
        fi
    done <<< "$versions"
}

for dockerfile in "${DOCKERFILES[@]}"; do
    if [[ ! -f "$dockerfile" ]]; then
        fail "$dockerfile" "$dockerfile is missing."
        continue
    fi
    check "$dockerfile" golang Go "$go_version" "$GO_MANIFEST's toolchain line" 1-3
    check "$dockerfile" node Node "$node_version" "$NODE_MANIFEST" 1
done

# 已提交的 mise 配置不自行固定 Node 和 Go 版本，Node 读取 .nvmrc，Go 读取 go.mod 的 toolchain 行。检查器在去除注释后严格识别各种 TOML 写法，除读取 .nvmrc 的设置外，出现 node 都失败，go 只允许 latest 或完整 toolchain 版本。工具版本文件和 Dev Container 配置不能声明这两个工具，用户本地覆盖文件保持不检查。
# mise_violations 会为每个违规项输出行号和原始行。
mise_violations() {
    local strict="" json=""
    case "$1" in
        .tool-versions) strict=1 ;;
        .devcontainer.json | .devcontainer/devcontainer.json | .devcontainer/*/devcontainer.json) strict=1 json=1 ;;
    esac
    awk -v strict="$strict" -v json="$json" -v want="$go_version" '
        # 截取 JSON 字符串之外的 // 注释前内容，字符串内的 URL 不作为注释。
        function json_code(s,    i, c, quoted) {
            for (i = 1; i <= length(s); i++) {
                c = substr(s, i, 1)
                if (quoted && c == "\\") i++
                else if (c == "\"") quoted = !quoted
                else if (!quoted && substr(s, i, 2) == "//") return substr(s, 1, i - 1)
            }
            return s
        }
        { sub(/\r$/, ""); if (json) $0 = json_code($0); else sub(/#.*/, "")
          line = tolower($0); bare = $0; gsub(/[ \t]/, "", bare) }
        !strict && line ~ /^[ \t]*idiomatic_version_file_enable_tools[ \t]*=/ { next }
        line ~ /(^|[^a-z0-9_])node/ { print FNR ": " $0; next }
        line ~ /(^|[^a-z0-9_])go(lang)?([^a-z0-9_]|$)/ {
            if (strict || (bare != "go=\"latest\"" && bare != "go=\047latest\047" &&
                bare != "go=\"" want "\"" && bare != "go=\047" want "\047")) print FNR ": " $0
        }
    ' "$1"
}

for file in mise.toml .mise.toml mise/config.toml .mise/config.toml .tool-versions \
    .devcontainer.json .devcontainer/devcontainer.json .devcontainer/*/devcontainer.json; do
    [[ -f "$file" ]] || continue
    case "$file" in
        *.toml) hint="the only allowed entry is go = \"latest\" or go = \"$go_version\"" ;;
        *) hint="$file must not name either" ;;
    esac
    while IFS= read -r violation; do
        fail "$file" "$file:$violation: Node comes from .nvmrc and Go from go.mod; $hint."
    done < <(mise_violations "$file")
done

if [[ "$status" -ne 0 ]]; then
    exit "$status"
fi
echo "Toolchain versions match: Go $go_version, Node $node_version."
