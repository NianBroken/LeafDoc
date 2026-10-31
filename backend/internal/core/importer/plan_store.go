package importer

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// errWrapFmt 保留状态不可用错误的可识别身份，底层原因以文字附在后面。
const errWrapFmt = "%w: %v"

// ErrNoPlan 表示状态库中没有当前导入计划。
var ErrNoPlan = errors.New("no plan available")

// ErrImportExecutionRunning 表示当前计划正在执行，不接受冲突操作。
var ErrImportExecutionRunning = errors.New("import execution already running")

// ErrImportCanceled 表示执行器在计划项边界观察到取消请求。
var ErrImportCanceled = errors.New("import execution canceled")

// ErrImportStateUnavailable 标识计划状态读取或保存失败造成的不可用状态。
var ErrImportStateUnavailable = errors.New("import state unavailable")

// ExecutionStatus 是计划保存的生命周期状态，用于控制启动、取消和恢复执行。
type ExecutionStatus string

const (
	ExecutionStatusPlanned   ExecutionStatus = "planned"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCanceled  ExecutionStatus = "canceled"
)

// StoredPlan 将计划、解压目录和执行进度保存在同一个状态对象中。
// 运行用户用于重启后继续执行，取消标记由执行器在每项开始前读取。
type StoredPlan struct {
	Plan            *PlanResult
	PlanOptions     PlanOptions
	WorkspaceRoot   string
	CreatedAt       time.Time
	ExecutionStatus ExecutionStatus
	ExecutionUserID string
	CancelRequested bool
	ExecutionResult *ExecutionResult
	ExecutionError  *string
	ExecutionProgress
}

// PlanStore 以读写锁保护单份当前计划，并按可选状态键将 JSON 保存到数据库。
// stateFile 仅用于拆出数据库目录和键名，不表示直接写入该路径上的文件。
// 读写失败会保存在 stateErr，后续主要操作返回该错误，不自动清除。
type PlanStore struct {
	mu        sync.RWMutex
	plan      *StoredPlan
	stateFile string
	stateErr  error
}

// NewPlanStore 使用首个可选参数加载状态，未传入或传入空串时仅在内存中保存。
// 构造不返回错误，加载失败由后续 Get、Set 等方法通过 stateErr 报告。
func NewPlanStore(stateFile ...string) *PlanStore {
	ps := &PlanStore{}
	if len(stateFile) > 0 {
		ps.stateFile = stateFile[0]
		if err := ps.load(); err != nil {
			ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		}
	}
	return ps
}

// Set 在写锁内替换当前对象并持久化，传入对象直接保存而不复制。
// 持久化失败不恢复先前对象，并将状态库置为不可用。
func (ps *PlanStore) Set(sp *StoredPlan) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return ps.stateErr
	}
	ps.plan = sp
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return ps.stateErr
	}
	return nil
}

// Get 在读锁内取得当前计划副本，没有计划与状态不可用分别返回对应错误。
// 副本隔离外层状态和结果项切片，不深复制计划指针及项内的引用字段。
func (ps *PlanStore) Get() (*StoredPlan, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if ps.stateErr != nil {
		return nil, ps.stateErr
	}
	if ps.plan == nil {
		return nil, ErrNoPlan
	}
	return cloneStoredPlan(ps.plan), nil
}

// Clear 清空当前引用并删除持久化记录，返回原对象供调用方清理工作区。
// 删除失败时内存引用仍为空，同时返回原对象和不可用错误。
func (ps *PlanStore) Clear() (*StoredPlan, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return nil, ps.stateErr
	}
	old := ps.plan
	ps.plan = nil
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return old, ps.stateErr
	}
	return old, nil
}

// TryStartExecution 在锁内检查并占用执行状态，阻止同一计划重复并发启动。
// 运行中或已完成时返回状态副本与 false，其余状态从零开始并清空累计结果。
// 启动信息必须先持久化成功，调用方才会收到 true。
func (ps *PlanStore) TryStartExecution(userID string) (*StoredPlan, bool, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return nil, false, ps.stateErr
	}

	if ps.plan == nil {
		return nil, false, ErrNoPlan
	}
	if ps.plan.Plan == nil {
		return nil, false, ErrImportStateUnavailable
	}

	switch ps.plan.ExecutionStatus {
	case ExecutionStatusRunning, ExecutionStatusCompleted:
		return cloneStoredPlan(ps.plan), false, nil
	}

	ps.plan.ExecutionStatus = ExecutionStatusRunning
	ps.plan.ExecutionUserID = userID
	ps.plan.CancelRequested = false
	ps.plan.ExecutionResult = nil
	ps.plan.ExecutionError = nil
	now := time.Now()
	ps.plan.ExecutionProgress = ExecutionProgress{
		ProcessedItems: 0,
		TotalItems:     len(ps.plan.Plan.Items),
		StartedAt:      &now,
	}
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return nil, false, ps.stateErr
	}

	return cloneStoredPlan(ps.plan), true, nil
}

// FinishExecution 只结束标识匹配的当前计划，过期执行结果不会覆盖其他计划。
// 取消单独标记为 canceled 且不保留错误文字，其他错误标记为 failed。
// 无错误时标记 completed 并令处理数等于总数，执行结果对象直接保存。
func (ps *PlanStore) FinishExecution(planID string, result *ExecutionResult, execErr error) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return ps.stateErr
	}

	if ps.plan == nil || ps.plan.Plan == nil || ps.plan.Plan.ID != planID {
		return nil
	}

	ps.plan.ExecutionResult = result
	ps.plan.ExecutionUserID = ""
	ps.plan.CancelRequested = false

	if execErr != nil {
		finishedAt := time.Now()
		ps.plan.FinishedAt = &finishedAt
		ps.plan.CurrentItemSourcePath = nil
		if errors.Is(execErr, ErrImportCanceled) {
			ps.plan.ExecutionStatus = ExecutionStatusCanceled
			ps.plan.ExecutionError = nil
			if err := ps.persistLocked(); err != nil {
				ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
				return ps.stateErr
			}
			return nil
		}
		errMsg := execErr.Error()
		ps.plan.ExecutionStatus = ExecutionStatusFailed
		ps.plan.ExecutionError = &errMsg
		if err := ps.persistLocked(); err != nil {
			ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
			return ps.stateErr
		}
		return nil
	}

	ps.plan.ExecutionStatus = ExecutionStatusCompleted
	ps.plan.ExecutionError = nil
	finishedAt := time.Now()
	ps.plan.FinishedAt = &finishedAt
	ps.plan.ProcessedItems = ps.plan.TotalItems
	ps.plan.CurrentItemSourcePath = nil
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return ps.stateErr
	}
	return nil
}

// UpdateExecutionProgress 写入匹配计划的处理数和当前路径，并持久化整个状态。
// 时间字段只有传入非空指针时才替换，部分结果存在时复制外层结构与结果项切片。
func (ps *PlanStore) UpdateExecutionProgress(planID string, progress ExecutionProgress, partialResult *ExecutionResult) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return ps.stateErr
	}

	if ps.plan == nil || ps.plan.Plan == nil || ps.plan.Plan.ID != planID {
		return nil
	}

	ps.plan.ProcessedItems = progress.ProcessedItems
	ps.plan.TotalItems = progress.TotalItems
	ps.plan.CurrentItemSourcePath = progress.CurrentItemSourcePath
	if progress.StartedAt != nil {
		ps.plan.StartedAt = progress.StartedAt
	}
	if progress.FinishedAt != nil {
		ps.plan.FinishedAt = progress.FinishedAt
	}
	if partialResult != nil {
		ps.plan.ExecutionResult = cloneExecutionResult(partialResult)
	}
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return ps.stateErr
	}
	return nil
}

// RequestCancel 只为运行中且尚未请求取消的计划设置标记，成功设置返回 true。
// 此方法不停止协程或当前 I/O，真正退出发生在执行器下一次取消检查。
func (ps *PlanStore) RequestCancel() (*StoredPlan, bool, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stateErr != nil {
		return nil, false, ps.stateErr
	}

	if ps.plan == nil {
		return nil, false, ErrNoPlan
	}
	if ps.plan.ExecutionStatus != ExecutionStatusRunning {
		return cloneStoredPlan(ps.plan), false, nil
	}
	if ps.plan.CancelRequested {
		return cloneStoredPlan(ps.plan), false, nil
	}

	ps.plan.CancelRequested = true
	if err := ps.persistLocked(); err != nil {
		ps.stateErr = fmt.Errorf(errWrapFmt, ErrImportStateUnavailable, err)
		return nil, false, ps.stateErr
	}
	return cloneStoredPlan(ps.plan), true, nil
}

// IsCancelRequested 在读锁内核对计划标识及取消标记，不读取持久化错误字段。
func (ps *PlanStore) IsCancelRequested(planID string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.plan != nil &&
		ps.plan.Plan != nil &&
		ps.plan.Plan.ID == planID &&
		ps.plan.CancelRequested
}

// load 根据状态路径拆出的目录和键读取 import_state 记录，并解码为当前计划。
// 数据库没有记录表示没有计划，不作为加载失败处理。
func (ps *PlanStore) load() error {
	if ps.stateFile == "" {
		return nil
	}

	raw, err := database.ReadRecord(filepath.Dir(ps.stateFile), "import_state", filepath.Base(ps.stateFile))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	var sp StoredPlan
	if err := json.Unmarshal(raw, &sp); err != nil {
		return err
	}

	ps.plan = &sp
	return nil
}

// persistLocked 在调用方持有写锁时保存整个计划，空状态键表示仅使用内存。
// 空计划删除数据库记录，其余计划编码成 JSON 后写入 import_state。
func (ps *PlanStore) persistLocked() error {
	if ps.stateFile == "" {
		return nil
	}
	directory, key := filepath.Dir(ps.stateFile), filepath.Base(ps.stateFile)
	if ps.plan == nil {
		return database.DeleteRecord(directory, "import_state", key)
	}
	raw, err := json.Marshal(ps.plan)
	if err != nil {
		return err
	}
	return database.WriteRecord(directory, "import_state", key, raw)
}

// cloneStoredPlan 复制外层状态并单独复制执行结果，Plan 和时间等指针保持共享。
func cloneStoredPlan(sp *StoredPlan) *StoredPlan {
	if sp == nil {
		return nil
	}

	clone := *sp
	clone.ExecutionResult = cloneExecutionResult(sp.ExecutionResult)
	return &clone
}

// cloneExecutionResult 复制结果结构和 Items 切片，避免后续追加直接共享切片存储。
// 项内的 Notes 切片与 Error 指针仍为浅复制，不构成完全独立的深层对象。
func cloneExecutionResult(res *ExecutionResult) *ExecutionResult {
	if res == nil {
		return nil
	}

	clone := *res
	if res.Items != nil {
		clone.Items = append([]ExecutionItemResult(nil), res.Items...)
	}
	return &clone
}
