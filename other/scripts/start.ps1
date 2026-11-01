# Windows 后台启动入口。Compose 管理双协议网关与后端，数据与凭据留在运行目录。
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
$requestTimeoutSeconds = 5
. (Join-Path $PSScriptRoot 'project-root.ps1')
$root = Get-LeafDocProjectRoot
$ports = Get-Content (Join-Path $root 'backend/internal/core/serviceports/ports.json') -Raw -Encoding UTF8 | ConvertFrom-Json
if ($Port -ne 0 -and $Port -ne $ports.frontend) { throw 'Frontend port must match the shared allocation.' }
if (!$DataDirectory) { $DataDirectory = Join-Path $root 'other/data' }
$DataDirectory = [IO.Path]::GetFullPath($DataDirectory)
$runtime = Join-Path $DataDirectory '.runtime'
New-Item -ItemType Directory -Path $runtime -Force | Out-Null
$composePath = Join-Path $runtime 'compose.yaml'
$statePath = Join-Path $runtime 'processes.json'
$environmentPath = Join-Path $runtime 'application.env'
$identity = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes($DataDirectory))).Substring(0,12).ToLowerInvariant()
$project = "leafdoc-$identity"
# 原生参数数组保留中文和空格路径，任何非零退出均阻止后续启动。
function Invoke-Container([string[]]$Arguments) {
    & docker @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Docker operation failed: $($Arguments[0])" }
}
if ($Stop) {
    if (Test-Path $composePath) { Invoke-Container -Arguments @('compose','-p',$project,'-f',$composePath,'down') }
    return
}
# 私有运行配置优先保留，进程环境中明确设置的值覆盖文件内容。
$values = @{}
foreach ($file in @((Join-Path $root 'other/.env'),$environmentPath)) {
    if (Test-Path $file) {
        foreach ($line in [IO.File]::ReadAllLines($file,[Text.Encoding]::UTF8)) {
            if (!$line.Trim() -or $line.TrimStart().StartsWith('#')) { continue }
            if ($line -notmatch '^([A-Za-z_][A-Za-z0-9_]*)=(.*)$') { throw 'Invalid environment assignment.' }
            $values[$Matches[1]] = $Matches[2].Trim().Trim('"').Trim("'")
        }
    }
}
foreach ($entry in Get-ChildItem Env: | Where-Object Name -like 'LEAFDOC_*') { $values[$entry.Name] = $entry.Value }
if (!$PSBoundParameters.ContainsKey('BasePath')) { $BasePath = $values['LEAFDOC_BASE_PATH'] }
$BasePath = '/' + ([string]$BasePath).Trim().Trim('/')
if ($BasePath -eq '/') { $BasePath = '' }
if ($BasePath -match '[?#\s]') { throw 'Invalid base path.' }
if (!$CertificatePath -and !$KeyPath) {
    $tls = & (Join-Path $PSScriptRoot 'ensure-tls.ps1')
    $CertificatePath = Join-Path $tls 'server.crt'
    $KeyPath = Join-Path $tls 'server.key'
}
foreach ($file in @($CertificatePath,$KeyPath)) { if (!$file -or !(Test-Path -LiteralPath $file -PathType Leaf)) { throw 'Both TLS certificate and key are required.' } }
$CertificatePath = (Resolve-Path -LiteralPath $CertificatePath).Path
$KeyPath = (Resolve-Path -LiteralPath $KeyPath).Path
if ($Build -or !(docker image ls -q leafdoc:local)) {
    Invoke-Container -Arguments @('build','-f',(Join-Path $root 'other/Dockerfile'),'--build-arg','APP_NAME=LeafDoc','--build-arg',("APP_VERSION=" + (Get-Content (Join-Path $root 'other/project.json') -Raw -Encoding UTF8 | ConvertFrom-Json).version),'-t','leafdoc:local',$root)
}
# 迁移旧入口保存的实例，核对归属与启动时间后停止，不删除数据卷。
if (Test-Path $statePath) {
    $state = Get-Content $statePath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ($state.containerId) {
        $existing = @(docker ps -aq --no-trunc) -contains $state.containerId
        if ($existing) {
            $owner = docker inspect --format '{{index .Config.Labels "leafdoc.source"}}' $state.containerId
            if ($owner -ne $root) { throw 'Saved container ownership mismatch.' }
            Invoke-Container -Arguments @('rm','-f',$state.containerId)
        }
    }
    foreach ($entry in $state.processes) {
        $process = Get-Process -Id $entry.id -ErrorAction SilentlyContinue
        if ($process -and $process.Path -eq $entry.path -and [Math]::Abs(($process.StartTime.ToUniversalTime() - ([DateTime]$entry.started).ToUniversalTime()).TotalSeconds) -lt 1) { Stop-Process -Id $process.Id -Force }
    }
}
if (Test-Path $composePath) { Invoke-Container -Arguments @('compose','-p',$project,'-f',$composePath,'down') }
& (Join-Path $PSScriptRoot 'release-service-ports.ps1') -Services 'frontend,backend'
if (!$values['LEAFDOC_JWT_SECRET']) { $values['LEAFDOC_JWT_SECRET'] = [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(64)) }
$values['LEAFDOC_DATA_DIR'] = '/app/data'
$values['LEAFDOC_HOST'] = '0.0.0.0'
$values['LEAFDOC_BASE_PATH'] = $BasePath
$values['LEAFDOC_TLS_CERT'] = '/run/leafdoc-tls/server.crt'
$values['LEAFDOC_TLS_KEY'] = '/run/leafdoc-tls/server.key'
if (!$values['LEAFDOC_MYSQL_HOST'] -and !(Test-Path (Join-Path $DataDirectory 'installation.json'))) {
    $values['LEAFDOC_WEB_SETUP'] = 'true'
    $tokenPath = Join-Path $DataDirectory 'setup-token'
    if (!(Test-Path $tokenPath)) {
        [IO.File]::WriteAllText($tokenPath,[Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)),[Text.UTF8Encoding]::new($false))
    }
}
if ($values['LEAFDOC_MYSQL_HOST'] -in @('localhost','127.0.0.1','::1')) { $values['LEAFDOC_MYSQL_HOST'] = 'host.docker.internal' }
[IO.File]::WriteAllLines($environmentPath,@($values.Keys | Sort-Object | ForEach-Object { "$_=$($values[$_])" }),[Text.UTF8Encoding]::new($false))
# JSON 是合法 YAML。通过结构序列化生成配置，避免路径和私密值被 shell 展开。
$mounts = @(
    @{ type='bind'; source=$CertificatePath; target='/run/leafdoc-tls/server.crt'; read_only=$true },
    @{ type='bind'; source=$KeyPath; target='/run/leafdoc-tls/server.key'; read_only=$true }
)
$healthPath = "$BasePath/api/health"
$services = @{
    app = @{
        image='leafdoc:local'; restart='unless-stopped'; env_file=@($environmentPath)
        labels=@{ 'leafdoc.source'=$root }; command=@('--enable-revision','--snapshot')
        ports=@("$($ports.backend):$($ports.backend)")
        volumes=@(@{type='bind';source=$DataDirectory;target='/app/data'}) + $mounts
        healthcheck=@{test=@('CMD','wget','-qO-',"http://127.0.0.1:$($ports.backend)$healthPath");interval='10s';timeout='5s';retries=6}
        stop_grace_period='45s'
    }
    gateway = @{
        image='leafdoc:local'; restart='unless-stopped'; command=@('gateway')
        labels=@{ 'leafdoc.source'=$root }; depends_on=@{ app=@{condition='service_healthy'} }
        environment=@{ LEAFDOC_BACKEND_URL="http://app:$($ports.backend)";LEAFDOC_TLS_CERT='/run/leafdoc-tls/server.crt';LEAFDOC_TLS_KEY='/run/leafdoc-tls/server.key';LEAFDOC_GATEWAY_HOST='0.0.0.0' }
        ports=@("$($ports.frontend):$($ports.frontend)");volumes=$mounts
        healthcheck=@{test=@('CMD','wget','-qO-',"http://127.0.0.1:$($ports.frontend)$healthPath");interval='10s';timeout='5s';retries=6}
    }
}
@{services=$services} | ConvertTo-Json -Depth 12 | Set-Content $composePath -Encoding UTF8
try {
Invoke-Container -Arguments @('compose','-p',$project,'-f',$composePath,'up','-d','--wait','--wait-timeout','180')
$handler = [Net.Http.HttpClientHandler]::new()
$handler.UseProxy=$false
$handler.AllowAutoRedirect=$false
$client=[Net.Http.HttpClient]::new($handler)
$client.Timeout=[TimeSpan]::FromSeconds($requestTimeoutSeconds)
try {
    foreach ($servicePort in @($ports.frontend,$ports.backend)) {
        foreach ($protocol in @('http','https')) {
            $url="${protocol}://localhost:$servicePort$healthPath"
            $response=$client.GetAsync($url).GetAwaiter().GetResult()
            try { if ([int]$response.StatusCode -ne 200) { throw "Readiness failed: $url" } } finally { $response.Dispose() }
        }
    }
} finally { $client.Dispose();$handler.Dispose() }
@{project=$project;compose=$composePath;dataDirectory=$DataDirectory;urls=@("http://localhost:$($ports.frontend)$BasePath","https://localhost:$($ports.frontend)$BasePath")} | ConvertTo-Json | Set-Content $statePath -Encoding UTF8
Write-Output "http://localhost:$($ports.frontend)$BasePath"
Write-Output "https://localhost:$($ports.frontend)$BasePath"
} catch {
    # 保存真实失败日志后停止未就绪实例，持久数据保留在绑定目录。
    & docker compose -p $project -f $composePath logs --no-color | Set-Content (Join-Path $runtime 'startup-failure.log') -Encoding UTF8
    & docker compose -p $project -f $composePath down
    throw
}
