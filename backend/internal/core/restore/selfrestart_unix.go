//go:build !windows

// 本文件为非 Windows 构建提供进程映像替换入口。
package restore

import (
	"os"
	"syscall"
)

// execFn 保存系统执行入口，测试替换它来捕获参数，避免实际替换测试进程。
var execFn = syscall.Exec

// SelfRestart 取得当前可执行路径，连同完整启动参数和环境交给系统替换进程映像。
// 替换成功沿用进程标识并重新执行启动流程，失败则返回系统错误。
// 此入口不验证磁盘业务数据是否完整，也不保证重启后的应用启动一定成功。
func SelfRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return execFn(exe, os.Args, os.Environ())
}
