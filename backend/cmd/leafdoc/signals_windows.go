//go:build windows

// 本文件提供 Windows 的同名信号登记入口，使公共启动流程无需平台分支。
package main

import "os"

// notifyReloadSignals 不登记信号，传入通道保持未接入状态。
func notifyReloadSignals(_ chan os.Signal) {
	// Windows 没有 SIGUSR1 与 SIGHUP，此平台不通过这两种信号触发重载。
}
