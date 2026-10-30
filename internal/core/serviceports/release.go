package serviceports

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Release identifies owning services and frees explicitly named fixed listeners.
func Release(services ...string) error {
	host := ""
	if runtime.GOOS == "windows" {
		host = "127.0.0.1"
	}
	return ReleaseAt(host, services...)
}

// ReleaseAt isolates concrete bind addresses while also detecting wildcard conflicts.
func ReleaseAt(host string, services ...string) error {
	if len(services) == 0 {
		return fmt.Errorf("at least one fixed service name is required")
	}
	var ports []string
	for _, service := range services {
		found := false
		for _, name := range Names() {
			if name == service {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown fixed service: %s", service)
		}
		ports = append(ports, Port(service))
	}
	var command *exec.Cmd
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "windows":
		command = exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", windowsRelease+" '"+strings.Join(ports, ",")+"'")
	case "linux":
		command = exec.CommandContext(ctx, "bash", "-c", linuxRelease, "leafdoc-release", strings.Join(ports, " "))
	default:
		return fmt.Errorf("port reclamation unsupported on %s", runtime.GOOS)
	}
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "LEAFDOC_RELEASE_HOST="+host)
	if err := command.Run(); err != nil {
		return fmt.Errorf("release fixed ports: %w", err)
	}
	return nil
}

// windowsRelease stops service ownership before killing residual process trees.
const windowsRelease = `& {
param([string]$allocation)
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
} `

// linuxRelease inspects both protocols and resolves systemd ownership before termination.
const linuxRelease = `set -euo pipefail
ports="$1"
command -v lsof >/dev/null
owners() {
 for port in $ports; do
  for protocol in TCP UDP; do
   args=(-nP -t "-i$protocol:$port")
   if [ "$protocol" = TCP ]; then args+=(-sTCP:LISTEN); fi
   result=0
   lsof "${args[@]}" || result=$?
   if [ "$result" -gt 1 ]; then return "$result"; fi
  done
 done | sort -un
}
kill_tree() {
 local owner="$1" child
 while read -r child; do [ -z "$child" ] || kill_tree "$child"; done < <(ps -o pid= --ppid "$owner")
 if [ -d "/proc/$owner" ]; then kill -KILL "$owner"; fi
}
owner_list=$(owners)
for owner in $owner_list; do
 if [ "$owner" -le 1 ] || [ "$owner" = "$$" ] || [ "$owner" = "$PPID" ]; then echo "Protected listener owner: $owner" >&2; exit 1; fi
 echo "Releasing fixed listener: PID=$owner executable=$(readlink /proc/$owner/exe)"
 unit=$(awk -F/ '{for(i=1;i<=NF;i++) if ($i ~ /[.]service$/) print $i}' "/proc/$owner/cgroup" | head -n 1)
 if [ -n "$unit" ] && [ "$unit" != "${SYSTEMD_UNIT:-}" ]; then echo "Stopping service $unit"; systemctl stop "$unit"; fi
 kill_tree "$owner"
done
for attempt in {1..100}; do
 remaining=$(owners)
 if [ -z "$remaining" ]; then exit 0; fi
 sleep 0.1
done
echo 'Fixed listeners remain occupied' >&2
exit 1
`
