// 本文件编排快照创建、数量保留、状态展示和归档文件访问。
// 运行标记排除本管理器内的并发快照，列表与删除本身不占用该任务标记。
package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// ErrAlreadyRunning 在同步或异步入口无法取得运行标记时返回，使用固定本地化错误。
var ErrAlreadyRunning = sharederrors.NewLocalizedError(
	"snapshot_already_running",
	"A snapshot is already in progress",
	"a snapshot is already in progress",
	nil,
)

const errFailedToListSnapshots = "Failed to list snapshots"

// SnapshotEntry 是 ZIP 旁 JSON 文件的摘要，列表依此显示标识、创建时间和压缩字节数。
type SnapshotEntry struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	SizeBytes int64     `json:"sizeBytes"`
}

// Manager 分别用配置锁和 Status 内部锁保护设置与任务状态。
// 配置读取返回值副本，不在归档全程持有配置锁。
type Manager struct {
	cfgMu    sync.RWMutex
	cfg      Config
	settings Settings
	status   Status
}

// NewManager 保存资源配置并按间隔与保留数生成默认设置，不在构造时创建快照。
func NewManager(cfg Config) *Manager {
	return &Manager{
		cfg:      cfg,
		settings: DefaultSettings(cfg.Interval, cfg.RetentionCount),
	}
}

// Settings 在读锁下取得当前设置副本，供调度器和保留清理使用。
func (m *Manager) Settings() Settings {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.settings
}

// UpdateSettings 保存设置并同步配置中的保留数，不在这里校验输入或通知调度器。
func (m *Manager) UpdateSettings(value Settings) {
	m.cfgMu.Lock()
	defer m.cfgMu.Unlock()
	m.settings = value
	m.cfg.RetentionCount = value.RetentionCount
}

// config 返回资源配置副本，回调和路径值按结构字段复制。
func (m *Manager) config() Config {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.cfg
}

// TriggerSnapshot 先取得运行标记，再启动后台归档，不等待执行结果。
// 后台错误保存到状态，协程不把错误返回给这个已经结束的调用。
func (m *Manager) TriggerSnapshot() error {
	if !m.status.TryStart() {
		return ErrAlreadyRunning
	}
	go func() {
		_ = m.runOnceLocked(context.Background())
	}()
	return nil
}

// RunOnce 同步执行快照、保留清理和状态更新，已有任务时立即返回专用错误。
// 上下文传入数据库导出，但不在此对整个文件归档过程增加取消检查。
func (m *Manager) RunOnce(ctx context.Context) error {
	if !m.status.TryStart() {
		return ErrAlreadyRunning
	}
	return m.runOnceLocked(ctx)
}

// runOnceLocked 要求调用方已取得运行标记。归档失败结束运行并返回错误，
// 保留清理失败则记警告和独立状态，归档仍算成功。可恢复的 panic 转成错误并释放标记。
// 名称中的 Locked 表示任务标记已占用，不表示配置互斥锁贯穿整个执行过程。
func (m *Manager) runOnceLocked(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during snapshot: %v", r)
			m.status.SetError(err.Error())
		}
	}()

	if _, createErr := createSnapshot(ctx, m.config()); createErr != nil {
		m.status.SetError(createErr.Error())
		return createErr
	}

	// 清理完成前保持运行标记，避免同一管理器同时开始另一轮归档。
	var pruneErrMsg string
	if pruneErr := m.pruneOldSnapshots(); pruneErr != nil {
		slog.Warn("snapshot retention pruning failed", "error", pruneErr)
		pruneErrMsg = pruneErr.Error()
	}
	m.status.SetSuccess(time.Now().UTC(), pruneErrMsg)
	return nil
}

// pruneOldSnapshots 按当前设置读取保留数，从最新优先列表中删除超出数量的旧项。
// 非正数不清理；任一删除失败立即结束，不继续处理剩余旧项。
func (m *Manager) pruneOldSnapshots() error {
	retentionCount := m.Settings().RetentionCount
	if retentionCount <= 0 {
		return nil
	}
	entries, err := m.List()
	if err != nil {
		return err
	}
	if len(entries) <= retentionCount {
		return nil
	}
	for _, entry := range entries[retentionCount:] {
		if err := m.Delete(entry.ID); err != nil {
			return err
		}
		slog.Info("snapshot pruned", "id", entry.ID, "retention_count", retentionCount)
	}
	return nil
}

// Status 返回内部状态的安全副本，不暴露状态锁或可修改的时间字段地址。
func (m *Manager) Status() StatusSnapshot {
	return m.status.Snapshot()
}

// List 读取备份目录内所有非目录且以小写 .json 结尾的文件，按元数据时间降序排列。
// 不检查对应 ZIP 是否存在、大小是否一致或标识是否匹配文件名，
// 任一 JSON 读取或解析失败都会终止整个列表，目录不存在则返回空列表。
func (m *Manager) List() ([]SnapshotEntry, error) {
	dirEntries, err := os.ReadDir(m.config().BackupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotEntry{}, nil
		}
		return nil, sharederrors.NewLocalizedError(
			"snapshot_list_failed",
			errFailedToListSnapshots,
			"failed to list snapshots",
			err,
		)
	}

	entries := make([]SnapshotEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if de.IsDir() || !strings.HasSuffix(de.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.config().BackupsDir, de.Name()))
		if err != nil {
			return nil, sharederrors.NewLocalizedError(
				"snapshot_list_failed",
				errFailedToListSnapshots,
				"failed to read snapshot metadata %s",
				err,
				de.Name(),
			)
		}
		var entry SnapshotEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, sharederrors.NewLocalizedError(
				"snapshot_list_failed",
				errFailedToListSnapshots,
				"failed to parse snapshot metadata %s",
				err,
				de.Name(),
			)
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}

// Delete 校验标识后依次尝试删除 ZIP 和 JSON，两次删除均会执行。
// 两者均不存在才返回未找到，只有一项缺失仍可成功；其他删除错误返回专用错误，
// 不恢复已经删除的另一项。
func (m *Manager) Delete(id string) error {
	if err := validateSnapshotID(id); err != nil {
		return err
	}

	backupsDir := m.config().BackupsDir
	zipPath := filepath.Join(backupsDir, id+".zip")
	jsonPath := filepath.Join(backupsDir, id+".json")

	zipErr := os.Remove(zipPath)
	jsonErr := os.Remove(jsonPath)

	if os.IsNotExist(zipErr) && os.IsNotExist(jsonErr) {
		return sharederrors.NewLocalizedError(
			"snapshot_not_found",
			"Snapshot not found",
			"snapshot %s not found",
			nil,
			id,
		)
	}
	if zipErr != nil && !os.IsNotExist(zipErr) {
		return sharederrors.NewLocalizedError(
			"snapshot_delete_failed",
			"Failed to delete snapshot",
			"failed to delete snapshot %s",
			zipErr,
			id,
		)
	}
	if jsonErr != nil && !os.IsNotExist(jsonErr) {
		return sharederrors.NewLocalizedError(
			"snapshot_delete_failed",
			"Failed to delete snapshot",
			"failed to delete snapshot %s",
			jsonErr,
			id,
		)
	}
	return nil
}

// BackupsDir 返回配置中的备份目录原值，供下载处理入口使用。
func (m *Manager) BackupsDir() string {
	return m.config().BackupsDir
}

// SnapshotZipPath 校验标识并检查 ZIP 路径可被 Stat 访问，返回拼接后的磁盘路径。
// 只有配置目录为绝对路径时结果才是绝对路径；此入口不检查路径是否为普通文件。
func (m *Manager) SnapshotZipPath(id string) (string, error) {
	if err := validateSnapshotID(id); err != nil {
		return "", err
	}
	path := filepath.Join(m.config().BackupsDir, id+".zip")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", sharederrors.NewLocalizedError(
				"snapshot_not_found",
				"Snapshot not found",
				"snapshot %s not found",
				nil,
				id,
			)
		}
		return "", sharederrors.NewLocalizedError(
			"snapshot_internal_error",
			"Failed to access snapshot",
			"failed to access snapshot %s",
			err,
			id,
		)
	}
	return path, nil
}

// validateSnapshotID 要求固定开头，并拒绝目录分隔符和任意连续双点。
// 不验证日期格式、时间是否真实、名称长度或其他文件名字符。
func validateSnapshotID(id string) error {
	if !strings.HasPrefix(id, "snapshot-") || strings.ContainsAny(id, "/\\") || strings.Contains(id, "..") {
		return sharederrors.NewLocalizedError(
			"snapshot_invalid_id",
			"Invalid snapshot id",
			"invalid snapshot id %s",
			nil,
			id,
		)
	}
	return nil
}
