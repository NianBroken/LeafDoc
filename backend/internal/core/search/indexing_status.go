// 本文件提供索引任务的内存状态计数器，读写方法通过互斥锁保护字段。
package search

import (
	"sync"
	"time"
)

// IndexingStatus 的公开字段用于序列化，调用方应使用 Snapshot 获取一致快照。
// 直接读写公开字段不会自动取得锁，也不在方法中验证任务生命周期是否合法。
type IndexingStatus struct {
	mu         sync.RWMutex
	Active     bool      `json:"active"`      // 是否标记为索引运行中。
	Indexed    int       `json:"indexed"`     // Success 方法累计的成功次数。
	Failed     int       `json:"failed"`      // Fail 方法累计的失败次数。
	FinishedAt time.Time `json:"finished_at"` // 最近一次 Finish 写入的完成时间。
}

// NewIndexingStatus 创建未运行也未完成的初始状态，初始 IsReady 返回 false。
func NewIndexingStatus() *IndexingStatus {
	return &IndexingStatus{
		Active:  false,
		Indexed: 0,
		Failed:  0,
	}
}

// Start 开始一轮计数，清空旧成功数、失败数和完成时间。
func (s *IndexingStatus) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Active = true
	s.Indexed = 0
	s.Failed = 0
	s.FinishedAt = time.Time{} // 运行中的任务不携带前一轮完成时间。
}

// Finish 标记不再运行并记录当前时间，不改变两项计数。
func (s *IndexingStatus) Finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Active = false
	s.FinishedAt = time.Now()
}

// Success 累加成功数，不检查 Active，因此由调用方保证只登记对应任务结果。
func (s *IndexingStatus) Success() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Indexed++
}

// Fail 累加失败数，不自动结束任务。
func (s *IndexingStatus) Fail() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Failed++
}

// IsActive 持读锁返回运行标志。
func (s *IndexingStatus) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Active
}

// IsFailed 要求任务已结束、完成时间已记录且失败数大于零。
func (s *IndexingStatus) IsFailed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.Active && s.Failed > 0 && !s.FinishedAt.IsZero()
}

// IsReady 要求任务已结束、完成时间非零且失败数为零，不要求成功数大于零。
func (s *IndexingStatus) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.Active && s.Failed == 0 && !s.FinishedAt.IsZero()
}

// Snapshot 在同一读锁内复制全部状态到新对象，返回对象使用独立零值锁。
func (s *IndexingStatus) Snapshot() *IndexingStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &IndexingStatus{
		Active:     s.Active,
		Indexed:    s.Indexed,
		Failed:     s.Failed,
		FinishedAt: s.FinishedAt,
	}
}
