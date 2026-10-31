// 本文件保存后台索引重建任务的共享状态。状态对象负责互斥与快照，
// 页面树和索引的实际处理由组装层提供的任务回调执行。
package wikiresync

import "sync"

// Phase 是供轮询接口返回的阶段名称，调用方在进入对应处理步骤时写入。
type Phase string

const (
	PhaseTree   Phase = "tree"
	PhaseLinks  Phase = "links"
	PhaseTags   Phase = "tags"
	PhaseSearch Phase = "search"
)

// JobStatus 是一次加读锁取得的状态副本。空阶段和空错误不写入 JSON，
// running 与 done 始终保留，使客户端能够区分空闲、执行中和已结束。
type JobStatus struct {
	Running bool   `json:"running"`
	Phase   string `json:"phase,omitempty"`
	Done    bool   `json:"done"`
	Error   string `json:"error,omitempty"`
}

// ResyncJob 用同一把读写锁保护运行标记、阶段与结束结果。
// Start 的检查和占用在写锁内完成，并发触发只有一个调用能够成功。
type ResyncJob struct {
	mu      sync.RWMutex
	running bool
	phase   Phase
	done    bool
	err     error
}

// NewResyncJob 返回空闲状态，尚未开始也尚未结束，阶段和错误均为空。
func NewResyncJob() *ResyncJob {
	return &ResyncJob{}
}

// Start 在没有任务运行时占用任务并清空上一轮的结束标记、阶段和错误。
// 已经运行时返回 false，状态保持不变，调用方据此拒绝重复启动。
func (j *ResyncJob) Start() bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.running {
		return false
	}
	j.running = true
	j.done = false
	j.phase = ""
	j.err = nil
	return true
}

// SetPhase 在写锁内替换阶段，不检查任务是否正在运行。执行步骤和调用时机
// 由工作回调负责，轮询接口能够在并发读取时取得完整的阶段值。
func (j *ResyncJob) SetPhase(p Phase) {
	j.mu.Lock()
	j.phase = p
	j.mu.Unlock()
}

// Finish 释放运行占用、设置结束标记并保存错误，保留结束时的阶段。
// 错误为 nil 表示没有错误文字；失败时客户端仍能看到停留的阶段。
func (j *ResyncJob) Finish(err error) {
	j.mu.Lock()
	j.running = false
	j.done = true
	j.err = err
	j.mu.Unlock()
}

// Status 在读锁内复制各字段，将保存的错误转换为接口需要的文字。
// 返回值不暴露内部字段，调用方读取副本不会改变任务状态。
func (j *ResyncJob) Status() JobStatus {
	j.mu.RLock()
	defer j.mu.RUnlock()

	s := JobStatus{
		Running: j.running,
		Phase:   string(j.phase),
		Done:    j.done,
	}
	if j.err != nil {
		s.Error = j.err.Error()
	}
	return s
}
