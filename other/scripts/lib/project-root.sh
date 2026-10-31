#!/usr/bin/env bash
# 本函数通过自身脚本位置调用共享 Node 定位器，与 Windows 和 Node 入口共用根目录规则。
# 定义函数时不输出内容，调用时直接返回 Node 的标准输出和退出状态。
leafdoc_project_root() {
    node "$(dirname "${BASH_SOURCE[0]}")/../project-root.mjs"
}
