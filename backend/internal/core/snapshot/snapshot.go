// 本文件把数据库导出与附件、品牌、头像及历史附件组成 ZIP，
// 在归档内保存校验摘要和版本元数据，并在归档旁写入供列表使用的 JSON 摘要。
package snapshot

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// Config 提供备份输出目录、业务资源路径、版本与调度初值。
// Quiesce 非空时负责进入业务写入暂停区，并返回释放函数，调用方保证函数有效。
type Config struct {
	BackupsDir     string
	DataDir        string
	AssetsDir      string
	BrandingDir    string
	AvatarsDir     string
	WikiVersion    string
	Interval       time.Duration
	RetentionCount int
	Quiesce        func() (func(), error)
}

// backupMeta 位于归档内部，记录快照标识、归档创建时间和应用版本。
type backupMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Version   string    `json:"version"`
}

// createSnapshot 创建输出目录并分配未占用标识，取得数据库维护锁与可选业务屏障，
// 再导出数据库、写 ZIP 和旁置摘要。屏障一直保持到函数返回，不只覆盖数据库导出。
// 上下文交给数据库导出，磁盘遍历与压缩不直接检查此上下文。
func createSnapshot(ctx context.Context, cfg Config) (string, error) {
	if err := os.MkdirAll(cfg.BackupsDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backups directory: %w", err)
	}

	id, err := uniqueSnapshotID(cfg.BackupsDir, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("failed to allocate snapshot id: %w", err)
	}

	// 标识分配发生在维护锁之前，名称检查不等同于原子占用输出路径。
	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	if cfg.Quiesce != nil {
		release, err := cfg.Quiesce()
		if err != nil {
			return "", err
		}
		defer release()
	}
	databaseBytes, err := database.Export(ctx, cfg.DataDir)
	if err != nil {
		return "", err
	}
	createdAt := time.Now().UTC()
	zipPath := filepath.Join(cfg.BackupsDir, id+".zip")
	if err := writeSnapshotZip(zipPath, cfg, id, createdAt, databaseBytes); err != nil {
		_ = os.Remove(zipPath) // 尝试删除写入失败的归档，删除错误不覆盖主要写入错误。
		return "", fmt.Errorf("failed to write snapshot zip: %w", err)
	}

	info, err := os.Stat(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat snapshot zip: %w", err)
	}

	// ZIP 成功后才记录实际字节数。后续 Stat 或旁置 JSON 写入失败不在此删除 ZIP。
	entry := SnapshotEntry{ID: id, CreatedAt: createdAt, SizeBytes: info.Size()}
	sidecarPath := filepath.Join(cfg.BackupsDir, id+".json")
	if err := writeJSONFile(sidecarPath, entry); err != nil {
		return "", fmt.Errorf("failed to write snapshot sidecar: %w", err)
	}

	return id, nil
}

// uniqueSnapshotID 用传入时间生成秒级名称，同时检查同名 ZIP 与 JSON。
// 任一存在就依次尝试 -2、-3 等后缀；非不存在类的 Stat 错误直接返回。
// 此函数只检查名称，不创建占位文件，不对并发调用提供原子预留。
func uniqueSnapshotID(backupsDir string, t time.Time) (string, error) {
	base := "snapshot-" + t.Format("20060102-150405")
	id := base
	for i := 2; ; i++ {
		_, zipErr := os.Stat(filepath.Join(backupsDir, id+".zip"))
		_, jsonErr := os.Stat(filepath.Join(backupsDir, id+".json"))
		if os.IsNotExist(zipErr) && os.IsNotExist(jsonErr) {
			return id, nil
		}
		if zipErr != nil && !os.IsNotExist(zipErr) {
			return "", zipErr
		}
		if jsonErr != nil && !os.IsNotExist(jsonErr) {
			return "", jsonErr
		}
		id = fmt.Sprintf("%s-%d", base, i)
	}
}

// writeSnapshotZip 创建或截断输出文件，写入数据库、四类资源和版本元数据。
// checksums.json 记录之前成功写入的各条目 SHA-256，不包含它自己的摘要。
// ZIP 写入器关闭成功后同步文件，延迟关闭把相关关闭错误合并到返回值。
func writeSnapshotZip(zipPath string, cfg Config, id string, createdAt time.Time, databaseBytes []byte) (resultErr error) {
	file, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(file)
	// 显式关闭与延迟关闭共用一次执行结果，避免重复结束 ZIP 写入器。
	closeWriter := sync.OnceValue(writer.Close)
	checksums := map[string]string{}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	defer func() { resultErr = errors.Join(resultErr, closeWriter()) }()
	if err := addBytesToZip(writer, "database.json", databaseBytes, checksums); err != nil {
		return err
	}
	// map 遍历不保证各资源目录进入归档的固定顺序，条目名称仍使用各自固定前缀。
	directories := map[string]string{"assets": cfg.AssetsDir, "branding": cfg.BrandingDir, "avatars": cfg.AvatarsDir, "revision-assets": filepath.Join(cfg.DataDir, ".leafdoc", "blobs", "assets")}
	for name, directory := range directories {
		if err := addDirToZip(writer, directory, name+"/", checksums); err != nil {
			return err
		}
	}
	meta, err := json.Marshal(backupMeta{ID: id, CreatedAt: createdAt, Version: cfg.WikiVersion})
	if err != nil {
		return err
	}
	if err := addBytesToZip(writer, "backup-meta.json", meta, checksums); err != nil {
		return err
	}
	manifest, err := json.Marshal(checksums)
	if err != nil {
		return err
	}
	if err := addBytesToZip(writer, "checksums.json", manifest); err != nil {
		return err
	}
	if err := closeWriter(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	return nil
}

// addDirToZip 遍历存在的资源目录，对非目录项生成相对路径并加上归档前缀。
// 空路径或不存在目录直接跳过，不写空目录条目，也不在这里专门拒绝符号链接。
func addDirToZip(w *zip.Writer, srcDir, prefix string, checksums ...map[string]string) error {
	if srcDir == "" {
		return nil
	}
	if _, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat %s: %w", srcDir, err)
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
		return addFileToZip(w, path, prefix+filepath.ToSlash(rel), checksums...)
	})
}

// addFileToZip 打开源路径，把读取内容同时写入 ZIP 和摘要计算器。
// 空路径或打开时已不存在的文件直接跳过；源句柄始终安排关闭，关闭错误不返回。
// 可变参数只使用第一个摘要映射，调用方需保证传入映射已初始化。
func addFileToZip(w *zip.Writer, srcFile, name string, checksums ...map[string]string) error {
	if srcFile == "" {
		return nil
	}
	f, err := os.Open(srcFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open %s: %w", srcFile, err)
	}
	defer func() { _ = f.Close() }()

	zw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", name, err)
	}
	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(zw, hasher), f); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", name, err)
	}
	if len(checksums) > 0 {
		checksums[0][name] = hex.EncodeToString(hasher.Sum(nil))
	}
	return nil
}

// addBytesToZip 将内存数据写入指定归档条目，写入成功后按需登记摘要。
// 没有摘要映射时仍正常写入，后续摘要参数不会被使用。
func addBytesToZip(w *zip.Writer, name string, data []byte, checksums ...map[string]string) error {
	zw, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", name, err)
	}
	if _, err := zw.Write(data); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", name, err)
	}
	if len(checksums) > 0 {
		sum := sha256.Sum256(data)
		checksums[0][name] = hex.EncodeToString(sum[:])
	}
	return nil
}

// writeJSONFile 使用两空格缩进序列化并直接写入目标文件，不做临时文件原子替换。
// 序列化失败不会开始写入，写入失败附带目标路径返回。
func writeJSONFile(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}
