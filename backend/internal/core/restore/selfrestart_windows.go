//go:build windows

// 本文件提供 Windows 的进程重启入口，使用创建子进程后退出的方式。
package restore

import (
	"os"
	"os/exec"
)

// SelfRestart 使用当前可执行文件和去掉程序名后的参数启动子进程，
// 继承环境与标准输入输出。Start 失败保留当前进程并返回错误，成功则立即退出。
// 此入口不等待子进程完成启动或健康检查，也不在退出前等待服务清理。
func SelfRestart() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	os.Exit(0)
	return nil // os.Exit 成功结束进程后不会执行到此行。
}
