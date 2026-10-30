# Builds and runs an embedded frontend with a detached HTTPS gateway on Windows.
# Credentials come from the environment; runtime state stays in the ignored data directory.
[CmdletBinding()]
param(
    [switch]$Build,
    [switch]$Stop,
    [string]$DataDirectory = $env:LEAFDOC_DATA_DIR,
    [string]$CertificatePath = $env:LEAFDOC_TLS_CERT,
    [string]$KeyPath = $env:LEAFDOC_TLS_KEY,
    [string]$BasePath = $env:LEAFDOC_BASE_PATH,
    [int]$Port = 0
)

$ErrorActionPreference = 'Stop'
$readinessAttempts = 120
$readinessIntervalMilliseconds = 500
$requestTimeoutSeconds = 5
$processExitTimeoutMilliseconds = 10000
$root = Split-Path $PSScriptRoot -Parent
# Private UTF-8 values fill unset process variables without evaluating shell code.
$environmentFile = Join-Path $root '.env'
if (Test-Path -LiteralPath $environmentFile -PathType Leaf) {
    foreach ($line in [IO.File]::ReadAllLines($environmentFile, [Text.Encoding]::UTF8)) {
        $entry = $line.Trim()
        if (-not $entry -or $entry.StartsWith('#')) { continue }
        if ($entry -notmatch '^([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$') {
            throw 'Invalid environment assignment in .env.'
        }
        $name = $Matches[1]
        $value = $Matches[2].Trim()
        if ($value.Length -ge 2 -and (($value.StartsWith('"') -and $value.EndsWith('"')) -or
            ($value.StartsWith("'") -and $value.EndsWith("'")))) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        if ($null -eq [Environment]::GetEnvironmentVariable($name, 'Process')) {
            [Environment]::SetEnvironmentVariable($name, $value, 'Process')
        }
    }
}
if (-not $PSBoundParameters.ContainsKey('DataDirectory')) { $DataDirectory = $env:LEAFDOC_DATA_DIR }
if (-not $PSBoundParameters.ContainsKey('CertificatePath')) { $CertificatePath = $env:LEAFDOC_TLS_CERT }
if (-not $PSBoundParameters.ContainsKey('KeyPath')) { $KeyPath = $env:LEAFDOC_TLS_KEY }
if (-not $PSBoundParameters.ContainsKey('BasePath')) { $BasePath = $env:LEAFDOC_BASE_PATH }
$ports = Get-Content -LiteralPath (Join-Path $root 'internal/core/serviceports/ports.json') -Raw -Encoding UTF8 | ConvertFrom-Json
if ($Port -ne 0 -and $Port -ne $ports.frontend) {
    throw 'The frontend port must match the shared fixed service allocation.'
}
$Port = [int]$ports.frontend
$backendPort = [int]$ports.backend
$adminPort = [int]$ports.gatewayAdmin
# Machine-wide certificate files are discovered without requiring registry writes.
$systemTlsDirectory = Join-Path ([Environment]::GetFolderPath('CommonApplicationData')) 'LeafDoc/tls'
if (-not $CertificatePath) { $CertificatePath = Join-Path $systemTlsDirectory 'server.crt' }
if (-not $KeyPath) { $KeyPath = Join-Path $systemTlsDirectory 'server.key' }
if (-not $DataDirectory) { $DataDirectory = Join-Path $root 'data' }
$DataDirectory = [IO.Path]::GetFullPath($DataDirectory)
$runtime = Join-Path $DataDirectory '.runtime'
$statePath = Join-Path $runtime 'processes.json'
$binary = Join-Path $runtime 'leafdoc.exe'
$BasePath = '/' + ([string]$BasePath).Trim().Trim('/')
if ($BasePath -eq '/') { $BasePath = '' }
if ($BasePath -match '[?#\s]') { throw 'The base path must be a URL path without query, fragment or whitespace.' }

# Match both the executable and creation time before stopping a saved process.
function Stop-SavedProcesses($state) {
    foreach ($entry in $state.processes) {
        $process = Get-Process -Id $entry.id -ErrorAction SilentlyContinue
        # JSON readers can return timestamps as either strings or DateTime values.
        if ($process -and $process.Path -eq $entry.path -and
            [Math]::Abs(($process.StartTime.ToUniversalTime() - ([DateTime]$entry.started).ToUniversalTime()).TotalSeconds) -lt 1) {
            Stop-Process -Id $process.Id -Force
            if (-not $process.WaitForExit($processExitTimeoutMilliseconds)) {
                throw 'A saved service did not stop within the allowed time.'
            }
        }
    }
}

if ($Stop) {
    if (Test-Path -LiteralPath $statePath) {
        Stop-SavedProcesses (Get-Content -LiteralPath $statePath -Raw -Encoding UTF8 | ConvertFrom-Json)
        $remaining = Get-Content -LiteralPath $statePath -Raw -Encoding UTF8 | ConvertFrom-Json
        foreach ($entry in $remaining.processes) {
            $process = Get-Process -Id $entry.id -ErrorAction SilentlyContinue
            if ($process -and $process.Path -eq $entry.path) {
                throw 'A saved service is still running. Its process record was retained.'
            }
        }
        Remove-Item -LiteralPath $statePath
    }
    return
}
# Resolve executables from the current process and refreshed machine environment.
function Resolve-Executable([string]$Name) {
    $command = Get-Command $Name -ErrorAction SilentlyContinue
    if ($command) { return $command.Source }
    $paths = @($env:Path, [Environment]::GetEnvironmentVariable('Path', 'Machine'))
    foreach ($directory in (($paths -join ';') -split ';' | Select-Object -Unique)) {
        if (-not $directory) { continue }
        $candidate = Join-Path ([Environment]::ExpandEnvironmentVariables($directory)) $Name
        if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
    }
    throw "Required executable was not found: $Name"
}
$go = Resolve-Executable 'go.exe'
$caddy = Resolve-Executable 'caddy.exe'
foreach ($file in @($CertificatePath, $KeyPath)) {
    if (-not $file -or -not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw 'Set LEAFDOC_TLS_CERT and LEAFDOC_TLS_KEY to a trusted localhost certificate and its private key.'
    }
}
$CertificatePath = (Resolve-Path -LiteralPath $CertificatePath).Path
$KeyPath = (Resolve-Path -LiteralPath $KeyPath).Path
New-Item -ItemType Directory -Path $runtime -Force | Out-Null

# The fixed listeners are reclaimed from saved services and direct TCP/UDP owners.
if (Test-Path -LiteralPath $statePath) {
    Stop-SavedProcesses (Get-Content -LiteralPath $statePath -Raw -Encoding UTF8 | ConvertFrom-Json)
    Remove-Item -LiteralPath $statePath
}
$requiredServices = @('frontend', 'backend', 'gatewayAdmin')
if ($env:LEAFDOC_ENABLE_METRICS -in @('true', '1')) {
    $requiredServices += 'metrics'
}
& (Join-Path $PSScriptRoot 'release-service-ports.ps1') -Services ($requiredServices -join ',')

if ($Build -or -not (Test-Path -LiteralPath $binary)) {
    Push-Location (Join-Path $root 'ui')
    try {
        & npm.cmd run build
        if ($LASTEXITCODE -ne 0) { throw 'Frontend build failed.' }
    } finally { Pop-Location }
    $destination = Join-Path $root 'internal/http/dist'
    # Only generated frontend output inside the project is replaced.
    $expected = [IO.Path]::GetFullPath((Join-Path $root 'internal/http/dist'))
    if ([IO.Path]::GetFullPath($destination) -ne $expected) { throw 'Invalid frontend output path.' }
    Get-ChildItem -LiteralPath $destination -Force |
        Where-Object Name -ne '.gitkeep' | Remove-Item -Recurse -Force
    Copy-Item -Path (Join-Path $root 'ui/dist/*') -Destination $destination -Recurse
    Push-Location $root
    try {
        $node = Resolve-Executable 'node.exe'
        $version = (& $node (Join-Path $root 'scripts/resolve-version.mjs')).Trim()
        if ($LASTEXITCODE -ne 0 -or -not $version) { throw 'Release version resolution failed.' }
        & $go build -ldflags "-X main.Version=$version -X github.com/NianBroken/LeafDoc/internal/http.EmbedFrontend=true" -o $binary ./cmd/leafdoc
        if ($LASTEXITCODE -ne 0) { throw 'Backend build failed.' }
    } finally { Pop-Location }
}

$secretPath = Join-Path $runtime 'jwt.key'
if (-not (Test-Path -LiteralPath $secretPath)) {
    [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(64)) |
        Set-Content -LiteralPath $secretPath -Encoding UTF8 -NoNewline
}
# The gateway's administration listener follows the shared service allocation.
$gatewayConfig = Join-Path $runtime 'Caddyfile'
@"
{
    auto_https off
    admin 127.0.0.1:$adminPort
}
https://localhost:$Port {
    bind 127.0.0.1
    tls "$($CertificatePath.Replace('\', '/'))" "$($KeyPath.Replace('\', '/'))"
    reverse_proxy 127.0.0.1:$backendPort
}
"@ | Set-Content -LiteralPath $gatewayConfig -Encoding UTF8
& $caddy fmt --overwrite $gatewayConfig
if ($LASTEXITCODE -ne 0) { throw 'Gateway configuration formatting failed.' }

$savedSecret = $env:LEAFDOC_JWT_SECRET
$processes = @()
$handler = [Net.Http.HttpClientHandler]::new()
$handler.UseProxy = $false
$client = [Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds($requestTimeoutSeconds)

# Probe the backend before exposing it through the gateway, then verify HTTPS.
function Wait-ServiceReady([string]$Url, [Diagnostics.Process[]]$Services) {
    for ($attempt = 0; $attempt -lt $readinessAttempts; $attempt++) {
        foreach ($service in $Services) {
            if ($service.HasExited) { throw 'A service exited. Inspect the runtime logs.' }
        }
        try {
            $response = $client.GetAsync($Url).GetAwaiter().GetResult()
            try {
                if ($response.IsSuccessStatusCode) { return }
            } finally { $response.Dispose() }
        } catch [Net.Http.HttpRequestException] {
            # Connection refusal is expected until the listener is ready.
        } catch [Threading.Tasks.TaskCanceledException] {
            # Individual requests are bounded by the client timeout.
        }
        Start-Sleep -Milliseconds $readinessIntervalMilliseconds
    }
    throw "Service readiness check failed for $Url. Inspect the runtime logs."
}

try {
    $env:LEAFDOC_JWT_SECRET = Get-Content -LiteralPath $secretPath -Raw -Encoding UTF8
    $backendArguments = @('--host', '127.0.0.1', '--port', "$backendPort", '--data-dir', "`"$DataDirectory`"", '--enable-revision', '--snapshot', '--metrics-port', "$($ports.metrics)")
    if ($BasePath) { $backendArguments += @('--base-path', $BasePath) }
    $backend = Start-Process -FilePath $binary -ArgumentList $backendArguments -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput (Join-Path $runtime 'backend.stdout.log') -RedirectStandardError (Join-Path $runtime 'backend.stderr.log') -PassThru
    $processes += $backend
    Wait-ServiceReady "http://127.0.0.1:$backendPort$BasePath/api/health" $processes
    $gateway = Start-Process -FilePath $caddy -ArgumentList @('run', '--config', "`"$gatewayConfig`"") -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput (Join-Path $runtime 'gateway.stdout.log') -RedirectStandardError (Join-Path $runtime 'gateway.stderr.log') -PassThru
    $processes += $gateway
    $url = "https://localhost:$Port$BasePath"
    Wait-ServiceReady "$url/api/config" $processes
    $state = @{ url = $url; dataDirectory = $DataDirectory; processes = @(
        foreach ($process in $processes) {
            @{ id = $process.Id; path = $process.Path; started = $process.StartTime.ToUniversalTime().ToString('o') }
        }
    ) }
    $state | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $statePath -Encoding UTF8
    Write-Output $url
} catch {
    foreach ($process in $processes) {
        if (-not $process.HasExited) { $process.Kill(); $process.WaitForExit() }
    }
    throw
} finally {
    $env:LEAFDOC_JWT_SECRET = $savedSecret
    foreach ($process in $processes) { $process.Dispose() }
    $client.Dispose()
    $handler.Dispose()
}
