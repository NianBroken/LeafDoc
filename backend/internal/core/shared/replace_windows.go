// 文件名的 _windows 后缀使本实现仅参与 Windows 构建。
package shared

import "golang.org/x/sys/windows"

// atomicReplace 将路径转换为 Windows 宽字符字符串后调用 MoveFileEx。
// 替换标记允许覆盖已有目标，写穿标记要求系统按该接口的规则完成移动写入。
// 路径包含无法转换的内容时直接返回错误，不先删除目标，也不尝试复制回退。
func atomicReplace(src, dst string) error {
	// 两个路径分别转换，任一转换失败都在调用文件移动接口之前返回。
	source, err := windows.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	destination, err := windows.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	// 目标替换和写穿请求由同一个系统调用提交，系统错误原样交给调用方处理。
	return windows.MoveFileEx(source, destination, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
