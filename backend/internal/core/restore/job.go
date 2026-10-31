// 本文件记录恢复任务的运行、阶段和结果，向进度接口提供不含凭据的状态副本。
package restore

import (
	"errors"
	"sync"
)

// Phase 是供进度展示使用的阶段标识，设置阶段不会自行执行对应业务动作。
type Phase string

const (
	PhaseValidating           Phase = "validating"
	PhaseSwapping             Phase = "swapping"
	PhaseReopeningAuth        Phase = "reopening_auth"
	PhaseInvalidatingSessions Phase = "invalidating_sessions"
	PhaseReloadingBranding    Phase = "reloading_branding"
)

// JobStatus 是状态接口的值副本。空阶段、错误、版本提示和未介入标记在 JSON 中省略。
type JobStatus struct {
	Running           bool   `json:"running"`
	Phase             string `json:"phase,omitempty"`
	Done              bool   `json:"done"`
	Error             string `json:"error,omitempty"`
	VersionWarning    string `json:"versionWarning,omitempty"`
	NeedsIntervention bool   `json:"needsIntervention,omitempty"`
}

// Job 用读写锁保存单个恢复任务状态。NeedsIntervention 由专门结束入口设置，
// 不在本对象中判定失败原因；恢复中的回滚、重载失败或 panic 均由调用方选择结束方式。
type Job struct {
	mu                sync.RWMutex
	running           bool
	phase             Phase
	done              bool
	err               error
	versionWarning    string
	needsIntervention bool
}

// NewJob 创建尚未运行、尚未完成且没有错误的状态对象。
func NewJob() *Job {
	return &Job{}
}

// Start 在没有运行任务时清空此前结果并标记开始，已有任务则返回 false。
// 此入口自身不因 NeedsIntervention 拒绝开始，管理器在调用前检查该限制。
func (j *Job) Start() bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.running {
		return false
	}
	j.running = true
	j.done = false
	j.phase = ""
	j.err = nil
	j.versionWarning = ""
	j.needsIntervention = false
	return true
}

// SetPhase 持锁写入阶段，不验证任务是否运行或阶段之间的顺序。
func (j *Job) SetPhase(p Phase) {
	j.mu.Lock()
	j.phase = p
	j.mu.Unlock()
}

// SetVersionWarning 保存非致命提示文本，不改变运行、完成或错误状态。
func (j *Job) SetVersionWarning(msg string) {
	j.mu.Lock()
	j.versionWarning = msg
	j.mu.Unlock()
}

// Finish 标记任务结束并保存传入错误，保留阶段、版本提示与已有介入标记。
// 介入标记不是在这里清空，而是由下一次成功 Start 清空。
func (j *Job) Finish(err error) {
	j.mu.Lock()
	j.running = false
	j.done = true
	j.err = err
	j.mu.Unlock()
}

// FinishNeedsIntervention 结束任务并置介入标记，供管理器阻止继续启动恢复。
// 它不自行回滚、操作写入屏障或重启进程。
func (j *Job) FinishNeedsIntervention(err error) {
	j.mu.Lock()
	j.running = false
	j.done = true
	j.err = err
	j.needsIntervention = true
	j.mu.Unlock()
}

// Status 在读锁内复制字段，错误只转换成展示文本，不返回内部错误对象。
func (j *Job) Status() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()

	s := JobStatus{
		Running:           j.running,
		Phase:             string(j.phase),
		Done:              j.done,
		VersionWarning:    j.versionWarning,
		NeedsIntervention: j.needsIntervention,
	}
	if j.err != nil {
		s.Error = j.err.Error()
	}
	return s
}

// recordCleanupError 把清理失败合并到现有错误，保留运行和介入状态。
func (j *Job) recordCleanupError(err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.err = errors.Join(j.err, err)
}
