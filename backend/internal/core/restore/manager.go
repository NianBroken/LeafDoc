// 本文件编排后台恢复、上传暂存和独立进度凭据。归档先解压验证，
// 再在维护锁和业务写入屏障内替换数据库与资源，重载成功后才重新开放写入。
package restore

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	coreshared "github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// ErrRestoreAlreadyRunning 在当前任务仍运行时拒绝另一项恢复，使用固定本地化错误。
var ErrRestoreAlreadyRunning = sharederrors.NewLocalizedError(
	"restore_already_running",
	"A restore is already in progress",
	"a restore is already in progress",
	nil,
)

// ErrRestoreNeedsIntervention 在任务被标记为需要介入时阻止下一项恢复。
// 管理器保留该状态供管理入口展示并选择重启，不在此自动清除残留资源。
var ErrRestoreNeedsIntervention = sharederrors.NewLocalizedError(
	"restore_needs_intervention",
	"This instance needs attention before a new restore can be started — restart the server first",
	"a previous restore needs intervention before a new restore can be started",
	nil,
)

// ErrWritesDisabled 供维护中间件拒绝写入请求，领域与 HTTP 层共用同一错误定义。
var ErrWritesDisabled = sharederrors.NewLocalizedError(
	"restore_writes_disabled",
	"A restore is in progress; writes are temporarily disabled",
	"a restore is in progress; writes are temporarily disabled",
	nil,
)

// gateDrainTimeout 限制关闭准入后等待在途写入结束的时间。
// 超时会结束恢复并重新开放准入，不进入资源交换。
const gateDrainTimeout = 10 * time.Second

// progressTokenTTL 限定进度凭据从任务开始起的一小时有效窗口，不随查询延长。
const progressTokenTTL = time.Hour

// 未配置正上传限额时，HTTP 上传处理入口读取此默认值。
const DefaultMaxUploadSizeBytes int64 = 500 * 1024 * 1024 // 五百 MiB，以字节表示。

// Manager 的进度锁把启动任务与替换进度凭据串行化，Job 单独保护状态。
// 配置在构造时保存，后台恢复通过等待组纳入进程关闭等待。
type Manager struct {
	progressMu        sync.Mutex
	progressTokenHash [sha256.Size]byte
	hasProgressToken  bool
	progressExpiresAt time.Time
	cfg               Config
	job               *Job
	// 跟踪按标识和上传启动的后台任务，让关闭流程先等待恢复结束再释放依赖服务。
	wg sync.WaitGroup
}

// NewManager 保存配置并创建空闲状态，不验证资源目录或启动协程。
func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg, job: NewJob()}
}

// TriggerRestore 按快照标识启动后台恢复，丢弃带凭据入口返回的进度令牌。
// 快照查找与归档错误发生在后台，通过任务状态展示，不作为此入口的同步错误。
func (m *Manager) TriggerRestore(id string) error {
	_, err := m.TriggerRestoreWithStatusToken(id)
	return err
}

// TriggerRestoreWithStatusToken 在进度锁内开始任务并生成凭据，然后启动后台工作。
// 返回凭据只用于读取当前任务进度，不提供管理员业务操作权限。
func (m *Manager) TriggerRestoreWithStatusToken(id string) (string, error) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	token, err := m.startProgressJob()
	if err != nil {
		return "", err
	}
	m.wg.Go(func() { m.runLocked(id) })
	return token, nil
}

// startProgressJob 要求调用方持有 progressMu，先拒绝介入和运行状态，
// 再生成三十二个随机字节并开始任务。只有成功开始才替换凭据摘要与到期时间。
// 完整凭据以十六进制返回，管理器只保存其 SHA-256，不写入任务 JSON。
func (m *Manager) startProgressJob() (string, error) {
	if m.job.Status().NeedsIntervention {
		return "", ErrRestoreNeedsIntervention
	}
	if m.job.Status().Running {
		return "", ErrRestoreAlreadyRunning
	}
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate restore progress credential: %w", err)
	}
	if !m.job.Start() {
		return "", ErrRestoreAlreadyRunning
	}
	token := hex.EncodeToString(random[:])
	m.progressTokenHash = sha256.Sum256([]byte(token))
	m.hasProgressToken = true
	m.progressExpiresAt = time.Now().Add(progressTokenTTL)
	return token, nil
}

// ProgressStatus 计算传入文本的摘要，与当前固定长度摘要作常量时间比较。
// 只有凭据存在、相等且严格未到期才返回状态；任务完成不自动撤销凭据，
// 下一任务开始或管理器重建后旧凭据不再有效。
func (m *Manager) ProgressStatus(token string) (JobStatus, bool) {
	candidate := sha256.Sum256([]byte(token))
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	matches := subtle.ConstantTimeCompare(candidate[:], m.progressTokenHash[:])
	if !m.hasProgressToken || matches != 1 || !time.Now().Before(m.progressExpiresAt) {
		return JobStatus{}, false
	}
	return m.job.Status(), true
}

// TriggerRestoreFromUpload 同步把上传流读到临时文件，再启动后台恢复并丢弃进度凭据。
// 上传流应在请求返回前读完，后台任务只依赖暂存路径，不继续读取请求体。
func (m *Manager) TriggerRestoreFromUpload(r io.Reader) error {
	_, err := m.TriggerRestoreFromUploadWithStatusToken(r)
	return err
}

// TriggerRestoreFromUploadWithStatusToken 先开始任务并生成凭据，再在同一进度锁内
// 同步暂存上传。暂存失败结束任务并返回空令牌；成功后把临时路径交给后台处理。
// 进度凭据独立于账号会话，恢复清空刷新会话后仍可查询当前任务。
func (m *Manager) TriggerRestoreFromUploadWithStatusToken(r io.Reader) (string, error) {
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	token, err := m.startProgressJob()
	if err != nil {
		return "", err
	}
	zipPath, cleanup, err := m.stageUpload(r)
	if err != nil {
		m.job.Finish(err)
		return "", err
	}
	m.wg.Go(func() {
		m.runGuarded(func() { defer cleanup(); m.runFromZipPath(zipPath, coreshared.DefaultZipExtractionLimits, cleanup) })
	})
	return token, nil
}

// stageUpload 在数据目录创建临时 ZIP 并读完上传流，关闭文件后返回路径。
// 此方法不限制读取字节数，不调用 Sync；上传限额由 HTTP 层使用管理器提供的值执行。
// 清理函数使用 Once，多次调用只尝试删除一次，删除错误不返回。
func (m *Manager) stageUpload(r io.Reader) (zipPath string, cleanup func(), err error) {
	f, err := os.CreateTemp(m.cfg.DataDir, ".leafdoc-restore-upload-*.zip")
	if err != nil {
		return "", nil, errStageUploadFailed(err)
	}
	var release sync.Once
	cleanup = func() { release.Do(func() { _ = os.Remove(f.Name()) }) }

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		cleanup()
		return "", nil, errStageUploadFailed(err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, errStageUploadFailed(err)
	}
	return f.Name(), cleanup, nil
}

// errStageUploadFailed 将同步暂存错误封装为本地化错误，同时保留底层原因。
// 调用方需传入非空错误，消息参数使用该错误的文本。
func errStageUploadFailed(cause error) error {
	return sharederrors.NewLocalizedError(
		"restore_upload_staging_failed",
		"Failed to save the uploaded backup to disk",
		"failed to stage uploaded backup: %s",
		cause,
		cause.Error(),
	)
}

// Status 返回当前任务状态副本，不验证进度凭据；访问控制由调用入口负责。
func (m *Manager) Status() JobStatus {
	return m.job.Status()
}

// MaxUploadSizeBytes 返回正配置限额，零或负数使用默认上传限额。
func (m *Manager) MaxUploadSizeBytes() int64 {
	if m.cfg.MaxUploadSizeBytes > 0 {
		return m.cfg.MaxUploadSizeBytes
	}
	return DefaultMaxUploadSizeBytes
}

// Wait 等待已经登记的后台恢复退出，不取消正在执行的工作。
// 调用方在关闭流程中先停止接收新任务，再等待并关闭依赖资源。
func (m *Manager) Wait() {
	m.wg.Wait()
}

// SelfRestart 委托平台实现重启，此方法不自行检查介入标记或请求权限。
func (m *Manager) SelfRestart() error {
	return SelfRestart()
}

// runLocked 从备份管理器取得已有快照路径，使用不限制大小的解压配置。
// 路径查询失败保存在验证阶段状态中；所有后台执行交给 panic 保护入口。
// 此处的本地来源选择不是对文件内容可信性的证明，内容仍须经过归档校验。
func (m *Manager) runLocked(id string) {
	m.runGuarded(func() {
		zipPath, err := m.cfg.SnapshotManager.SnapshotZipPath(id)
		if err != nil {
			m.job.SetPhase(PhaseValidating)
			m.job.Finish(err)
			return
		}
		m.runFromZipPath(zipPath, coreshared.UnrestrictedExtractionLimits, nil)
	})
}

// runGuarded 捕获 fn 的可恢复 panic，记录错误并把任务标为需要介入。
// 它不自行 Engage 或 Disengage，写入屏障保留 panic 发生时的状态。
func (m *Manager) runGuarded(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("panic during restore", "panic", r)
			m.job.FinishNeedsIntervention(fmt.Errorf("panic during restore: %v", r))
		}
	}()
	fn()
}

// runFromZipPath 先在维护锁之外解压校验，随后释放上传临时文件。
// 校验成功才取得维护锁并关闭新写入准入；在途写入超时则退出而不交换数据。
// 数据恢复或运行态重载失败按错误类型决定是否保留关闭状态，成功才重新开放。
func (m *Manager) runFromZipPath(zipPath string, limits coreshared.ExtractionLimits, releaseUpload func()) {
	m.job.SetPhase(PhaseValidating)
	staging, _, err := extractAndValidateWithLimits(zipPath, m.cfg.DataDir, limits)
	if releaseUpload != nil {
		releaseUpload()
	}
	if err != nil {
		m.job.Finish(err)
		return
	}
	defer func() {
		if err := os.RemoveAll(staging); err != nil {
			m.job.recordCleanupError(err)
		}
	}()
	// 暂存目录在函数退出时删除，删除错误合并入任务状态，不在此返回同步错误。
	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	m.cfg.WriteGate.Engage()
	if !m.cfg.WriteGate.WaitForDrain(gateDrainTimeout) {
		m.cfg.WriteGate.Disengage()
		m.job.Finish(fmt.Errorf("maintenance drain timed out"))
		return
	}
	m.job.SetPhase(PhaseSwapping)
	if err := restoreStaged(m.cfg.DataDir, staging); err != nil {
		var intervention *interventionError
		if errors.As(err, &intervention) {
			m.job.FinishNeedsIntervention(err)
			return
		}
		m.cfg.WriteGate.Disengage()
		m.job.Finish(err)
		return
	}
	// 数据已提交后重载设置和用户解析状态，失败不会在此回滚已提交数据库。
	if err := m.reloadAll(); err != nil {
		m.job.FinishNeedsIntervention(err)
		return
	}
	if m.cfg.UserResolver != nil {
		if err := m.cfg.UserResolver.Reload(); err != nil {
			m.job.FinishNeedsIntervention(err)
			return
		}
	}
	// 明确的运行态重载回调优先，只有没有它时才触发重同步回调。
	if m.cfg.ReloadRuntime != nil {
		if err := m.cfg.ReloadRuntime(); err != nil {
			m.job.FinishNeedsIntervention(err)
			return
		}
	} else if m.cfg.TriggerResync != nil {
		m.cfg.TriggerResync()
	}
	m.cfg.WriteGate.Disengage()
	m.job.Finish(nil)
}

// reloadAll 按配置顺序重载每个非空入口，即使某项失败也继续后续项，
// 结束后合并所有错误。重复配置项会被重复调用，不在这里去重或回滚已重载设置。
func (m *Manager) reloadAll() error {
	var errs []error
	for _, r := range m.cfg.Reloadables {
		if r == nil {
			continue
		}
		if err := r.Reload(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
