# 本脚本在 Windows 启动后台 Linux 应用容器和独立 Caddy 网关，
# 验证后端健康接口及 HTTPS 配置接口后保存进程记录，返回可访问地址。
# 运行数据和进程状态放在数据目录，私有值从进程环境或 .env 读取。
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

# 等待次数、轮询间隔、单请求超时和停止进程等待时间集中声明。
$ErrorActionPreference = 'Stop'
$readinessAttempts = 120
$readinessIntervalMilliseconds = 500
$requestTimeoutSeconds = 5
$processExitTimeoutMilliseconds = 10000
. (Join-Path $PSScriptRoot 'project-root.ps1')
$root = Get-LeafDocProjectRoot
# 私有配置按 UTF-8 逐行读取，跳过空行和井号开头行，只接受变量赋值格式。
# 配对引号只移除外层字符，不执行转义展开、变量替换或 shell 代码。
# 仅填充尚未定义的进程变量，已有环境值优先。
$environmentFile = Join-Path $PSScriptRoot '../.env'
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
# 参数默认值在 .env 读取前绑定，未显式传入的参数在此重新取得环境值。
if (-not $PSBoundParameters.ContainsKey('DataDirectory')) { $DataDirectory = $env:LEAFDOC_DATA_DIR }
if (-not $PSBoundParameters.ContainsKey('CertificatePath')) { $CertificatePath = $env:LEAFDOC_TLS_CERT }
if (-not $PSBoundParameters.ContainsKey('KeyPath')) { $KeyPath = $env:LEAFDOC_TLS_KEY }
if (-not $PSBoundParameters.ContainsKey('BasePath')) { $BasePath = $env:LEAFDOC_BASE_PATH }
# 监听端口以共享表为准，Port 只允许零值或共享前端端口，不用于任意自选端口。
$ports = Get-Content -LiteralPath (Join-Path $root 'backend/internal/core/serviceports/ports.json') -Raw -Encoding UTF8 | ConvertFrom-Json
if ($Port -ne 0 -and $Port -ne $ports.frontend) {
    throw 'The frontend port must match the shared fixed service allocation.'
}
$Port = [int]$ports.frontend
$backendPort = [int]$ports.backend
$adminPort = [int]$ports.gatewayAdmin
# 未显式提供证书时使用系统公共数据目录下的证书和私钥，不在此生成或安装信任。
$systemTlsDirectory = Join-Path ([Environment]::GetFolderPath('CommonApplicationData')) 'LeafDoc/tls'
if (-not $CertificatePath) { $CertificatePath = Join-Path $systemTlsDirectory 'server.crt' }
if (-not $KeyPath) { $KeyPath = Join-Path $systemTlsDirectory 'server.key' }
if (-not $DataDirectory) { $DataDirectory = Join-Path $root 'other/data' }
$DataDirectory = [IO.Path]::GetFullPath($DataDirectory)
$runtime = Join-Path $DataDirectory '.runtime'
$statePath = Join-Path $runtime 'processes.json'
$image = 'leafdoc:local'
$containerId = $null
# 基础路径去除边缘空白和斜杠，再保留一个开头斜杠，根路径表示为空。
# 查询符号、片段和任意空白被拒绝，其余路径字符不在这里进一步编码。
$BasePath = '/' + ([string]$BasePath).Trim().Trim('/')
if ($BasePath -eq '/') { $BasePath = '' }
if ($BasePath -match '[?#\s]') { throw 'The base path must be a URL path without query, fragment or whitespace.' }

# 停止保存的容器前核对项目标签，停止进程前同时核对执行路径和启动时间，
# 避免进程 ID 被复用后终止其他程序。容器删除不删除绑定的数据目录。
function Stop-SavedProcesses($state) {
    if ($state.containerId) {
        $existing = & docker ps -a --filter "id=$($state.containerId)" --format '{{.ID}}'
        if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect saved container.' }
        if ($existing) {
            $owner = & docker inspect --format '{{index .Config.Labels "leafdoc.source"}}' $state.containerId
            if ($LASTEXITCODE -ne 0 -or $owner -ne $root) { throw 'Saved container ownership does not match this project.' }
            & docker rm -f $state.containerId | Out-Null
            if ($LASTEXITCODE -ne 0) { throw 'Cannot stop saved container.' }
        }
    }
    foreach ($entry in $state.processes) {
        $process = Get-Process -Id $entry.id -ErrorAction SilentlyContinue
        # 保存时间统一转成 UTC 后比较，兼容 JSON 读取器返回字符串或 DateTime。
        if ($process -and $process.Path -eq $entry.path -and
            [Math]::Abs(($process.StartTime.ToUniversalTime() - ([DateTime]$entry.started).ToUniversalTime()).TotalSeconds) -lt 1) {
            Stop-Process -Id $process.Id -Force
            if (-not $process.WaitForExit($processExitTimeoutMilliseconds)) {
                throw 'A saved service did not stop within the allowed time.'
            }
        }
    }
}

# Stop 分支只处理保存的实例，检查进程退出后才移除状态文件，不继续启动流程。
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
# 可执行文件优先按当前命令解析，再检查进程与系统 Path 的合并目录。
# 路径内的环境变量在检查前展开，没有找到时抛错，不自动下载程序。
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
# Docker、Caddy 与证书都在停止旧实例之前检查，证书路径转成实际绝对路径。
$docker = Resolve-Executable 'docker.exe'
$caddy = Resolve-Executable 'caddy.exe'
foreach ($file in @($CertificatePath, $KeyPath)) {
    if (-not $file -or -not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw 'Set LEAFDOC_TLS_CERT and LEAFDOC_TLS_KEY to a trusted localhost certificate and its private key.'
    }
}
$CertificatePath = (Resolve-Path -LiteralPath $CertificatePath).Path
$KeyPath = (Resolve-Path -LiteralPath $KeyPath).Path
New-Item -ItemType Directory -Path $runtime -Force | Out-Null

# 先确认 Docker 为 Linux 模式，需要时完成镜像构建，再处理现有实例。
# 已有镜像且未要求 Build 时直接复用，不在这里判断镜像是否对应当前源码。
$dockerSystem = & $docker info --format '{{.OSType}}'
if ($LASTEXITCODE -ne 0 -or $dockerSystem -ne 'linux') { throw 'Docker must be running Linux containers.' }
$availableImage = & $docker image ls --quiet $image
if ($LASTEXITCODE -ne 0) { throw 'Cannot inspect the application image.' }
if ($Build -or -not $availableImage) {
    $node = Resolve-Executable 'node.exe'
    & $node (Join-Path $PSScriptRoot 'container.mjs') build
    if ($LASTEXITCODE -ne 0) { throw 'Linux application image build failed.' }
}
if (Test-Path -LiteralPath $statePath) {
    Stop-SavedProcesses (Get-Content -LiteralPath $statePath -Raw -Encoding UTF8 | ConvertFrom-Json)
    Remove-Item -LiteralPath $statePath
}
# 旧状态处理完后释放共享监听，指标启用时也释放指标端口。
$requiredServices = @('frontend', 'backend', 'gatewayAdmin')
if ($env:LEAFDOC_ENABLE_METRICS -in @('true', '1')) { $requiredServices += 'metrics' }
& (Join-Path $PSScriptRoot 'release-service-ports.ps1') -Services ($requiredServices -join ',')

# JWT 密钥只在文件缺失时生成，后续启动复用，避免每次启动改变签名密钥。
$secretPath = Join-Path $runtime 'jwt.key'
if (-not (Test-Path -LiteralPath $secretPath)) {
    [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(64)) |
        Set-Content -LiteralPath $secretPath -Encoding UTF8 -NoNewline
}
# 网关配置绑定本机环回地址，使用给定证书并代理后端，管理端口来自共享表。
# 下方 here-string 属于运行配置内容，由 Caddy 格式化后使用。
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
# 就绪探测复用 HTTP 客户端并禁用代理，单次请求受集中设置的超时限制。
# 未关闭证书验证，HTTPS 探测需要本机信任所用证书。
$handler = [Net.Http.HttpClientHandler]::new()
$handler.UseProxy = $false
$client = [Net.Http.HttpClient]::new($handler)
$client.Timeout = [TimeSpan]::FromSeconds($requestTimeoutSeconds)

# 每轮确认容器与已启动服务仍存活，再发送探测请求。任意成功状态码即视为就绪，
# 不在这里解析响应 JSON 或检查管理员登录。连接或请求超时继续下一轮，
# 其他异常直接向外传播。
function Wait-ServiceReady([string]$Url, [Diagnostics.Process[]]$Services) {
    for ($attempt = 0; $attempt -lt $readinessAttempts; $attempt++) {
        if ($containerId) {
            $running = & $docker inspect --format '{{.State.Running}}' $containerId
            if ($LASTEXITCODE -ne 0 -or $running -ne 'true') { throw 'Backend container exited. Inspect docker logs.' }
        }
        foreach ($service in $Services) {
            if ($service.HasExited) { throw 'A service exited. Inspect the runtime logs.' }
        }
        try {
            $response = $client.GetAsync($Url).GetAwaiter().GetResult()
            try {
                if ($response.IsSuccessStatusCode) { return }
            } finally { $response.Dispose() }
        } catch [Net.Http.HttpRequestException] {
            # 尚未监听或其他 HTTP 连接失败时继续重试。
        } catch [Threading.Tasks.TaskCanceledException] {
            # 单次请求达到客户端超时后继续下一轮，整体轮数仍有限。
        }
        Start-Sleep -Milliseconds $readinessIntervalMilliseconds
    }
    throw "Service readiness check failed for $Url. Inspect the runtime logs."
}

try {
    $env:LEAFDOC_JWT_SECRET = Get-Content -LiteralPath $secretPath -Raw -Encoding UTF8
    # 应用私有环境按变量名交给 Docker 读取，不把密码值展开到启动命令参数。
    # 容器名称由数据目录摘要生成，相同目录对应同一名称，实例带项目归属标签。
    $identity = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes($DataDirectory))).Substring(0, 12).ToLowerInvariant()
    $arguments = @('run', '-d', '--name', "leafdoc-$identity", '--restart', 'unless-stopped', '--label', "leafdoc.source=$root",
        '--mount', "type=bind,source=$DataDirectory,target=/app/data", '-p', "127.0.0.1:${backendPort}:${backendPort}",
        '--health-cmd', "wget -qO- http://127.0.0.1:$backendPort$BasePath/api/health", '--health-interval', '10s', '--health-timeout', '5s', '--health-retries', '6')
    foreach ($variable in Get-ChildItem Env: | Where-Object Name -like 'LEAFDOC_*') {
        if ($variable.Name -notin @('LEAFDOC_DATA_DIR', 'LEAFDOC_MYSQL_HOST')) { $arguments += @('--env', $variable.Name) }
    }
    # 宿主环回数据库地址转换成容器可访问的宿主入口，其余主机值沿用环境配置。
    $databaseHost = $env:LEAFDOC_MYSQL_HOST
    if ($databaseHost -in @('localhost', '127.0.0.1', '::1')) { $databaseHost = 'host.docker.internal' }
    $arguments += @('--env', "LEAFDOC_MYSQL_HOST=$databaseHost", '--env', 'LEAFDOC_DATA_DIR=/app/data')
    if ($env:LEAFDOC_ENABLE_METRICS -in @('true', '1')) { $arguments += @('-p', "127.0.0.1:$($ports.metrics):$($ports.metrics)") }
    $arguments += @($image, '--host', '0.0.0.0', '--port', "$backendPort", '--data-dir', '/app/data', '--enable-revision', '--snapshot', '--metrics-port', "$($ports.metrics)")
    if ($BasePath) { $arguments += @('--base-path', $BasePath) }
    # -d 让容器脱离当前终端，unless-stopped 使 Docker 按策略维持后台运行。
    $containerId = (& $docker @arguments).Trim()
    if ($LASTEXITCODE -ne 0 -or -not $containerId) { throw 'Backend container did not start.' }
    Wait-ServiceReady "http://127.0.0.1:$backendPort$BasePath/api/health" @()
    # 后端健康通过才启动隐藏网关进程，标准输出与错误输出分别保存到运行目录。
    $gateway = Start-Process -FilePath $caddy -ArgumentList @('run', '--config', "`"$gatewayConfig`"") -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput (Join-Path $runtime 'gateway.stdout.log') -RedirectStandardError (Join-Path $runtime 'gateway.stderr.log') -PassThru
    $processes += $gateway
    $url = "https://localhost:$Port$BasePath"
    Wait-ServiceReady "$url/api/config" $processes
    # HTTPS 配置接口通过后才写入可供停止入口核对的进程与容器记录。
    $state = @{ url = $url; dataDirectory = $DataDirectory; containerId = $containerId; image = $image; processes = @(
        foreach ($process in $processes) {
            @{ id = $process.Id; path = $process.Path; started = $process.StartTime.ToUniversalTime().ToString('o') }
        }
    ) }
    $state | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $statePath -Encoding UTF8
    # 就绪后把运行目录中指定名称的 Windows 二进制送入回收站，其他文件不参与处理。
    $obsoleteBinary = Join-Path $runtime 'leafdoc.exe'
    if (Test-Path -LiteralPath $obsoleteBinary -PathType Leaf) {
        Add-Type -AssemblyName Microsoft.VisualBasic.Core
        [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile($obsoleteBinary, 'OnlyErrorDialogs', 'SendToRecycleBin')
    }
    Write-Output $url
} catch {
    # try 范围内启动失败时删除已取得 ID 的容器并结束已登记网关，再把原异常抛出。
    if ($containerId) {
        & $docker rm -f $containerId | Out-Null
        if ($LASTEXITCODE -ne 0) { Write-Error 'Failed to clean up the unsuccessful backend container.' }
    }
    foreach ($process in $processes) {
        if (-not $process.HasExited) { $process.Kill(); $process.WaitForExit() }
    }
    throw
} finally {
    # 恢复原 JWT 环境值并释放本脚本的对象句柄，Dispose 不终止已经就绪的后台进程。
    $env:LEAFDOC_JWT_SECRET = $savedSecret
    foreach ($process in $processes) { $process.Dispose() }
    $client.Dispose()
    $handler.Dispose()
}
