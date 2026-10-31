# 由工具链检查脚本和环境诊断脚本从项目根目录加载，集中读取语言版本来源。

# 从 go.mod 的 toolchain 行提取完整 Go 版本号，无符合格式的行时输出空值。
toolchain_go_version() {
    local file="${1:-go.mod}"
    sed -n -E 's/^toolchain go([0-9]+\.[0-9]+\.[0-9]+).*/\1/p' "$file" 2>/dev/null
}

# 从 .nvmrc 首个非空行取 Node 主版本，剥去可选 v 前缀和后续小数部分。
# 文件缺失或首个非空值不含数字时命令不输出版本号。
toolchain_node_major() {
    local file="${1:-.nvmrc}"
    awk 'NF { sub(/^v/, "", $1); sub(/[^0-9].*/, "", $1); print $1; exit }' "$file" 2>/dev/null
}
