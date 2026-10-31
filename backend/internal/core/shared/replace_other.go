//go:build !windows

// 本文件仅参与非 Windows 平台的构建，为原子文件写入提供最终替换步骤。
// 文件顶部的 go:build 是构建工具读取的平台条件，必须保留其原文和独立行格式。
package shared

import "os"

// atomicReplace 将临时源文件重命名到目标位置，直接返回操作系统的执行结果。
// 是否允许替换现有文件及跨文件系统移动由 os.Rename 的平台规则决定，
// 本函数不复制文件、不创建目录，也不在失败后删除源文件。
func atomicReplace(src, dst string) error { return os.Rename(src, dst) }
