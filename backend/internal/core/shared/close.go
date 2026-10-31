// 本文件统一处理无法通过返回值继续向上传递的关闭错误。
package shared

import (
	"log/slog"
)

// LogClose 调用传入的关闭函数，失败时通过默认 slog 日志记录错误。
// 它不返回错误、不重试关闭，适合放在 defer 中释放行集等资源，例如：
//
//	defer shared.LogClose(rows.Close, "could not close rows")
//
// closer 必须是可调用函数；此处不捕获 panic，也不检查资源是否已经关闭。
func LogClose(closer func() error, msg string) {
	if err := closer(); err != nil {
		slog.Default().Error(msg, "error", err)
	}
}
