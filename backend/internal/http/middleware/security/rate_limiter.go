package security

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// KeyedLimiter 按字符串键保存请求发生时间，只计入当前滑动时间窗口内的记录。
// 互斥锁保护所有键的切片及删除操作，没有后台清理任务，清理由 Allow 触发。
// 调用方可直接限制部分请求，也可使用 NewRateLimiter 将整个路由处理链包起来。
type KeyedLimiter struct {
	mu             sync.Mutex
	hits           map[string][]time.Time
	limit          int
	window         time.Duration
	resetOnSuccess bool
}

// NewKeyedLimiter 保存每个键在 window 内允许登记的次数上限。
// resetOnSuccess 开启时，成功通知立即删除该键的全部历史，并非只扣除一次。
// 参数按传入值使用，这里不检查次数和窗口是否为正数。
func NewKeyedLimiter(limit int, window time.Duration, resetOnSuccess bool) *KeyedLimiter {
	return &KeyedLimiter{
		hits:           make(map[string][]time.Time),
		limit:          limit,
		window:         window,
		resetOnSuccess: resetOnSuccess,
	}
}

// cleanup 在锁内遍历全部键，原地压紧仍严格晚于窗口起点的时间记录。
// 恰好位于起点的记录也被丢弃，没有有效记录的键从映射中删除。
func (rl *KeyedLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)

	for key, events := range rl.hits {
		n := 0
		for _, t := range events {
			if t.After(cutoff) {
				// 有效记录向切片前部移动，复用现有存储空间。
				events[n] = t
				n++
			}
		}

		if n == 0 {
			// 无有效记录时释放映射条目，避免长期保留不再请求的键。
			delete(rl.hits, key)
		} else {
			// 只暴露压紧后的有效区间，后续计数不包含过期记录。
			rl.hits[key] = events[:n]
		}
	}
}

// Allow 清理历史后判断当前键是否达到上限，允许时登记本次时间并返回 true。
// 达到上限返回 false，不将被拒绝的请求加入历史，因此拒绝本身不延长窗口。
func (rl *KeyedLimiter) Allow(key string) bool {
	rl.cleanup()

	// 全局清理与本次判断分别加锁，这里按较晚的当前时间再次筛选目标键。
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	var events []time.Time
	if old, ok := rl.hits[key]; ok {
		events = old[:0]
		for _, t := range old {
			if t.After(cutoff) {
				events = append(events, t)
			}
		}
	}
	if len(events) >= rl.limit {
		rl.hits[key] = events
		return false
	}

	// 上限判断与登记在同一临界区内，其他并发请求不能越过登记步骤抢占计数。
	events = append(events, now)
	rl.hits[key] = events
	return true
}

// NotifyResult 在启用成功清空且 success 为 true 时删除该键的全部请求记录。
// 失败结果或关闭成功清空时不改变历史，也不登记额外请求。
func (rl *KeyedLimiter) NotifyResult(key string, success bool) {
	if !rl.resetOnSuccess || !success {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.hits, key)
}

// ClientKey 从连接的 RemoteAddr 提取主机部分，解析失败则使用完整原值。
// 这里不读取代理转发头，也不调用 Gin 的 ClientIP，因此代理后的请求按对端计数。
func ClientKey(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}

// NewRateLimiter 将按连接主机计数的限制器包装成 Gin 中间件。
// 超限时返回 429 并终止处理链，允许时等待后续处理完成，再按响应状态通知结果。
// 仅 200 至 299 的响应视为成功，是否清空历史仍遵循 resetOnSuccess。
func NewRateLimiter(limit int, window time.Duration, resetOnSuccess bool) gin.HandlerFunc {
	rl := NewKeyedLimiter(limit, window, resetOnSuccess)

	return func(c *gin.Context) {
		key := ClientKey(c)
		if !rl.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests, please try again later",
			})
			return
		}

		c.Next()

		status := c.Writer.Status()
		rl.NotifyResult(key, status >= 200 && status < 300)
	}
}
