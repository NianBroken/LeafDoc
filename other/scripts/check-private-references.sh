#!/bin/bash
# 本脚本搜索私有笔记标记和指定形式的个人主目录路径，命中即报告文件与行号。
# Git 工作区使用受跟踪文件搜索，容器源码快照使用无索引搜索，均排除列出的样本和锁文件。
set -euo pipefail

. "$(dirname "$0")/lib/project-root.sh"
ROOT="$(leafdoc_project_root)"
cd "$ROOT"

# 私有笔记标记忽略大小写匹配，不把这些标记扩展成其他内容推断。
PRIVATE_NOTES='pleaf|ai-gen-infos'
# 个人主目录匹配 /home/ 或 /Users/ 后的 ASCII 名称，可不带尾部斜杠。
# 目录前缀按表达式原样比较，Windows 路径不在此规则范围内，供路径测试样本使用。
HOME_PATHS='/(home|Users)/[A-Za-z_][A-Za-z0-9._-]*'

EXCLUDED=(
    ':(exclude)other/scripts/check-private-references.sh'
    # glob 路径规则中的开头 **/ 也覆盖零层目录，包含顶层测试数据和样本目录。
    ':(exclude,glob)**/testdata/**'
    ':(exclude,glob)**/fixtures/**'
    ':(exclude,glob)**/package-lock.json'
)

# 搜索退出零表示命中，一表示未命中，大于一表示搜索失败，
# 搜索失败必须单独终止，不能被当作没有违规内容。
search() {
    local status=0
    # 没有 Git 工作区时直接搜索源码树，忽略规则仍通过 --exclude-standard 生效。
    if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        git grep -n -I -E "$@" -- . "${EXCLUDED[@]}" || status=$?
    else
        git grep --no-index --exclude-standard -n -I -E "$@" -- . "${EXCLUDED[@]}" || status=$?
    fi
    if [[ "$status" -gt 1 ]]; then
        echo "::error::git grep failed (exit $status); could not search source at $ROOT" >&2
        exit 2
    fi
}

# 两组搜索结果汇总后按文件、行号输出诊断，不改写命中的原文件。
hits="$(
    search -i -e "$PRIVATE_NOTES"
    search -e "$HOME_PATHS"
)"

if [[ -n "$hits" ]]; then
    while IFS= read -r hit; do
        file="${hit%%:*}"
        rest="${hit#*:}"
        line="${rest%%:*}"
        echo "::error file=${file},line=${line}::Private reference: ${rest#*:}"
    done <<< "$hits"
    echo "Reference public GitHub issues instead of private machine paths."
    exit 1
fi
echo "No private references found."
