# Builds and runs an embedded frontend with a detached HTTPS gateway on Windows.
# Credentials come from the environment; runtime state stays in the ignored data directory.
[CmdletBinding()]
param(
    [switch]$Build,
    [switch]$Stop,
    [string]$DataDirectory = $env:LEAFWIKI_DATA_DIR,
    [string]$CertificatePath = $env:LEAFWIKI_TLS_CERT,
    [string]$KeyPath = $env:LEAFWIKI_TLS_KEY,
    [int]$Port = 0
)

$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
if (-not $DataDirectory) { $DataDirectory = Join-Path $root 'data' }
$DataDirectory = [IO.Path]::GetFullPath($DataDirectory)
$runtime = Join-Path $DataDirectory '.runtime'
$statePath = Join-Path $runtime 'processes.json'
$binary = Join-Path $runtime 'leafwiki.exe'

# Match both the executable and creation time before stopping a saved process.
function Stop-SavedProcesses($state) {
    foreach ($entry in $state.processes) {
        $process = Get-Process -Id $entry.id -ErrorAction SilentlyContinue
        if ($process -and $process.Path -eq $entry.path -and
            $process.StartTime.ToUniversalTime().ToString('o') -eq $entry.started) {
            Stop-Process -Id $process.Id
            $process.WaitForExit()
        }
    }
}

if ($Stop) {
    if (Test-Path -LiteralPath $statePath) {
        Stop-SavedProcesses (Get-Content -LiteralPath $statePath -Raw -Encoding UTF8 | ConvertFrom-Json)
        Remove-Item -LiteralPath $statePath
    }
    return
}
if (Test-Path -LiteralPath $statePath) {
    throw 'A saved runtime exists. Stop it with this script and -Stop before starting again.'
}
$go = Get-Command go.exe -ErrorAction Stop
$caddy = Get-Command caddy.exe -ErrorAction Stop
foreach ($file in @($CertificatePath, $KeyPath)) {
    if (-not $file -or -not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw 'Set LEAFWIKI_TLS_CERT and LEAFWIKI_TLS_KEY to a trusted localhost certificate and its private key.'
    }
}
$CertificatePath = (Resolve-Path -LiteralPath $CertificatePath).Path
$KeyPath = (Resolve-Path -LiteralPath $KeyPath).Path
New-Item -ItemType Directory -Path $runtime -Force | Out-Null

# Bind to port zero to discover available ports, without assuming a local service layout.
function Get-FreePort {
    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
    try {
        $listener.Start()
        return $listener.LocalEndpoint.Port
    } finally { $listener.Stop() }
}
if ($Port -eq 0) { $Port = Get-FreePort }
$backendPort = Get-FreePort
while ($backendPort -eq $Port) { $backendPort = Get-FreePort }
$probe = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $Port)
try { $probe.Start() } finally { $probe.Stop() }

if ($Build -or -not (Test-Path -LiteralPath $binary)) {
    Push-Location (Join-Path $root 'ui/leafwiki-ui')
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
    Copy-Item -Path (Join-Path $root 'ui/leafwiki-ui/dist/*') -Destination $destination -Recurse
    Push-Location $root
    try {
        & $go.Source build -ldflags '-X github.com/perber/wiki/internal/http.EmbedFrontend=true -X github.com/perber/wiki/internal/http.Environment=production' -o $binary ./cmd/leafwiki
        if ($LASTEXITCODE -ne 0) { throw 'Backend build failed.' }
    } finally { Pop-Location }
}

$secretPath = Join-Path $runtime 'jwt.key'
if (-not (Test-Path -LiteralPath $secretPath)) {
    [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(64)) |
        Set-Content -LiteralPath $secretPath -Encoding UTF8 -NoNewline
}
$gatewayConfig = Join-Path $runtime 'Caddyfile'
@"
{
    auto_https off
}
https://localhost:$Port {
    bind 127.0.0.1
    tls "$($CertificatePath.Replace('\', '/'))" "$($KeyPath.Replace('\', '/'))"
    reverse_proxy 127.0.0.1:$backendPort
}
"@ | Set-Content -LiteralPath $gatewayConfig -Encoding UTF8
& $caddy.Source fmt --overwrite $gatewayConfig
if ($LASTEXITCODE -ne 0) { throw 'Gateway configuration formatting failed.' }

$savedSecret = $env:LEAFWIKI_JWT_SECRET
$processes = @()
try {
    $env:LEAFWIKI_JWT_SECRET = Get-Content -LiteralPath $secretPath -Raw -Encoding UTF8
    $backend = Start-Process -FilePath $binary -ArgumentList @('--host', '127.0.0.1', '--port', "$backendPort", '--data-dir', "`"$DataDirectory`"", '--enable-revision') -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput (Join-Path $runtime 'backend.stdout.log') -RedirectStandardError (Join-Path $runtime 'backend.stderr.log') -PassThru
    $processes += $backend
    $gateway = Start-Process -FilePath $caddy.Source -ArgumentList @('run', '--config', "`"$gatewayConfig`"") -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput (Join-Path $runtime 'gateway.stdout.log') -RedirectStandardError (Join-Path $runtime 'gateway.stderr.log') -PassThru
    $processes += $gateway
    $url = "https://localhost:$Port"
    $ready = $false
    $handler = [Net.Http.HttpClientHandler]::new()
    $handler.UseProxy = $false
    $client = [Net.Http.HttpClient]::new($handler)
    $client.Timeout = [TimeSpan]::FromSeconds(5)
    try {
        for ($attempt = 0; $attempt -lt 120; $attempt++) {
            if ($backend.HasExited -or $gateway.HasExited) { throw 'A service exited. Inspect the runtime logs.' }
            try {
                $response = $client.GetAsync("$url/api/config").GetAwaiter().GetResult()
                if ($response.IsSuccessStatusCode) { $ready = $true; break }
            } catch {
                # Readiness retries end with an explicit failure and retain service diagnostics.
            }
            Start-Sleep -Milliseconds 500
        }
    } finally {
        $client.Dispose()
        $handler.Dispose()
    }
    if (-not $ready) { throw 'HTTPS readiness check failed. Inspect the runtime logs.' }
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
    $env:LEAFWIKI_JWT_SECRET = $savedSecret
    foreach ($process in $processes) { $process.Dispose() }
}
