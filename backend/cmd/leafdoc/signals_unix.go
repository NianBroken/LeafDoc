//go:build !windows

// 本文件只参与非 Windows 构建，为启动入口登记两种运行时重载信号。
package main

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyReloadSignals 将 SIGUSR1 与 SIGHUP 转送到调用方通道。
// 调用方负责消费信号，并在生命周期结束时使用 signal.Stop 撤销登记。
func notifyReloadSignals(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGHUP)
}
