// 本文件保存快照任务的运行状态及最近结果，提供持锁读取的接口快照。
package snapshot

import (
	"sync"
	"time"
)

// Status 通过方法保护状态读写。运行标记覆盖归档及保留清理阶段，
// 最近成功时间、主任务错误和保留清理错误分别保存。
type Status struct {
	mu             sync.RWMutex
	IsRunning      bool
	LastSnapshotAt time.Time
	LastError      string
	LastPruneError string
}

// TryStart 在锁内检查并设置运行标记，已有任务时返回 false。
// 开始任务不会清空此前的时间和错误，运行期间仍可查询最近完成结果。
func (s *Status) TryStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsRunning {
		return false
	}
	s.IsRunning = true
	return true
}

// SetSuccess 结束运行，记录成功时间，清空主任务错误，并保存保留清理错误文本。
// 管理器在清理结束后调用，使清理期间的 TryStart 仍被拒绝。
func (s *Status) SetSuccess(t time.Time, pruneErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsRunning = false
	s.LastSnapshotAt = t
	s.LastError = ""
	s.LastPruneError = pruneErr
}

// SetError 结束运行并保存主任务错误，保留此前成功时间及保留清理错误。
func (s *Status) SetError(err string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsRunning = false
	s.LastError = err
}

// Snapshot 在读锁内复制全部状态，非零成功时间复制到单独对象，
// 避免返回指针直接引用内部时间字段。零时间以空指针表示。
func (s *Status) Snapshot() StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var lastSnapshotAt *time.Time
	if !s.LastSnapshotAt.IsZero() {
		t := s.LastSnapshotAt
		lastSnapshotAt = &t
	}
	return StatusSnapshot{
		IsRunning:      s.IsRunning,
		LastSnapshotAt: lastSnapshotAt,
		LastError:      s.LastError,
		LastPruneError: s.LastPruneError,
	}
}

// StatusSnapshot 是可序列化的状态值，尚无成功时间和空错误字段从 JSON 中省略。
type StatusSnapshot struct {
	IsRunning      bool       `json:"isRunning"`
	LastSnapshotAt *time.Time `json:"lastSnapshotAt,omitempty"`
	LastError      string     `json:"lastError,omitempty"`
	LastPruneError string     `json:"lastPruneError,omitempty"`
}
