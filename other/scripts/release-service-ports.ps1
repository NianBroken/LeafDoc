# 本脚本从共享端口表解析服务名，停止 Windows 上占用所选监听地址和端口的进程。
# 默认选择前端、后端和网关管理端口，不编译 Go，也不启动业务服务。
[CmdletBinding()]
param([string]$Services='frontend,backend,gatewayAdmin')
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'project-root.ps1')
$projectRoot = Get-LeafDocProjectRoot
# 配置按 UTF-8 读取，每个服务必须能在共享表中找到，未知名称直接拒绝。
$allocationMap = Get-Content -LiteralPath (Join-Path $projectRoot 'backend/internal/core/serviceports/ports.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$selected = @()
foreach ($service in $Services.Split(',')) {
    $property = $allocationMap.PSObject.Properties[$service]
    if ($null -eq $property) { throw "Unknown fixed service: $service" }
    $selected += [string]$property.Value
}
$allocation = $selected -join ','

$ErrorActionPreference='Stop'
$ports=@($allocation.Split(',') | ForEach-Object { [int]$_ })
# Owners 合并 TCP 监听与 UDP 端点的进程 ID 并去重。未限制主机或指定通配主机时
# 匹配所有本地地址；指定环回 IPv4 时也检查 IPv6 环回及通配监听。
function Owners {
 $all = $env:LEAFDOC_RELEASE_HOST -in @('', '0.0.0.0', '::')
 $addresses = @($env:LEAFDOC_RELEASE_HOST, '0.0.0.0', '::')
 if ($env:LEAFDOC_RELEASE_HOST -eq '127.0.0.1') { $addresses += '::1' }
 @(Get-NetTCPConnection -State Listen | Where-Object { $_.LocalPort -in $ports -and ($all -or $_.LocalAddress -in $addresses) } | Select-Object -ExpandProperty OwningProcess
 Get-NetUDPEndpoint | Where-Object { $_.LocalPort -in $ports -and ($all -or $_.LocalAddress -in $addresses) } | Select-Object -ExpandProperty OwningProcess) | Sort-Object -Unique
}
# 先保护低编号系统进程和当前 PowerShell，随后停止关联 Windows 服务，
# 仍存在的进程通过 taskkill 连同进程树强制结束。已经退出的进程直接跳过。
foreach ($ownerId in @(Owners)) {
 if ($ownerId -le 4 -or $ownerId -eq $PID) { throw "Protected listener owner: $ownerId" }
 $process=Get-CimInstance Win32_Process -Filter "ProcessId=$ownerId"
 if (-not $process) {continue}
 Write-Output "Releasing fixed listener: PID=$ownerId executable=$($process.ExecutablePath)"
 foreach ($service in @(Get-CimInstance Win32_Service -Filter "ProcessId=$ownerId")) {
  Write-Output "Stopping service $($service.Name)"
  Stop-Service -Name $service.Name -Force
 }
 if (Get-Process -Id $ownerId -ErrorAction SilentlyContinue) {
  & taskkill.exe /PID $ownerId /T /F
  if ($LASTEXITCODE -ne 0) { throw "Cannot stop listener process $ownerId" }
 }
}
# 结束命令返回后继续检查端点，最多等待十秒，仍占用则明确失败。
$deadline=[DateTime]::UtcNow.AddSeconds(10)
while (@(Owners).Count -gt 0) {
 if ([DateTime]::UtcNow -ge $deadline) {throw 'Fixed listeners remain occupied'}
 Start-Sleep -Milliseconds 100
}
