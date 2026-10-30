# Source-tree launchers delegate port ownership to the shared Go command.
[CmdletBinding()]
param([string]$Services='frontend,backend,gatewayAdmin')
$ErrorActionPreference='Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    & go.exe run ./cmd/service-ports @($Services.Split(','))
    if ($LASTEXITCODE -ne 0) { throw 'Could not release fixed service ports.' }
} finally { Pop-Location }
