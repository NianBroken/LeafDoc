# All PowerShell launchers share the source-root resolver used by Node and shell tools.
function Get-LeafDocProjectRoot {
    $resolved = & node (Join-Path $PSScriptRoot 'project-root.mjs')
    if ($LASTEXITCODE -ne 0 -or -not $resolved) { throw 'LeafDoc source root was not found.' }
    return $resolved.Trim()
}
