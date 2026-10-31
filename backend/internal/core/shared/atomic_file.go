// 本文件提供字节和数据流两种临时文件写入方式，在写入完成后替换目标文件。
// 临时文件放在目标目录，调用方负责保证该目录已经存在。
// 提交流程同步临时文件的数据，不对父目录执行额外同步；持久化保证受文件系统规则约束。
package shared

import (
	"fmt"
	"io"
	"log/slog"

	"os"
	"path/filepath"
	"strings"
)

// 关闭失败的日志文本由多条写入失败路径共用，原始操作错误仍作为返回值保留。
const errFailedToCloseTempFile = "failed to close temp file"

// atomicWriteDir 将两种斜线统一为当前平台可识别的路径，再提取父目录。
// 处理过程不创建目录，也不验证传入路径对应的文件是否存在。
func atomicWriteDir(filename string) string {
	normalized := strings.ReplaceAll(filename, `\`, `/`)
	return filepath.Dir(filepath.FromSlash(normalized))
}

// WriteFileAtomic 将完整字节写入目标目录中的临时文件，同步并关闭后替换目标。
// perm 非零时应用指定权限，零值保留 CreateTemp 设置的权限。
// 创建、权限、写入、同步、关闭和替换失败均返回对应错误，不提前删除目标文件。
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir := atomicWriteDir(filename)

	// 随机临时名称隔离同目录中的并行写入；同一目标最终保留哪次结果由替换顺序决定。
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmpFile.Name()
	// 函数返回时尝试移除临时路径。替换成功后该路径已经不存在，移除结果不参与返回值。
	defer func() {
		_ = os.Remove(tmpName)
	}()

	// 字节写入方式在写入前设置权限，区别于下方数据流方式的写入后设置。
	if perm != 0 {
		if err := tmpFile.Chmod(perm); err != nil {
			chmodErr := fmt.Errorf("chmod temp file: %w", err)
			if closeErr := tmpFile.Close(); closeErr != nil {
				slog.Default().Error(errFailedToCloseTempFile, "operation", "chmod", "error", closeErr)
			}
			return chmodErr
		}
	}

	// 完整字节切片交给文件句柄写入，只有写入无错才进入同步与提交阶段。
	if _, err := tmpFile.Write(data); err != nil {
		writeErr := fmt.Errorf("write temp file: %w", err)
		if closeErr := tmpFile.Close(); closeErr != nil {
			slog.Default().Error(errFailedToCloseTempFile, "operation", "write", "error", closeErr)
		}
		return writeErr
	}

	// 同步失败会终止替换，关闭失败单独记日志，保留同步错误作为本次返回原因。
	if err := tmpFile.Sync(); err != nil {
		syncErr := fmt.Errorf("sync temp file: %w", err)
		if closeErr := tmpFile.Close(); closeErr != nil {
			slog.Default().Error(errFailedToCloseTempFile, "operation", "sync", "error", closeErr)
		}
		return syncErr
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// 关闭句柄后调用平台实现，Windows 和其他平台使用各自的文件替换接口。
	if err := atomicReplace(tmpName, filename); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

// WriteStreamAtomic 从 src 读取内容，写入临时文件并在成功后替换目标路径。
// CopyWithLimit 最多写入 maxBytes 加一字节来识别超限，超限时不替换目标。
// 函数不关闭 src，源数据的生命周期由调用方管理。
func WriteStreamAtomic(targetPath string, src io.Reader, maxBytes int64, perm os.FileMode) error {
	dir := atomicWriteDir(targetPath)

	out, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := out.Name()

	// ok 表示目标替换已成功，out 为 nil 表示句柄已经显式关闭。
	// 未成功时尝试删除临时文件；延迟关闭失败会记日志并结束清理，不再执行删除。
	ok := false
	defer func() {
		if out == nil {
			if !ok {
				_ = os.Remove(tmp)
			}
			return
		}

		if err := out.Close(); err != nil {
			slog.Default().Error("Failed to close temp file", "file", tmp, "error", err)
			return
		}
		if !ok {
			_ = os.Remove(tmp)
		}
	}()

	if err := CopyWithLimit(out, src, maxBytes); err != nil {
		return err
	}

	// 数据同步是提交前的必要步骤，失败直接返回，不继续权限设置和目标替换。
	if err := out.Sync(); err != nil {
		return err
	}

	// 数据写完后才应用非零权限，写入期间使用 CreateTemp 的初始访问权限。
	if perm != 0 {
		if err := out.Chmod(perm); err != nil {
			return fmt.Errorf("chmod temp file: %w", err)
		}
	}

	// 此处关闭失败仍保留 out，返回后的延迟清理会再次尝试关闭句柄。
	if err := out.Close(); err != nil {
		return err
	}
	// 清空句柄引用，使延迟清理不会重复关闭已经成功关闭的文件。
	out = nil

	if err := atomicReplace(tmp, targetPath); err != nil {
		return err
	}

	ok = true
	return nil
}
