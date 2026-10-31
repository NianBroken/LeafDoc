// 本文件把历史描述、正文和清单写入 MySQL，资源字节块按内容摘要保存在数据目录。
// 数据库与文件系统操作不在一个共同事务中，调用方须分别处理相应失败。
package revision

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// MySQLStore 保存存储目录与可选外部执行器，未绑定时使用数据库共享连接池。
type MySQLStore struct {
	executor   database.Executor
	storageDir string
	log        *slog.Logger
}

// NewMySQLStore 保存配置并派生组件日志器，不在构造时建立连接或创建目录。
func NewMySQLStore(storageDir string, logger *slog.Logger) *MySQLStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &MySQLStore{
		storageDir: storageDir,
		log:        logger.With("component", "MySQLStore"),
	}
}

// SaveContentBlob 校验页面 ID，以正文 SHA-256 作为键的一部分写入 MySQL。
// 数据库写入失败时仍返回已计算哈希和错误，调用方必须检查错误。
func (s *MySQLStore) SaveContentBlob(pageID string, content []byte) (string, error) {
	if err := validateStorageID(pageID); err != nil {
		return "", err
	}
	hash := sha256HexBytes(content)
	return hash, s.records().Write("revision_content", pageID+":"+hash, content)
}

// SaveAssetBlobFromPath 将源文件复制到临时文件，同时计算摘要和字节数。
// 目标按摘要分片保存，已存在时丢弃临时文件，不重新验证现有目标的内容。
// 复制、权限设置、关闭或移动失败均清理临时文件，清理失败不覆盖主要返回错误。
func (s *MySQLStore) SaveAssetBlobFromPath(srcPath string) (string, int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("open live asset %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()

	tmpDir := filepath.Join(s.baseDir(), "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("ensure tmp dir: %w", err)
	}

	tmp, err := os.CreateTemp(tmpDir, "asset-blob-*")
	if err != nil {
		return "", 0, fmt.Errorf("create temp asset blob: %w", err)
	}

	cleanupTmp := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), src)
	if err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("copy asset to temp blob: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	dst := s.AssetBlobPath(hash)

	if err := tmp.Chmod(0o644); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("chmod temp asset blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("close temp asset blob: %w", err)
	}

	if fileExists(dst) {
		_ = os.Remove(tmp.Name())
		return hash, written, nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("ensure asset blob dir: %w", err)
	}

	// 并发写入已生成同名目标时接受该目标，其余移动失败向上传递。
	if err := os.Rename(tmp.Name(), dst); err != nil {
		if fileExists(dst) {
			_ = os.Remove(tmp.Name())
			return hash, written, nil
		}
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("move asset blob into place: %w", err)
	}

	return hash, written, nil
}

// SaveAssetManifest 复制并排序资源列表后编码为 JSON，以编码内容哈希写入共享清单记录。
// 相同名称按资源哈希次序排列，不在此处去重、验证资源存在或核对字节数。
func (s *MySQLStore) SaveAssetManifest(items []AssetRef) (string, error) {
	raw, err := json.Marshal(assetManifest{Items: cloneAndSortAssetRefs(items)})
	if err != nil {
		return "", err
	}
	hash := sha256HexBytes(raw)
	return hash, s.records().Write("revision_manifests", hash, raw)
}

// SaveRevision 校验非空描述、页面 ID、历史 ID 和创建时间，再写入 JSON 描述。
// 同一页面和历史 ID 已存在时覆盖载荷，因此该存储入口不强制历史描述不可变。
func (s *MySQLStore) SaveRevision(rev *Revision) error {
	if rev == nil {
		return fmt.Errorf("revision is required")
	}
	if err := validateStorageID(rev.PageID); err != nil {
		return err
	}
	if err := validateStorageID(rev.ID); err != nil {
		return err
	}
	if rev.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	raw, err := json.Marshal(rev)
	if err != nil {
		return err
	}
	return s.records().Write("revisions", rev.PageID+":"+rev.ID, raw)
}

// ListRevisions 使用不限数量的分页入口获取页面全部历史。
func (s *MySQLStore) ListRevisions(pageID string) ([]*Revision, error) {
	revisions, _, err := s.ListRevisionsPage(pageID, "", 0)
	if err != nil {
		return nil, err
	}
	return revisions, nil
}

// ListRevisionsPage 先读取 revisions 命名空间全部载荷，再按键前缀筛选页面。
// 匹配记录解码后按创建时间降序、同时间按 ID 降序排序，分页在内存中完成。
// 游标存在时从其后一项开始，未知游标得到空页；非正数 limit 不限制数量。
func (s *MySQLStore) ListRevisionsPage(pageID, cursor string, limit int) ([]*Revision, string, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, "", err
	}
	records, err := s.records().List("revisions")
	if err != nil {
		return nil, "", err
	}
	all := []*Revision{}
	for key, raw := range records {
		if !strings.HasPrefix(key, pageID+":") {
			continue
		}
		var rev Revision
		if err = json.Unmarshal(raw, &rev); err != nil {
			return nil, "", err
		}
		all = append(all, &rev)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	// 游标本身不包含时间信息，通过当前排序列表中的 ID 位置决定起点。
	start := 0
	if cursor != "" {
		start = len(all)
		for i, rev := range all {
			if rev.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	end := len(all)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	next := ""
	if end < len(all) && end > start {
		next = all[end-1].ID
	}
	return all[start:end], next, nil
}

// GetLatestRevision 获取第一页的一项，没有历史时返回 nil 且无错误。
func (s *MySQLStore) GetLatestRevision(pageID string) (*Revision, error) {
	items, _, err := s.ListRevisionsPage(pageID, "", 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

// GetRevision 校验页面 ID 并读取组合键，找不到记录时映射为 os.ErrNotExist。
// revisionID 不在本方法单独校验，JSON 解码失败与已分配描述指针一起返回。
func (s *MySQLStore) GetRevision(pageID, revisionID string) (*Revision, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, err
	}
	raw, err := s.records().Read("revisions", pageID+":"+revisionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	var rev Revision
	err = json.Unmarshal(raw, &rev)
	return &rev, err
}

// UpdateRevision 先要求原记录可读，再通过 SaveRevision 覆盖描述，供连续保存合并使用。
// 存在性检查与写入不是本方法创建的单一事务。
func (s *MySQLStore) UpdateRevision(rev *Revision) error {
	if rev == nil {
		return fmt.Errorf("revision is required")
	}
	if _, err := s.GetRevision(rev.PageID, rev.ID); err != nil {
		return err
	}
	return s.SaveRevision(rev)
}

// PruneRevisions 按当前排序保留前 keepCount 项，删除其余历史描述。
// 非正数上限不清理，正文载荷、资源清单和磁盘资源块都不在此方法删除。
// 普通连接使用事务批量删除，绑定执行器时借用外部事务，回滚错误合并到返回值。
func (s *MySQLStore) PruneRevisions(pageID string, keepCount int) (resultErr error) {
	if keepCount <= 0 {
		return nil
	}
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	if len(items) <= keepCount {
		return nil
	}
	db, err := s.records().Connection()
	if err != nil {
		return err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return err
	}
	defer database.Rollback(tx, &resultErr)
	for _, rev := range items[keepCount:] {
		if _, err = tx.Exec("DELETE FROM content_records WHERE namespace='revisions' AND record_key=?", pageID+":"+rev.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ReadContentBlob 校验页面 ID 后读取正文，空哈希直接表示空正文。
// 没有对应数据库记录时返回 os.ErrNotExist，不在读取阶段重新校验正文哈希。
func (s *MySQLStore) ReadContentBlob(pageID, hash string) ([]byte, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, err
	}
	if hash == "" {
		return []byte{}, nil
	}
	raw, err := s.records().Read("revision_content", pageID+":"+hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, os.ErrNotExist
	}
	return raw, err
}

// OpenContentBlob 先完整读取正文到内存，再包装为可关闭读取器。
// 该接口便于与流式调用方配合，但不会降低数据库正文读取时的内存占用。
// 返回的 NopCloser 关闭时不释放额外底层资源，调用方仍按接口约定关闭。
func (s *MySQLStore) OpenContentBlob(pageID, hash string) (io.ReadCloser, error) {
	raw, err := s.ReadContentBlob(pageID, hash)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

// DeleteContentBlobIfUnreferenced 扫描该页历史，仍有描述引用哈希时保留正文记录。
// 没有引用才删除对应载荷，查询或删除错误直接返回，是否记录日志由调用方决定。
// 扫描和删除不在本方法创建的共同事务中。
func (s *MySQLStore) DeleteContentBlobIfUnreferenced(pageID, hash string) error {
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	for _, rev := range items {
		if rev.ContentHash == hash {
			return nil
		}
	}
	return s.records().Delete("revision_content", pageID+":"+hash)
}

// LoadAssetManifest 读取并解码共享清单，返回复制排序后的资源引用。
// 空哈希返回空列表，非空哈希没有记录时保留数据库错误，不校验清单哈希。
func (s *MySQLStore) LoadAssetManifest(hash string) ([]AssetRef, error) {
	if hash == "" {
		return []AssetRef{}, nil
	}
	raw, err := s.records().Read("revision_manifests", hash)
	if err != nil {
		return nil, err
	}
	var manifest assetManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return cloneAndSortAssetRefs(manifest.Items), nil
}

// ReadAssetBlob 裁去哈希边缘空白并完整读入磁盘资源，不重新计算摘要。
func (s *MySQLStore) ReadAssetBlob(hash string) ([]byte, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, fmt.Errorf("asset hash is required")
	}

	raw, err := os.ReadFile(s.AssetBlobPath(hash))
	if err != nil {
		return nil, fmt.Errorf("read asset blob: %w", err)
	}
	return raw, nil
}

// OpenAssetBlob 打开哈希对应的资源文件，拒绝空哈希但不验证哈希格式或文件内容。
// 成功返回的文件句柄由调用方负责关闭。
func (s *MySQLStore) OpenAssetBlob(hash string) (*os.File, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, fmt.Errorf("asset hash is required")
	}
	f, err := os.Open(s.AssetBlobPath(hash))
	if err != nil {
		return nil, fmt.Errorf("open asset blob: %w", err)
	}
	return f, nil
}

// CopyAssetBlobToPath 在目标目录创建临时文件，流式复制后校验实际摘要和大小。
// 两项均一致才重命名到目标路径，目标父目录须已存在，移动语义由文件系统执行。
// 失败路径尝试删除临时文件，源文件延迟关闭，清理错误不单独向上传递。
func (s *MySQLStore) CopyAssetBlobToPath(hash string, expectedSize int64, dstPath string) error {
	hash = strings.ToLower(strings.TrimSpace(hash))
	src, err := s.OpenAssetBlob(hash)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	tmpDir := filepath.Dir(dstPath)
	tmp, err := os.CreateTemp(tmpDir, "asset-restore-*")
	if err != nil {
		return fmt.Errorf("create temp restore file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = tmp.Close(); _ = os.Remove(tmpName) }

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), src)
	if err != nil {
		cleanup()
		return fmt.Errorf("stream asset blob to %s: %w", dstPath, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod restored asset: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp restore file: %w", err)
	}
	if computedHash := hex.EncodeToString(hasher.Sum(nil)); computedHash != hash {
		_ = os.Remove(tmpName)
		return fmt.Errorf("asset blob hash mismatch: computed %s, want %s", computedHash, hash)
	}
	if written != expectedSize {
		_ = os.Remove(tmpName)
		return fmt.Errorf("asset blob size mismatch: got %d, want %d", written, expectedSize)
	}
	if err := os.Rename(tmpName, dstPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("move restored asset into place: %w", err)
	}
	return nil
}

// DeletePageRevisions 逐条删除页面历史，再删除正文命名空间中该页面前缀的记录。
// 中途失败立即返回，普通共享连接下已完成删除不回滚，清单和磁盘资源块保留。
func (s *MySQLStore) DeletePageRevisions(pageID string) error {
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	for _, rev := range items {
		if err = s.records().Delete("revisions", pageID+":"+rev.ID); err != nil {
			return err
		}
	}
	records, err := s.records().List("revision_content")
	if err != nil {
		return err
	}
	for key := range records {
		if strings.HasPrefix(key, pageID+":") {
			if err = s.records().Delete("revision_content", key); err != nil {
				return err
			}
		}
	}
	return nil
}

// baseDir 返回数据目录下历史资源所用隐藏目录，不在此处创建目录。
func (s *MySQLStore) baseDir() string {
	return filepath.Join(s.storageDir, ".leafdoc")
}

// contentBlobPath 构造诊断用数据库记录定位文本，不是文件系统路径。
func (s *MySQLStore) contentBlobPath(pageID, hash string) string {
	return "mysql:revision_content:" + pageID + ":" + hash
}

// AssetBlobPath 用摘要前两字节分片构造资源路径，不执行存在性或格式校验。
func (s *MySQLStore) AssetBlobPath(hash string) string {
	return filepath.Join(s.baseDir(), "blobs", "assets", "sha256", shardHash(hash), hash)
}

// AssetManifestExists 仅在非空哈希记录可读时返回 true，所有读取错误统一视为 false。
func (s *MySQLStore) AssetManifestExists(hash string) bool {
	if hash == "" {
		return false
	}
	_, err := s.records().Read("revision_manifests", hash)
	return err == nil
}

// assetManifestPath 返回诊断用清单记录位置，不指向本地文件。
func (s *MySQLStore) assetManifestPath(hash string) string {
	return "mysql:revision_manifests:" + hash
}

// validateStorageID 拒绝纯空白、路径分隔符以及精确等于单点或双点的 ID。
// 不裁去返回值、不限制其他标点或长度，调用方仍使用原输入建立记录键。
func validateStorageID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id must not be empty")
	}
	if strings.ContainsAny(id, "/\\") {
		return fmt.Errorf("id must not contain path separators")
	}
	if id == "." || id == ".." {
		return fmt.Errorf("id must not be a dot component")
	}
	return nil
}

// shardHash 使用摘要前两字节作为目录名，不足两字节时使用 00。
func shardHash(hash string) string {
	if len(hash) < 2 {
		return "00"
	}
	return hash[:2]
}

// sha256HexBytes 计算完整字节序列的 SHA-256 并返回小写十六进制字符串。
func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// cloneAndSortAssetRefs 复制引用列表，按名称及同名时的哈希稳定排序，不改动原切片。
func cloneAndSortAssetRefs(items []AssetRef) []AssetRef {
	cloned := make([]AssetRef, len(items))
	copy(cloned, items)

	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Name == cloned[j].Name {
			return cloned[i].SHA256 < cloned[j].SHA256
		}
		return cloned[i].Name < cloned[j].Name
	})

	return cloned
}

// fileExists 将任何 Stat 错误视为不存在，不额外验证对象为常规文件。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// records 生成带目录与当前执行器的记录访问器，不在构造访问器时建立连接。
func (s *MySQLStore) records() database.Records {
	return database.Records{Dir: s.storageDir, Executor: s.executor}
}
