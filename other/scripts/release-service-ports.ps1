# Resolves shared listener allocations and releases Windows owners without compilation.
[CmdletBinding()]
param([string]$Services='frontend,backend,gatewayAdmin')
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'project-root.ps1')
$projectRoot = Get-LeafDocProjectRoot
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
function Owners {
 $all = $env:LEAFDOC_RELEASE_HOST -in @('', '0.0.0.0', '::')
 $addresses = @($env:LEAFDOC_RELEASE_HOST, '0.0.0.0', '::')
 if ($env:LEAFDOC_RELEASE_HOST -eq '127.0.0.1') { $addresses += '::1' }
 @(Get-NetTCPConnection -State Listen | Where-Object { $_.LocalPort -in $ports -and ($all -or $_.LocalAddress -in $addresses) } | Select-Object -ExpandProperty OwningProcess
 Get-NetUDPEndpoint | Where-Object { $_.LocalPort -in $ports -and ($all -or $_.LocalAddress -in $addresses) } | Select-Object -ExpandProperty OwningProcess) | Sort-Object -Unique
}
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
$deadline=[DateTime]::UtcNow.AddSeconds(10)
while (@(Owners).Count -gt 0) {
 if ([DateTime]::UtcNow -ge $deadline) {throw 'Fixed listeners remain occupied'}
 Start-Sleep -Milliseconds 100
}
