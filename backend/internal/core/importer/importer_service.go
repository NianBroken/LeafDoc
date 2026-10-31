package importer

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// ImporterService 串联上传暂存、解压、规划、状态保存和计划执行。
// 服务使用单份当前计划，后台执行的进度与取消请求通过 PlanStore 交换。
type ImporterService struct {
	planner                 *Planner
	planStore               *PlanStore
	extractor               *ZipExtractor
	logger                  *slog.Logger
	assetMaxUploadSizeBytes int64
	workspaceBaseDir        string
	zipExtractionLimits     shared.ExtractionLimits
}

// CurrentPlanState 是对外返回的当前计划视图，不包含本机工作区路径和执行用户。
// TreeHash 来自原计划，执行后的树摘要位于 ExecutionResult 中。
type CurrentPlanState struct {
	ID              string           `json:"id"`
	TreeHash        string           `json:"tree_hash"`
	Items           []PlanItem       `json:"items"`
	Errors          []string         `json:"errors"`
	ExecutionStatus ExecutionStatus  `json:"execution_status"`
	CancelRequested bool             `json:"cancel_requested"`
	ExecutionResult *ExecutionResult `json:"execution_result,omitempty"`
	ExecutionError  *string          `json:"execution_error,omitempty"`
	ExecutionProgress
}

// SetAssetMaxUploadSizeBytes 接受正数附件上限，非正数不改变现有配置。
// 若该值超过当前解压单文件上限，同时提高解压上限，使允许上传的附件能被解压。
// 较小值只影响附件上传，不降低已有解压单文件上限。
func (is *ImporterService) SetAssetMaxUploadSizeBytes(n int64) {
	if n > 0 {
		is.assetMaxUploadSizeBytes = n
		if n > is.zipExtractionLimits.MaxEntryBytes {
			is.zipExtractionLimits.MaxEntryBytes = n
		}
	}
}

// SetZipExtractionLimits 直接保存解压限制，供下一次上传解压使用。
// 它不重新与附件大小上限比较，调用顺序会影响最终单文件解压限制。
func (is *ImporterService) SetZipExtractionLimits(limits shared.ExtractionLimits) {
	is.zipExtractionLimits = limits
}

// NewImporterService 设置附件和解压默认限制，并建立工作区的基础目录配置。
// 空工作区路径使用系统临时目录下的 wiki-imports，目录在实际解压时建立。
// 构造完成即检查持久化的运行状态，符合条件时启动后台恢复执行。
func NewImporterService(planner *Planner, planStore *PlanStore, workspaceBaseDir string, assetMaxUploadSizeBytes int64) *ImporterService {
	if assetMaxUploadSizeBytes <= 0 {
		assetMaxUploadSizeBytes = assets.DefaultMaxUploadSizeBytes
	}
	if workspaceBaseDir == "" {
		workspaceBaseDir = filepath.Join(os.TempDir(), "wiki-imports")
	}
	zipExtractionLimits := shared.DefaultZipExtractionLimits
	if assetMaxUploadSizeBytes > zipExtractionLimits.MaxEntryBytes {
		zipExtractionLimits.MaxEntryBytes = assetMaxUploadSizeBytes
	}
	service := &ImporterService{
		planner:                 planner,
		planStore:               planStore,
		extractor:               NewZipExtractor(),
		logger:                  slog.Default().With("component", "ImporterService"),
		assetMaxUploadSizeBytes: assetMaxUploadSizeBytes,
		workspaceBaseDir:        workspaceBaseDir,
		zipExtractionLimits:     zipExtractionLimits,
	}
	service.resumeInterruptedExecution()
	return service
}

// CreateImportPlanFromFolder 扫描指定目录并保存一份待执行计划，目录归入计划工作区。
// 计划完成或清理时会删除该目录，调用方须传入允许由导入流程清理的目录。
// 正在运行的计划不能替换，其他已有计划先删除工作区再清空状态。
func (is *ImporterService) CreateImportPlanFromFolder(folderPath string, targetBasePath string) (*PlanResult, error) {
	// 旧状态读取成功才进入清理；其他读取错误由后续保存操作继续报告。
	if old, err := is.planStore.Get(); err == nil && old != nil {
		if old.ExecutionStatus == ExecutionStatusRunning {
			return nil, ErrImportExecutionRunning
		}
		err = os.RemoveAll(old.WorkspaceRoot)
		if err != nil {
			return nil, fmt.Errorf("cleanup old import workspace: %w", err)
		}
		if _, err := is.planStore.Clear(); err != nil {
			return nil, err
		}
		is.logger.Info("Old import workspace cleaned up")
	}

	entries, err := FindMarkdownEntries(folderPath)
	if err != nil {
		return nil, err
	}

	opts := PlanOptions{
		SourceBasePath: folderPath,
		TargetBasePath: targetBasePath,
	}

	plan, err := is.planner.CreatePlan(entries, opts)
	if err != nil {
		return nil, err
	}

	if err := is.planStore.Set(&StoredPlan{
		Plan:            plan,
		PlanOptions:     opts,
		WorkspaceRoot:   folderPath,
		CreatedAt:       time.Now(),
		ExecutionStatus: ExecutionStatusPlanned,
	}); err != nil {
		return nil, err
	}
	is.logger.Info("Import plan created", "entries", len(entries), "workspace", folderPath)
	return plan, nil
}

// GetCurrentPlan 将状态库返回的计划转换为对外视图，读取错误直接向上传递。
func (is *ImporterService) GetCurrentPlan() (*CurrentPlanState, error) {
	sp, err := is.planStore.Get()
	if err != nil {
		return nil, err
	}
	return currentPlanStateFromStored(sp), nil
}

// ClearCurrentPlan 拒绝清理运行中的计划，其余计划先尝试删除工作区再清空状态。
// 工作区删除失败只记录日志，不阻止清除计划，最终返回值来自状态库 Clear。
func (is *ImporterService) ClearCurrentPlan() error {
	if sp, err := is.planStore.Get(); err == nil && sp != nil {
		if sp.ExecutionStatus == ExecutionStatusRunning {
			return ErrImportExecutionRunning
		}
		if err := os.RemoveAll(sp.WorkspaceRoot); err != nil {
			is.logger.Error("remove workspace failed", "error", err)
		}
	}
	_, err := is.planStore.Clear()
	return err
}

// CancelCurrentPlan 请求当前后台执行在计划项边界停止，返回状态与是否首次设置标记。
func (is *ImporterService) CancelCurrentPlan() (*CurrentPlanState, bool, error) {
	sp, requested, err := is.planStore.RequestCancel()
	if err != nil {
		return nil, false, err
	}
	return currentPlanStateFromStored(sp), requested, nil
}

// ExecuteCurrentPlan 同步占用并执行当前计划，已完成计划直接返回保存的执行结果。
// 执行结束先保存最终状态，再删除工作区；最终状态保存失败时不清理工作区。
// 执行器返回错误时对外返回该错误，部分结果保存在状态库而不随此返回值交付。
func (is *ImporterService) ExecuteCurrentPlan(userID string) (*ExecutionResult, error) {
	sp, started, err := is.planStore.TryStartExecution(userID)
	if err != nil {
		return nil, err
	}
	if !started {
		switch sp.ExecutionStatus {
		case ExecutionStatusRunning:
			return nil, ErrImportExecutionRunning
		case ExecutionStatusCompleted:
			if sp.ExecutionResult == nil {
				return nil, errors.New("import completed without result")
			}
			return sp.ExecutionResult, nil
		case ExecutionStatusFailed:
			if sp.ExecutionError != nil {
				return nil, errors.New(*sp.ExecutionError)
			}
			return nil, errors.New("import execution failed")
		}
	}

	res, err := is.executeStoredPlan(sp)
	if finishErr := is.planStore.FinishExecution(sp.Plan.ID, res, err); finishErr != nil {
		return nil, finishErr
	}
	is.cleanupWorkspace(sp.WorkspaceRoot)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// StartCurrentPlanExecution 成功占用状态后启动后台协程，立即返回启动时的计划视图。
// 重复调用不启动第二个协程，started 表示此调用是否实际开始执行。
// 后台结束先保存结果再清理工作区，保存失败保留工作区并记录错误。
func (is *ImporterService) StartCurrentPlanExecution(userID string) (*CurrentPlanState, bool, error) {
	sp, started, err := is.planStore.TryStartExecution(userID)
	if err != nil {
		return nil, false, err
	}

	if started {
		go func(snapshot *StoredPlan) {
			res, execErr := is.executeStoredPlan(snapshot)
			if finishErr := is.planStore.FinishExecution(snapshot.Plan.ID, res, execErr); finishErr != nil {
				is.logger.Error("failed to persist finished import state", "error", finishErr)
				return
			}
			is.cleanupWorkspace(snapshot.WorkspaceRoot)
		}(sp)
	}

	return currentPlanStateFromStored(sp), started, nil
}

// FindMarkdownEntries 遍历源目录，收集扩展名不区分大小写的 .md 文件相对路径。
// 遍历或相对路径计算失败会放弃整个结果，不返回已经收集的部分列表。
func FindMarkdownEntries(sourceBasePath string) ([]ImportMDFile, error) {
	out := []ImportMDFile{}

	err := filepath.WalkDir(sourceBasePath, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if strings.ToLower(filepath.Ext(d.Name())) != ".md" {
			return nil
		}

		rel, err := filepath.Rel(sourceBasePath, p)
		if err != nil {
			return fmt.Errorf("rel: %w", err)
		}

		out = append(out, ImportMDFile{
			SourcePath: filepath.ToSlash(rel),
		})
		return nil
	})

	// 先按路径层级从浅到深排列，同层按源路径文字排序。
	if err == nil {
		sort.SliceStable(out, func(i, j int) bool {
			depthI := strings.Count(out[i].SourcePath, "/")
			depthJ := strings.Count(out[j].SourcePath, "/")
			if depthI == depthJ {
				return out[i].SourcePath < out[j].SourcePath
			}
			return depthI < depthJ
		})
	}

	// 再稳定地把所有 index.md 放在普通页面之前，两组内部保留上一轮排序。
	// 因此深层分区首页也会先于浅层普通页面处理。
	if err == nil {
		sort.SliceStable(out, func(i, j int) bool {
			nameI := strings.ToLower(filepath.Base(out[i].SourcePath))
			nameJ := strings.ToLower(filepath.Base(out[j].SourcePath))
			if nameI == indexFilename && nameJ != indexFilename {
				return true
			}
			if nameJ == indexFilename && nameI != indexFilename {
				return false
			}
			return false
		})
	}

	if err != nil {
		return nil, err
	}
	return out, nil
}

// CreateImportPlanFromZipUpload 暂存上传流并解压，然后使用解压目录建立计划。
// 建立计划失败时清理解压目录，清理错误记录日志而不覆盖原始建计划错误。
func (is *ImporterService) CreateImportPlanFromZipUpload(
	r io.Reader,
	targetBasePath string,
) (*PlanResult, error) {
	ws, err := is.extractZipReaderToTemp(r)
	if err != nil {
		return nil, fmt.Errorf("extract zip to temp: %w", err)
	}

	plan, err := is.CreateImportPlanFromFolder(ws.Root, targetBasePath)
	if err != nil {
		if err := ws.Cleanup(); err != nil {
			is.logger.Error("cleanup failed", "error", err)
		}
		return nil, fmt.Errorf("create import plan from folder: %w", err)
	}
	return plan, nil
}

// extractZipReaderToTemp 将上传流完整写入临时 ZIP，关闭后交给解压器处理。
// 临时 ZIP 无论成功失败都会尝试删除，成功解压的目录留给计划生命周期管理。
// 上传暂存使用 io.Copy，此处不对输入流累计字节数设置独立上限。
func (is *ImporterService) extractZipReaderToTemp(r io.Reader) (*ZipWorkspace, error) {
	if err := os.MkdirAll(is.workspaceBaseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create import temp dir: %w", err)
	}

	tmp, err := os.CreateTemp(is.workspaceBaseDir, "upload-*.zip")
	if err != nil {
		return nil, fmt.Errorf("create temp zip: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if err := os.Remove(tmpPath); err != nil {
			is.logger.Error("remove temp zip failed", "error", err)
		}
	}()

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		return nil, fmt.Errorf("store uploaded zip: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp zip: %w", err)
	}

	ws, err := is.extractor.extractToDirWithLimits(tmpPath, is.workspaceBaseDir, is.zipExtractionLimits)
	if err != nil {
		return nil, fmt.Errorf("extract zip: %w", err)
	}
	return ws, nil
}

// executeStoredPlan 为状态快照建立执行器，并连接进度保存、取消检查和恢复位置。
// 进度保存失败记录日志，回调不向执行器返回错误，因此不会直接停止执行循环。
func (is *ImporterService) executeStoredPlan(sp *StoredPlan) (*ExecutionResult, error) {
	exec := NewExecutor(sp.Plan, &sp.PlanOptions, is.assetMaxUploadSizeBytes, is.planner.wiki, is.planner.log).
		WithProgressCallback(func(progress ExecutionProgress, result *ExecutionResult) {
			if err := is.planStore.UpdateExecutionProgress(sp.Plan.ID, progress, result); err != nil {
				is.logger.Error("failed to persist import progress", "error", err)
			}
		}).
		WithCancelCheck(func() bool {
			return is.planStore.IsCancelRequested(sp.Plan.ID)
		}).
		WithResumeState(sp.ProcessedItems, sp.ExecutionResult)
	return exec.Execute(sp.ExecutionUserID)
}

// cleanupWorkspace 删除非空工作区路径，失败仅记录日志，不修改已保存的执行状态。
func (is *ImporterService) cleanupWorkspace(workspaceRoot string) {
	if workspaceRoot == "" {
		return
	}
	if err := os.RemoveAll(workspaceRoot); err != nil {
		is.logger.Error("remove workspace failed", "error", err)
	}
}

// currentPlanStateFromStored 组装公开状态字段，空对象或空计划返回空值。
// 此处不深复制计划项和执行结果，隔离程度沿用状态库提供的快照。
func currentPlanStateFromStored(sp *StoredPlan) *CurrentPlanState {
	if sp == nil || sp.Plan == nil {
		return nil
	}

	return &CurrentPlanState{
		ID:              sp.Plan.ID,
		TreeHash:        sp.Plan.TreeHash,
		Items:           sp.Plan.Items,
		Errors:          sp.Plan.Errors,
		ExecutionStatus: sp.ExecutionStatus,
		CancelRequested: sp.CancelRequested,
		ExecutionResult: sp.ExecutionResult,
		ExecutionError:  sp.ExecutionError,
		ExecutionProgress: ExecutionProgress{
			ProcessedItems:        sp.ProcessedItems,
			TotalItems:            sp.TotalItems,
			CurrentItemSourcePath: sp.CurrentItemSourcePath,
			StartedAt:             sp.StartedAt,
			FinishedAt:            sp.FinishedAt,
		},
	}
}

// resumeInterruptedExecution 只恢复保存为 running 的计划，其他状态保持原样。
// 缺少执行用户时使用 system，然后在后台从保存的处理位置继续。
// 读取或最终保存失败会记录错误，最终保存失败时保留工作区。
func (is *ImporterService) resumeInterruptedExecution() {
	sp, err := is.planStore.Get()
	if err != nil || sp == nil {
		if err != nil && !errors.Is(err, ErrNoPlan) {
			is.logger.Error("failed to load persisted importer state", "error", err)
		}
		return
	}
	if sp.ExecutionStatus != ExecutionStatusRunning {
		return
	}
	if sp.ExecutionUserID == "" {
		sp.ExecutionUserID = "system"
	}

	go func(snapshot *StoredPlan) {
		res, execErr := is.executeStoredPlan(snapshot)
		if finishErr := is.planStore.FinishExecution(snapshot.Plan.ID, res, execErr); finishErr != nil {
			is.logger.Error("failed to persist resumed import completion", "error", finishErr)
			return
		}
		is.cleanupWorkspace(snapshot.WorkspaceRoot)
	}(sp)
}
