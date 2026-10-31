// 本文件将 ZIP 条目解压到独立工作目录，检查路径边界并共用归档大小预算。
package importer

import (
	"archive/zip"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// 多处延迟关闭共用日志正文，关闭错误记入日志而不替换主流程返回的错误。
const logCloseFailed = "close failed"

// ZipExtractor 保存解压过程的日志器，实际归档读取和磁盘写入在方法调用内进行。
type ZipExtractor struct {
	log *slog.Logger
}

// NewZipExtractor 以默认日志器建立带组件字段的实例，不立即创建工作目录。
func NewZipExtractor() *ZipExtractor {
	return &ZipExtractor{
		log: slog.Default().With("component", "ZipExtractor"),
	}
}

// ExtractToTemp 使用系统临时目录作为工作目录父级，其余规则与 ExtractToDir 相同。
func (x *ZipExtractor) ExtractToTemp(zipPath string) (*ZipWorkspace, error) {
	return x.ExtractToDir(zipPath, "")
}

// ExtractToDir 使用常规 ZIP 限制，在 baseDir 下创建 import- 前缀的独立目录。
// baseDir 为空时由 os.MkdirTemp 选择系统临时目录，成功后的清理责任交给调用方。
func (x *ZipExtractor) ExtractToDir(zipPath string, baseDir string) (*ZipWorkspace, error) {
	return x.extractToDirWithLimits(zipPath, baseDir, shared.DefaultZipExtractionLimits)
}

// extractToDirWithLimits 按归档中的条目顺序解压，任一失败都会尝试清理整个工作目录。
// 输出文件使用普通文件写入，不恢复归档中的权限、时间和符号链接属性。
func (x *ZipExtractor) extractToDirWithLimits(zipPath string, baseDir string, limits shared.ExtractionLimits) (*ZipWorkspace, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			x.log.Error(logCloseFailed, "error", err)
		}
	}()

	parentDir := baseDir
	if parentDir == "" {
		parentDir = ""
	} else if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdirall: %w", err)
	}

	root, err := os.MkdirTemp(parentDir, "import-*")
	if err != nil {
		return nil, fmt.Errorf("mkdtemp: %w", err)
	}

	ws := &ZipWorkspace{Root: root}
	// 失败清理的错误独立记日志，返回值保留触发解压失败的原始原因。
	fail := func(e error) (*ZipWorkspace, error) {
		if err = ws.Cleanup(); err != nil {
			x.log.Error("cleanup failed", "error", err)
		}
		return nil, e
	}

	// 所有非目录条目共用一个预算，成功复制会扣减后续条目的可用字节数。
	budget := shared.NewSizeBudget(limits.MaxTotalBytes)
	for _, f := range r.File {
		// 条目名先去掉两端空白，空名称和目录项不单独创建输出。
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		if f.FileInfo().IsDir() {
			continue
		}

		destPath, err := safeJoin(ws.Root, name)
		if err != nil {
			return fail(fmt.Errorf("invalid zip entry %q: %w", f.Name, err))
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fail(fmt.Errorf("mkdir: %w", err))
		}

		// 每个文件使用独立函数作用域，使输入和输出句柄在进入下一条目前关闭。
		if err := func() error {
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("open zip entry: %w", err)
			}
			defer func() {
				if err := rc.Close(); err != nil {
					x.log.Error(logCloseFailed, "error", err)
				}
			}()

			// 同一路径重复出现会截断并覆盖前项输出，已消耗的累计预算不会因此退回。
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer func() {
				if err := out.Close(); err != nil {
					x.log.Error(logCloseFailed, "error", err)
				}
			}()

			if err := shared.CopyWithBudget(out, rc, f.CompressedSize64, limits, budget); err != nil {
				return fmt.Errorf("write file: %w", err)
			}

			return nil
		}(); err != nil {
			return fail(err)
		}
	}

	return ws, nil
}

// safeJoin 按当前平台规则规范条目路径，拒绝绝对路径和清理后越出根目录的路径。
// 前缀比较带目录分隔符，避免相邻名称目录被误认作根目录的子路径。
// 这里仅处理路径文本，不解析文件系统中的符号链接，也不创建任何文件。
func safeJoin(baseDir, zipEntryName string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(zipEntryName))
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute path not allowed")
	}
	dest := filepath.Join(baseDir, clean)

	baseClean := filepath.Clean(baseDir) + string(filepath.Separator)
	destClean := filepath.Clean(dest)

	if !strings.HasPrefix(destClean+string(filepath.Separator), baseClean) {
		return "", fmt.Errorf("path traversal detected: %q", zipEntryName)
	}
	return destClean, nil
}
