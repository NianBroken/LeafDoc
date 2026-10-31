# PowerShell 入口共用 Node 的项目定位器，使用脚本所在目录而非调用方当前目录。
# 定位失败或没有输出时直接抛错，成功后返回去掉边缘空白的路径文本。
function Get-LeafDocProjectRoot {
    $resolved = & node (Join-Path $PSScriptRoot 'project-root.mjs')
    if ($LASTEXITCODE -ne 0 -or -not $resolved) { throw 'LeafDoc source root was not found.' }
    return $resolved.Trim()
}
