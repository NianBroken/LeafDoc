#!/usr/bin/env bash
# 本函数从脚本位置查找项目标记，不要求服务器安装 Node。
# 输出物理绝对路径，调用目录、中文目录和空格不影响定位。
leafdoc_project_root() {
    local script_path directory
    script_path="${BASH_SOURCE[0]}"
    while [ -L "$script_path" ]; do
        local target
        target="$(readlink -- "$script_path")"
        case "$target" in /*) script_path="$target";; *) script_path="$(dirname -- "$script_path")/$target";; esac
    done
    case "$script_path" in /*) ;; *) script_path="$(pwd -P)/$script_path";; esac
    directory="$(cd -- "$(dirname -- "$script_path")/../../.." && pwd -P)"
    test -f "$directory/backend/go.mod" && test -f "$directory/frontend/package.json" || return 1
    printf '%s\n' "$directory"
}
