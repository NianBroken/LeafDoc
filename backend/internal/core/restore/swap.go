// 本文件把备份解压到数据目录内的暂存区，核对归档条目、摘要和数据库结构，
// 再以目录改名实现四类资源的替换与回滚。数据库事务由 offline.go 编排。
package restore

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// backupMeta 对应归档内部的版本元数据，字段与快照写入格式一致。
// 解码后不在这里校验 ID、创建时间或版本是否匹配当前实例。
type backupMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Version   string    `json:"version"`
}

// 归档必须具有数据库载荷、版本元数据和校验清单三个入口。
var requiredZipEntries = []string{"backup-meta.json", "database.json", "checksums.json"}

// extractAndValidate 使用默认解压限额完成验证，成功返回的暂存目录由调用方删除。
func extractAndValidate(zipPath, dataDir string) (stagingDir string, meta backupMeta, err error) {
	return extractAndValidateWithLimits(zipPath, dataDir, shared.DefaultZipExtractionLimits)
}

// extractAndValidateWithLimits 先检查条目名称重复和必要入口，再逐项按共享预算解压。
// 摘要与数据库格式通过后，确保四类资源暂存目录都存在；缺失资源会成为空目录，
// 因而正常恢复会用空目录替换相应现有资源，不表示保留备份中缺少的资源。
func extractAndValidateWithLimits(zipPath, dataDir string, limits shared.ExtractionLimits) (stagingDir string, meta backupMeta, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", backupMeta{}, fmt.Errorf("failed to open snapshot zip: %w", err)
	}
	defer func() { _ = r.Close() }()

	// 名称按宿主路径清理、正斜杠和小写形式去重，真正解压仍使用条目的原始名称。
	present := map[string]bool{}
	for _, f := range r.File {
		name := strings.ToLower(filepath.ToSlash(filepath.Clean(f.Name)))
		if present[name] {
			return "", backupMeta{}, fmt.Errorf("duplicate backup entry: %s", f.Name)
		}
		present[name] = true
	}
	for _, want := range requiredZipEntries {
		if !present[want] {
			return "", backupMeta{}, fmt.Errorf("unsupported backup format: required MySQL entry %q is missing", want)
		}
	}

	stagingDir, err = os.MkdirTemp(dataDir, ".leafdoc-restore-*")
	if err != nil {
		return "", backupMeta{}, fmt.Errorf("failed to create staging directory: %w", err)
	}

	// 全部文件共用总字节预算，单文件及压缩比例由每条目的复制工具继续检查。
	budget := shared.NewSizeBudget(limits.MaxTotalBytes)
	for _, f := range r.File {
		if err := extractZipEntry(f, stagingDir, limits, budget); err != nil {
			_ = os.RemoveAll(stagingDir)
			return "", backupMeta{}, fmt.Errorf("failed to extract %s: %w", f.Name, err)
		}
	}

	metaBytes, err := os.ReadFile(filepath.Join(stagingDir, "backup-meta.json"))
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", backupMeta{}, fmt.Errorf("failed to read backup-meta.json: %w", err)
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", backupMeta{}, fmt.Errorf("failed to parse backup-meta.json: %w", err)
	}

	if err := validateArchive(stagingDir); err != nil {
		return "", backupMeta{}, errors.Join(err, os.RemoveAll(stagingDir))
	}
	// 这些空目录在摘要检查之后创建，不要求空目录本身列在摘要清单中。
	for _, name := range swapNames {
		if err := os.MkdirAll(filepath.Join(stagingDir, name), 0o755); err != nil {
			return "", backupMeta{}, errors.Join(err, os.RemoveAll(stagingDir))
		}
	}

	return stagingDir, meta, nil
}

// extractZipEntry 使用宿主系统路径规则检查条目，拒绝点路径、双点开头、
// 绝对路径和相对计算后逃出目标目录的路径。目录直接创建，其他条目按普通文件写入，
// 不按照归档权限创建符号链接；文件和输入流都延迟关闭，关闭错误不返回。
func extractZipEntry(f *zip.File, destDir string, limits shared.ExtractionLimits, budget *shared.SizeBudget) error {
	cleanName := filepath.Clean(f.Name)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
		return fmt.Errorf("unsafe entry path %q", f.Name)
	}
	destPath := filepath.Join(destDir, cleanName)
	rel, err := filepath.Rel(destDir, destPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("unsafe entry path %q", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if err := shared.CopyWithBudget(out, rc, f.CompressedSize64, limits, budget); err != nil {
		return err
	}
	return nil
}

// 目录交换仅覆盖四类文件资源，业务表通过数据库导入事务恢复。
var swapNames = []string{"assets", "branding", "avatars", "revision-assets"}

// swapItem 记录一个资源的当前路径、暂存路径、旁置旧副本与两步移动进度。
type swapItem struct {
	name       string
	livePath   string
	stagedPath string
	preRestore string
	// 当前路径成功移走或 Stat 未报告存在后设为真，即使第二步移入失败，
	// 回滚仍需要检查旧副本。此状态与替换资源是否已就位分开保存。
	movedAside bool
	// 暂存资源成功移入当前路径后设为真，此时 movedAside 已经为真。
	swapped bool
}

// swapper 逐项先移走当前目录，再移入暂存目录，不要求改名直接覆盖已有目标。
// 两次文件系统操作不是整体原子操作，进度字段用于失败时尝试恢复。
type swapper struct {
	items []*swapItem
	// 测试可替换改名函数精确制造失败，回滚入口仍直接使用 os.Rename。
	rename func(string, string) error
}

// newSwapper 生成四组路径及带纳秒时间文本的旧副本名称，不执行移动或预留名称。
// 历史附件映射到 .leafdoc/blobs/assets，其余资源位于数据目录同名子目录。
// 正常验证流程会建立所有暂存目录，因此缺失资源以空目录参与交换。
func newSwapper(dataDir, stagingDir string) *swapper {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	items := make([]*swapItem, 0, len(swapNames))
	for _, name := range swapNames {
		live := filepath.Join(dataDir, name)
		if name == "revision-assets" {
			live = filepath.Join(dataDir, ".leafdoc", "blobs", "assets")
		}
		items = append(items, &swapItem{
			name:       name,
			livePath:   live,
			stagedPath: filepath.Join(stagingDir, name),
			preRestore: live + ".pre-restore-" + stamp,
		})
	}
	return &swapper{items: items}
}

// SwapAll 按资源顺序执行两步改名，首个明确失败直接返回，不自行回滚。
// 暂存路径任意 Stat 错误都会跳过该项，当前路径任意 Stat 错误都会跳过移走步骤；
// 成功移入后记录 swapped，由调用方在数据库提交失败时选择回滚。
func (sw *swapper) SwapAll() error {
	rename := sw.rename
	if rename == nil {
		rename = os.Rename
	}
	for _, item := range sw.items {
		if _, err := os.Stat(item.stagedPath); err != nil {
			continue // 暂存路径 Stat 失败时保留当前资源，不区分具体错误原因。
		}
		if err := os.MkdirAll(filepath.Dir(item.livePath), 0o755); err != nil {
			return fmt.Errorf("prepare restored resource directory: %w", err)
		}

		if _, err := os.Stat(item.livePath); err == nil {
			if err := rename(item.livePath, item.preRestore); err != nil {
				return fmt.Errorf("failed to move aside %s: %w", item.name, err)
			}
		}
		item.movedAside = true

		if err := rename(item.stagedPath, item.livePath); err != nil {
			return fmt.Errorf("failed to move in restored %s: %w", item.name, err)
		}
		item.swapped = true
	}
	return nil
}

// RollbackAll 按资源顺序处理 movedAside 为真的项，删除当前路径后尝试把旧副本移回。
// 某项失败仍继续其他项，并合并错误；旧副本 Stat 的任何错误都按无需移回处理。
// 只有成功恢复或判定无副本时清空进度，失败项保留标记供调用方识别未完成恢复。
func (sw *swapper) RollbackAll() error {
	var errs []error
	for _, item := range sw.items {
		// 第二步移动失败也必须回滚，因此检查 movedAside 而不是只检查 swapped。
		if !item.movedAside {
			continue
		}
		// 当前路径不存在时 RemoveAll 可成功，已移入资源则在此删除。
		if err := os.RemoveAll(item.livePath); err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to remove restored content: %w", item.name, err))
			continue
		}
		if _, statErr := os.Stat(item.preRestore); statErr != nil {
			// 本分支不区分不存在与其他 Stat 错误，直接清空进度并继续。
			item.movedAside = false
			item.swapped = false
			continue
		}
		if err := os.Rename(item.preRestore, item.livePath); err != nil {
			errs = append(errs, fmt.Errorf("%s: failed to restore pre-restore copy: %w", item.name, err))
			continue
		}
		item.movedAside = false
		item.swapped = false
	}
	return errors.Join(errs...)
}

// CommitAll 删除已成功移入项的旧副本，失败记录警告并继续，不返回错误。
// restoreStaged 在数据库提交成功后调用，在线设置与运行态重载尚在其后执行。
func (sw *swapper) CommitAll() {
	for _, item := range sw.items {
		if !item.swapped {
			continue
		}
		if err := os.RemoveAll(item.preRestore); err != nil {
			slog.Default().Warn("restore: failed to clean up pre-restore backup copy", "item", item.name, "path", item.preRestore, "error", err)
		}
	}
}

// validateArchive 要求数据库和元数据各有非空摘要，并检查所有非目录文件。
// 除清单自身外，每个文件都须有匹配 SHA-256，检查后删除对应键，
// 剩余键表示清单声明了缺失资源。通过后再验证数据库载荷格式。
// 摘要证明清单与载荷一致，不构成归档来源的签名认证。
func validateArchive(dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, "checksums.json"))
	if err != nil {
		return err
	}
	checksums := map[string]string{}
	if err := json.Unmarshal(raw, &checksums); err != nil {
		return err
	}
	for _, required := range []string{"database.json", "backup-meta.json"} {
		if checksums[required] == "" {
			return fmt.Errorf("missing required checksum")
		}
	}
	err = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if name == "checksums.json" {
			return nil
		}
		expected, ok := checksums[name]
		if !ok {
			return fmt.Errorf("unverified backup entry: %s", name)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(bytes)
		if hex.EncodeToString(sum[:]) != expected {
			return fmt.Errorf("backup checksum mismatch: %s", name)
		}
		delete(checksums, name)
		return nil
	})
	if err != nil {
		return err
	}
	if len(checksums) != 0 {
		return fmt.Errorf("backup resource is missing")
	}
	raw, err = os.ReadFile(filepath.Join(dir, "database.json"))
	if err != nil {
		return err
	}
	_, err = database.ValidateDump(raw)
	return err
}
