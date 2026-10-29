# Reclaims only explicitly named fixed listeners from their direct process owners.
[CmdletBinding()]
param([string]$Services = 'frontend,backend,gatewayAdmin')

$ErrorActionPreference = 'Stop'
$processExitTimeoutMilliseconds = 10000
$root = Split-Path $PSScriptRoot -Parent
$allocation = Get-Content -LiteralPath (Join-Path $root 'internal/core/serviceports/ports.json') -Raw -Encoding UTF8 | ConvertFrom-Json
$requiredPorts = @(
    foreach ($service in $Services.Split(',')) {
        $name = $service.Trim()
        if ($name -notin $allocation.PSObject.Properties.Name) { throw "Unknown fixed service: $name" }
        [int]$allocation.$name
    }
)

function Get-ListenerOwners {
    $owners = @(
        Get-NetTCPConnection -State Listen | Where-Object LocalPort -In $requiredPorts | Select-Object -ExpandProperty OwningProcess
        Get-NetUDPEndpoint | Where-Object LocalPort -In $requiredPorts | Select-Object -ExpandProperty OwningProcess
    )
    return @($owners | Sort-Object -Unique)
}

foreach ($ownerId in (Get-ListenerOwners)) {
    if ($ownerId -eq $PID -or $ownerId -le 4) {
        throw "The fixed port is held by a process that cannot be terminated: $ownerId"
    }
    $owner = Get-Process -Id $ownerId -ErrorAction SilentlyContinue
    if ($owner) {
        Write-Output "Stopping process $ownerId [$($owner.ProcessName)] occupying a fixed service port."
        try {
            Stop-Process -Id $ownerId -Force
            if (-not $owner.WaitForExit($processExitTimeoutMilliseconds)) {
                throw "The process occupying a fixed service port did not exit: $ownerId"
            }
        } finally { $owner.Dispose() }
    }
}
if (@(Get-ListenerOwners).Count -ne 0) {
    throw 'A fixed service port is still occupied. No alternate port will be used.'
}
