// 本文件通过 Linux 进程工具释放固定端口，供服务启动前回收监听资源。
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

// Release 使用空主机参数释放指定服务名称对应的固定端口。
// 具体服务校验、平台限制和进程终止由 ReleaseAt 执行。
func Release(services ...string) error {
	return ReleaseAt("", services...)
}

// ReleaseAt 校验服务名称后，在 Linux 上调用 Bash 脚本终止端口占用进程。
// host 通过 LEAFDOC_RELEASE_HOST 传入子进程，但当前脚本未读取该变量，
// 因此实际检查按端口和协议执行，不按绑定地址隔离进程。
// 整个子进程最长运行三十秒，非 Linux 平台直接返回不支持的错误。
func ReleaseAt(host string, services ...string) error {
	if len(services) == 0 {
		return fmt.Errorf("at least one fixed service name is required")
	}
	// 先验证所有名称，再启动外部进程，避免未知名称造成部分端口已经被释放。
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
	// 超时上下文约束 Bash 命令的等待时间，cancel 在函数结束时释放相关计时资源。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if runtime.GOOS != "linux" {
		return fmt.Errorf("port reclamation unsupported on %s", runtime.GOOS)
	}
	// 固定脚本通过 bash -c 执行，服务端口作为位置参数传递，不拼接到脚本正文。
	command := exec.CommandContext(ctx, "bash", "-c", linuxRelease, "leafdoc-release", strings.Join(ports, " "))
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	command.Env = append(os.Environ(), "LEAFDOC_RELEASE_HOST="+host)
	if err := command.Run(); err != nil {
		return fmt.Errorf("release fixed ports: %w", err)
	}
	return nil
}

// linuxRelease 对每个端口查询 TCP 监听和 UDP 占用，合并并去重进程号。
// lsof 返回 1 表示没有匹配项，更大的退出码视为查询失败。
// 进程号不大于 1、当前脚本和直接父进程均受到保护，遇到这些占用立即失败。
// 发现不同于 SYSTEMD_UNIT 的服务归属时先停止该服务，再递归强制终止进程树。
// 处理后每隔零点一秒复查，最多一百轮，仍有占用就以非零状态退出。
// 脚本本身保存在运行时字符串中，文字或命令变更都会影响实际执行内容。
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
