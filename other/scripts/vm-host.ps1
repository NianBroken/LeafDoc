# Windows 本地验证宿主管理器，按持久配置监护专属于 LeafDoc 的 QEMU 虚拟机。
# Ubuntu 内部服务独立启动，本入口不替代 Ubuntu 的 systemd 或 Docker。
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Configuration,
    [switch]$Once
)
$ErrorActionPreference = 'Stop'
$pollSeconds = 10
$configurationPath = [IO.Path]::GetFullPath($Configuration)
$runtimeDirectory = Split-Path -Parent $configurationPath
$machines = [IO.File]::ReadAllText($configurationPath, [Text.Encoding]::UTF8) | ConvertFrom-Json
do {
    foreach ($machine in $machines) {
        if (-not $machine.enabled) { continue }
        $directory = [IO.Path]::GetFullPath((Join-Path $runtimeDirectory $machine.name))
        if (-not $directory.StartsWith($runtimeDirectory + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) {
            throw 'Virtual machine path escapes the configured runtime directory.'
        }
        $disk = Join-Path $directory 'disk.qcow2'
        $owned = @(Get-CimInstance Win32_Process -Filter "Name='qemu-system-x86_64.exe'" | Where-Object {
            $_.CommandLine -and $_.CommandLine.Contains($disk)
        })
        if ($owned.Count -gt 1) { throw "Multiple processes own $disk" }
        if ($owned.Count -eq 1) { continue }
        $arguments = @('-machine', 'q35', '-accel', $machine.accelerator, '-cpu', 'max', '-smp', $machine.cpus,
            '-m', $machine.memory, '-drive', "file=$disk,if=virtio,format=qcow2",
            '-drive', "file=$(Join-Path $directory 'seed.iso'),media=cdrom,readonly=on,format=raw",
            '-netdev', "user,id=net0,hostfwd=tcp:127.0.0.1:$($machine.ssh)-:22,hostfwd=tcp:0.0.0.0:$($machine.https)-:$($machine.guestHttps)",
            '-device', 'virtio-net-pci,netdev=net0', '-display', 'none', '-serial', "file:$(Join-Path $directory 'serial.log')")
        # 路径中的空格通过逐项引号保留，参数值来自本地机器配置，不接受命令文本。
        $quoted = @($arguments | ForEach-Object { '"' + ([string]$_).Replace('"', '\"') + '"' })
        $process = Start-Process -FilePath $machine.executable -ArgumentList $quoted -WorkingDirectory $directory -WindowStyle Hidden -RedirectStandardOutput (Join-Path $directory 'qemu.stdout.log') -RedirectStandardError (Join-Path $directory 'qemu.stderr.log') -PassThru
        try {
            [IO.File]::WriteAllText((Join-Path $directory 'pid.txt'), [string]$process.Id, [Text.UTF8Encoding]::new($false))
        } finally { $process.Dispose() }
    }
    if ($Once) { break }
    Start-Sleep -Seconds $pollSeconds
} while ($true)
