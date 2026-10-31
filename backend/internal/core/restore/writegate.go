// 本文件提供与 HTTP 无关的写入准入计数。中间件在可变更请求开始时登记，
// 维护流程关闭准入后等待已经登记的请求结束，再处理数据库和资源目录。
package restore

import (
	"sync"
	"sync/atomic"
	"time"
)

// WriteGate 用互斥锁把开关检查与登记计数连为一个操作，避免关门之后仍登记新请求。
// 它只约束调用 TryEnter 的操作，不自动暂停所有后台任务或直接数据库写入。
type WriteGate struct {
	mu      sync.Mutex
	engaged bool
	// 原子整数支持离开函数在锁外递减，也提供所需的六十四位对齐。
	inflight atomic.Int64
}

// NewWriteGate 创建默认开放且没有在途请求的屏障。
func NewWriteGate() *WriteGate {
	return &WriteGate{}
}

// Engage 关闭新请求准入，不取消已经登记的请求，也不等待它们结束。
func (g *WriteGate) Engage() {
	g.mu.Lock()
	g.engaged = true
	g.mu.Unlock()
}

// Disengage 重新允许登记，不改变仍在执行的请求计数。
func (g *WriteGate) Disengage() {
	g.mu.Lock()
	g.engaged = false
	g.mu.Unlock()
}

// Engaged 在锁内读取当前准入开关，不表示已有请求是否全部结束。
func (g *WriteGate) Engaged() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.engaged
}

// TryEnter 在屏障开放时递增计数并返回离开函数，关闭时返回空函数及 false。
// 成功调用方必须恰好执行一次 leave，本函数不替重复调用提供幂等保护。
func (g *WriteGate) TryEnter() (leave func(), ok bool) {
	g.mu.Lock()
	if g.engaged {
		g.mu.Unlock()
		return nil, false
	}
	g.inflight.Add(1)
	g.mu.Unlock()
	return func() { g.inflight.Add(-1) }, true
}

// WaitForDrain 每二十毫秒检查在途计数，等到零返回 true，超过截止时间返回 false。
// 它不自行 Engage，调用方先关闭准入才能阻止等待期间登记新请求。
// 恢复管理器在超时时重新开放屏障并结束任务，不继续交换业务资源。
func (g *WriteGate) WaitForDrain(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if g.inflight.Load() == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}
