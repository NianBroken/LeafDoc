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
	return ReleaseAt("", services...)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if runtime.GOOS != "linux" {
		return fmt.Errorf("port reclamation unsupported on %s", runtime.GOOS)
	}
	command := exec.CommandContext(ctx, "bash", "-c", linuxRelease, "leafdoc-release", strings.Join(ports, " "))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "LEAFDOC_RELEASE_HOST="+host)
	if err := command.Run(); err != nil {
		return fmt.Errorf("release fixed ports: %w", err)
	}
	return nil
}

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
