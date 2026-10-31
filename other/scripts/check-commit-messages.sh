#!/bin/bash
# 检查指定提交范围内非合并提交的标题是否符合项目约定格式。
# 输出供工作流使用时转义百分号、回车和换行，避免特殊字符被解释为命令。
#
# Usage: other/scripts/check-commit-messages.sh <base>..<head>
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <base>..<head>" >&2
    exit 2
fi
RANGE="$1"

TYPES='build|chore|ci|docs|feat|fix|perf|refactor|revert|style|test'
CONVENTIONAL="^(${TYPES})(\([A-Za-z0-9][A-Za-z0-9._/-]*\))?!?: [^ ]"
# 允许 GitHub 撤销按钮生成的 Revert 标题格式。
GITHUB_REVERT='^Revert ".+"$'

# 工作流命令对百分号、回车和换行有特殊处理，逐项编码后输出。
escape() {
    local s="${1//'%'/'%25'}"
    s="${s//$'\r'/'%0D'}"
    printf '%s' "${s//$'\n'/'%0A'}"
}

log="$(git log --no-merges --format='%h%x09%s' "$RANGE")"

failed=0
while IFS=$'\t' read -r sha subject; do
    [[ -z "$sha" ]] && continue
    if [[ "$subject" =~ $CONVENTIONAL || "$subject" =~ $GITHUB_REVERT ]]; then
        continue
    fi
    echo "::error title=Commit ${sha}::Not a Conventional Commit subject: $(escape "$subject")"
    failed=1
done <<< "$log"

if [[ "$failed" -ne 0 ]]; then
    echo "Use \"type(scope): description\" with type one of ${TYPES//|/, };" \
        "fixup!/squash!/WIP commits must be squashed first. See CONTRIBUTING.md#commit-style."
    exit 1
fi
echo "All commit subjects in ${RANGE} are Conventional Commits."
